package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
)

func (s *examQuestionGroupDraftService) StartExtraction(
	ctx context.Context,
	tenantID uint64,
	userID string,
	taskID string,
	force bool,
) (*types.ExamQuestionGroupDraftExtractionResult, error) {
	task, _, err := s.prepareExtraction(ctx, tenantID, userID, taskID, force)
	if err != nil {
		return nil, err
	}
	if s.taskEnqueuer == nil {
		return nil, errors.New("exam question group task enqueuer is not configured")
	}
	now := time.Now()
	task.Progress = types.ExamStructuringProgress{
		Phase:     types.ExamStructuringPhaseQueued,
		Percent:   0,
		Message:   "题组结构化任务已进入队列",
		Warnings:  []types.ExamStructuringWarning{},
		StartedAt: &now,
	}
	if err := s.persistExtractionProgress(ctx, task, types.ExamStructuringTaskStatusExtracting, ""); err != nil {
		return nil, err
	}
	payload := types.ExamQuestionGroupExtractionPayload{
		TenantID: tenantID, UserID: userID, TaskID: task.ID, Force: force,
	}
	if err := s.enqueueQuestionGroupExtraction(payload); err != nil {
		s.failQuestionGroupExtraction(ctx, task, err)
		return nil, err
	}
	return &types.ExamQuestionGroupDraftExtractionResult{Task: task, Drafts: []*types.ExamQuestionGroupDraft{}}, nil
}

func (s *examQuestionGroupDraftService) enqueueQuestionGroupExtraction(payload types.ExamQuestionGroupExtractionPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(types.TypeExamQuestionGroupExtraction, data)
	_, err = s.taskEnqueuer.Enqueue(task, asynq.Queue(types.QueueQuestion), asynq.MaxRetry(0))
	return err
}

func (s *examQuestionGroupDraftService) ProcessExtractionTask(ctx context.Context, asynqTask *asynq.Task) (err error) {
	var payload types.ExamQuestionGroupExtractionPayload
	if err = json.Unmarshal(asynqTask.Payload(), &payload); err != nil {
		return fmt.Errorf("decode exam question group extraction payload: %w", err)
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)
	ctx = context.WithValue(ctx, types.UserIDContextKey, payload.UserID)
	var task *types.ExamStructuringTask
	defer func() {
		if recovered := recover(); recovered != nil {
			panicErr := fmt.Errorf("exam question group extraction panic: %v", recovered)
			err = s.failQuestionGroupExtraction(ctx, task, panicErr)
		}
	}()

	task, material, err := s.loadQueuedExtraction(ctx, payload)
	if err != nil {
		if task != nil {
			return s.failQuestionGroupExtraction(ctx, task, err)
		}
		return err
	}
	_, err = s.runQuestionGroupExtraction(ctx, payload.TenantID, task, material, payload.Force)
	return err
}

func (s *examQuestionGroupDraftService) loadQueuedExtraction(
	ctx context.Context,
	payload types.ExamQuestionGroupExtractionPayload,
) (*types.ExamStructuringTask, *types.ExamMaterial, error) {
	task, err := s.materialRepo.GetStructuringTaskByIDAndTenant(ctx, payload.TaskID, payload.TenantID)
	if err != nil {
		return nil, nil, err
	}
	if task.Status != types.ExamStructuringTaskStatusExtracting {
		return task, nil, ErrExamInvalidRequest
	}
	material, err := s.materialRepo.GetMaterialByIDAndTenant(ctx, task.MaterialID, payload.TenantID)
	if err != nil {
		return task, nil, err
	}
	if material.IngestStatus != types.ExamMaterialIngestStatusCompleted {
		return task, nil, ErrExamInvalidRequest
	}
	return task, material, nil
}

func (s *examQuestionGroupDraftService) runQuestionGroupExtraction(
	ctx context.Context,
	tenantID uint64,
	task *types.ExamStructuringTask,
	material *types.ExamMaterial,
	force bool,
) (*types.ExamQuestionGroupDraftExtractionResult, error) {
	if force {
		if err := s.draftRepo.DeleteDraftsByTask(ctx, tenantID, task.ID); err != nil {
			return nil, s.failQuestionGroupExtraction(ctx, task, err)
		}
	}
	if task.Progress.StartedAt == nil {
		now := time.Now()
		task.Progress.StartedAt = &now
	}
	chunks, err := s.loadExtractionChunks(ctx, task, material)
	if err != nil {
		return nil, err
	}
	candidates, rawOutput, err := s.extractQuestionGroups(ctx, tenantID, task, material, chunks)
	if err != nil {
		return nil, s.failQuestionGroupExtraction(ctx, task, err)
	}
	return s.persistQuestionGroupDrafts(ctx, tenantID, task, material, candidates, rawOutput)
}

func (s *examQuestionGroupDraftService) loadExtractionChunks(
	ctx context.Context,
	task *types.ExamStructuringTask,
	material *types.ExamMaterial,
) ([]*types.Chunk, error) {
	chunks, err := s.chunkReader.ListChunksByKnowledgeID(ctx, material.KnowledgeID)
	if err != nil {
		return nil, s.failQuestionGroupExtraction(ctx, task, err)
	}
	if len(chunks) == 0 {
		return nil, s.failQuestionGroupExtraction(ctx, task, errors.New("exam material has no available chunks"))
	}
	task.Progress.Phase = types.ExamStructuringPhasePreflight
	task.Progress.Percent = 5
	task.Progress.Message = "正在检查公式与图形资源"
	task.Progress.Warnings = preflightQuestionGroupChunks(chunks)
	if err := s.persistExtractionProgress(ctx, task, types.ExamStructuringTaskStatusExtracting, ""); err != nil {
		return nil, err
	}
	return chunks, nil
}

func (s *examQuestionGroupDraftService) extractQuestionGroups(
	ctx context.Context,
	tenantID uint64,
	task *types.ExamStructuringTask,
	material *types.ExamMaterial,
	chunks []*types.Chunk,
) ([]*types.ExamQuestionGroupDraftCandidate, string, error) {
	extractCtx := context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	observer := func(progress interfaces.ExamQuestionGroupBatchProgress) error {
		return s.updateBatchProgress(ctx, task, progress)
	}
	if extractor, ok := s.extractor.(interfaces.ExamQuestionGroupProgressExtractor); ok {
		return extractor.ExtractWithProgress(extractCtx, material, task, chunks, observer)
	}
	if err := observer(interfaces.ExamQuestionGroupBatchProgress{Total: 1, Current: 1}); err != nil {
		return nil, "", err
	}
	candidates, raw, err := s.extractor.Extract(extractCtx, material, task, chunks)
	if err == nil {
		err = observer(interfaces.ExamQuestionGroupBatchProgress{Total: 1, Current: 1, Completed: 1})
	}
	return candidates, raw, err
}

func (s *examQuestionGroupDraftService) updateBatchProgress(
	ctx context.Context,
	task *types.ExamStructuringTask,
	progress interfaces.ExamQuestionGroupBatchProgress,
) error {
	task.Progress.Phase = types.ExamStructuringPhaseExtracting
	task.Progress.TotalBatches = progress.Total
	task.Progress.CurrentBatch = progress.Current
	task.Progress.CompletedBatches = progress.Completed
	task.Progress.Percent = 10
	if progress.Total > 0 {
		task.Progress.Percent += progress.Completed * 75 / progress.Total
	}
	task.Progress.Message = fmt.Sprintf("正在抽取第 %d/%d 批", progress.Current, progress.Total)
	return s.persistExtractionProgress(ctx, task, types.ExamStructuringTaskStatusExtracting, "")
}

func (s *examQuestionGroupDraftService) persistQuestionGroupDrafts(
	ctx context.Context,
	tenantID uint64,
	task *types.ExamStructuringTask,
	material *types.ExamMaterial,
	candidates []*types.ExamQuestionGroupDraftCandidate,
	rawOutput string,
) (*types.ExamQuestionGroupDraftExtractionResult, error) {
	drafts, err := buildQuestionGroupDrafts(task, material, candidates, rawOutput)
	if err != nil {
		return nil, s.failQuestionGroupExtraction(ctx, task, err)
	}
	task.Progress.Phase = types.ExamStructuringPhaseQualityCheck
	task.Progress.Percent = 90
	task.Progress.Message = "正在执行数学题目质量检查"
	task.Progress.QualitySummary = summarizeQuestionGroupQuality(drafts)
	if err := s.draftRepo.CreateDrafts(ctx, drafts); err != nil {
		return nil, s.failQuestionGroupExtraction(ctx, task, err)
	}
	now := time.Now()
	task.Progress.Phase = types.ExamStructuringPhaseCompleted
	task.Progress.Percent = 100
	task.Progress.Message = "题组抽取完成，等待人工校对"
	task.Progress.FinishedAt = &now
	task.StructuredQuestionCount = len(drafts)
	if err := s.persistExtractionProgress(ctx, task, types.ExamStructuringTaskStatusReviewing, ""); err != nil {
		return nil, err
	}
	stats, err := s.draftRepo.CountDraftsByTask(ctx, tenantID, task.ID)
	if err != nil {
		return nil, err
	}
	return &types.ExamQuestionGroupDraftExtractionResult{Task: task, Drafts: drafts, Stats: stats}, nil
}

func (s *examQuestionGroupDraftService) persistExtractionProgress(
	ctx context.Context,
	task *types.ExamStructuringTask,
	status types.ExamStructuringTaskStatus,
	errorMessage string,
) error {
	task.Status = status
	task.ErrorMessage = errorMessage
	task.UpdatedAt = time.Now()
	return s.materialRepo.UpdateStructuringTask(context.WithoutCancel(ctx), task)
}

func (s *examQuestionGroupDraftService) failQuestionGroupExtraction(ctx context.Context, task *types.ExamStructuringTask, err error) error {
	if task != nil {
		now := time.Now()
		task.Progress.Phase = types.ExamStructuringPhaseFailed
		task.Progress.FailedBatch = task.Progress.CurrentBatch
		task.Progress.Message = err.Error()
		task.Progress.FinishedAt = &now
		_ = s.persistExtractionProgress(ctx, task, types.ExamStructuringTaskStatusFailed, err.Error())
	}
	return err
}

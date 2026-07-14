package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type examRAGEvaluationService struct {
	repo            interfaces.ExamRAGEvaluationRepository
	questionService interfaces.ExamQuestionService
	diagnostic      interfaces.ExamRAGDiagnosticService
	tenantService   interfaces.TenantService
	taskEnqueuer    interfaces.TaskEnqueuer
}

type examRAGEvaluationRequestSnapshot struct {
	types.RunExamRAGDiagnosticRequest
	UsedDefaultCases bool `json:"used_default_cases"`
}

func NewExamRAGEvaluationService(
	repo interfaces.ExamRAGEvaluationRepository,
	questionService interfaces.ExamQuestionService,
	diagnostic interfaces.ExamRAGDiagnosticService,
	tenantService interfaces.TenantService,
	taskEnqueuer interfaces.TaskEnqueuer,
) interfaces.ExamRAGEvaluationService {
	return &examRAGEvaluationService{
		repo: repo, questionService: questionService, diagnostic: diagnostic,
		tenantService: tenantService, taskEnqueuer: taskEnqueuer,
	}
}

func (s *examRAGEvaluationService) CreateRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	req *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGEvaluationRun, error) {
	preparation, err := s.diagnostic.PrepareQuestionBank(ctx, tenantID, userID, strings.TrimSpace(bankID), req)
	if err != nil {
		return nil, err
	}
	run, err := newExamRAGEvaluationRun(tenantID, userID, preparation)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	if err := s.enqueueRun(run.ID); err != nil {
		return nil, s.failQueuedRun(ctx, run, err)
	}
	return run, nil
}

func (s *examRAGEvaluationService) ListRuns(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	limit int,
) ([]*types.ExamRAGEvaluationRun, error) {
	if err := s.authorizeBank(ctx, tenantID, userID, bankID); err != nil {
		return nil, err
	}
	runs, err := s.repo.ListRuns(ctx, tenantID, strings.TrimSpace(bankID), limit)
	if err != nil {
		return nil, err
	}
	return compactRAGEvaluationRuns(runs), nil
}

func (s *examRAGEvaluationService) GetRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	if err := s.authorizeBank(ctx, tenantID, userID, bankID); err != nil {
		return nil, err
	}
	return s.repo.GetRun(ctx, tenantID, strings.TrimSpace(bankID), strings.TrimSpace(runID))
}

func (s *examRAGEvaluationService) authorizeBank(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
) error {
	_, err := s.questionService.GetQuestionBank(ctx, tenantID, userID, strings.TrimSpace(bankID))
	return err
}

func newExamRAGEvaluationRun(
	tenantID uint64,
	userID string,
	preparation *types.ExamRAGDiagnosticPreparation,
) (*types.ExamRAGEvaluationRun, error) {
	if preparation == nil || preparation.QuestionBank == nil {
		return nil, ErrExamInvalidRequest
	}
	request, err := json.Marshal(examRAGEvaluationRequestSnapshot{
		RunExamRAGDiagnosticRequest: preparation.Request,
		UsedDefaultCases:            preparation.UsedDefaultCases,
	})
	if err != nil {
		return nil, err
	}
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: len(preparation.Request.Cases)})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &types.ExamRAGEvaluationRun{
		ID: uuid.NewString(), TenantID: tenantID, QuestionBankID: preparation.QuestionBank.ID,
		CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
		Progress: progress, RequestSnapshot: request, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func compactRAGEvaluationRuns(runs []*types.ExamRAGEvaluationRun) []*types.ExamRAGEvaluationRun {
	compacted := make([]*types.ExamRAGEvaluationRun, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		clone := *run
		clone.ResultSnapshot = compactRAGEvaluationResultSnapshot(run.ResultSnapshot)
		compacted = append(compacted, &clone)
	}
	return compacted
}

func compactRAGEvaluationResultSnapshot(snapshot types.JSON) types.JSON {
	if len(snapshot) == 0 {
		return nil
	}
	var result types.ExamRAGDiagnosticResult
	if err := json.Unmarshal(snapshot, &result); err != nil {
		return nil
	}
	result.QuestionBank = nil
	result.KnowledgeBaseIDs = nil
	result.Cases = nil
	result.Summary.Results = nil
	compacted, err := json.Marshal(result)
	if err != nil {
		return nil
	}
	return compacted
}

func (s *examRAGEvaluationService) enqueueRun(runID string) error {
	if s.taskEnqueuer == nil {
		return errors.New("exam RAG evaluation task enqueuer is not configured")
	}
	payload, err := json.Marshal(types.ExamRAGEvaluationTaskPayload{RunID: runID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(types.TypeExamRAGEvaluationRun, payload)
	_, err = s.taskEnqueuer.Enqueue(task, asynq.Queue(types.QueueQuestion), asynq.MaxRetry(0))
	return err
}

func (s *examRAGEvaluationService) failQueuedRun(
	ctx context.Context,
	run *types.ExamRAGEvaluationRun,
	cause error,
) error {
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusFailed
	run.ErrorMessage = cause.Error()
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, run); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

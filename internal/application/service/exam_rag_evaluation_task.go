package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

func (s *examRAGEvaluationService) ProcessRunTask(ctx context.Context, task *asynq.Task) error {
	var payload types.ExamRAGEvaluationTaskPayload
	if task == nil || json.Unmarshal(task.Payload(), &payload) != nil || strings.TrimSpace(payload.RunID) == "" {
		return fmt.Errorf("decode exam RAG evaluation payload: %w", ErrExamInvalidRequest)
	}
	run, err := s.repo.GetRunForTask(ctx, strings.TrimSpace(payload.RunID))
	if err != nil {
		return err
	}
	if isTerminalRAGEvaluationRun(run.Status) {
		return nil
	}
	request, usedDefaultCases, err := parseRAGEvaluationRequest(run.RequestSnapshot)
	if err != nil {
		return s.persistRunFailure(ctx, run, err)
	}
	if err := s.markRunRunning(ctx, run); err != nil {
		return err
	}
	workerCtx, err := s.buildRAGEvaluationWorkerContext(ctx, run)
	if err != nil {
		return s.persistRunFailure(ctx, run, err)
	}
	result, err := s.diagnostic.EvaluateQuestionBank(
		workerCtx, run.TenantID, run.CreatedBy, run.QuestionBankID, request,
	)
	if err != nil {
		return s.persistRunFailure(workerCtx, run, err)
	}
	if result == nil {
		return s.persistRunFailure(workerCtx, run, errors.New("exam RAG evaluation returned empty result"))
	}
	result.UsedDefaultCases = usedDefaultCases
	return s.persistRunCompletion(workerCtx, run, result)
}

func (s *examRAGEvaluationService) buildRAGEvaluationWorkerContext(
	ctx context.Context,
	run *types.ExamRAGEvaluationRun,
) (context.Context, error) {
	if s.tenantService == nil {
		return nil, errors.New("exam RAG evaluation tenant service is not configured")
	}
	tenant, err := s.tenantService.GetTenantByID(ctx, run.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load exam RAG evaluation tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("exam RAG evaluation tenant was not found")
	}
	workerCtx := context.WithValue(ctx, types.TenantIDContextKey, run.TenantID)
	workerCtx = context.WithValue(workerCtx, types.TenantInfoContextKey, tenant)
	workerCtx = context.WithValue(workerCtx, types.UserIDContextKey, run.CreatedBy)
	return workerCtx, nil
}

func parseRAGEvaluationRequest(snapshot types.JSON) (*types.RunExamRAGDiagnosticRequest, bool, error) {
	var requestSnapshot examRAGEvaluationRequestSnapshot
	if err := json.Unmarshal(snapshot, &requestSnapshot); err != nil {
		return nil, false, fmt.Errorf("decode exam RAG evaluation request: %w", err)
	}
	return &requestSnapshot.RunExamRAGDiagnosticRequest, requestSnapshot.UsedDefaultCases, nil
}

func (s *examRAGEvaluationService) markRunRunning(ctx context.Context, run *types.ExamRAGEvaluationRun) error {
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusRunning
	run.ErrorMessage = ""
	if run.StartedAt == nil {
		run.StartedAt = &now
	}
	run.UpdatedAt = now
	return s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, run)
}

func (s *examRAGEvaluationService) persistRunCompletion(
	ctx context.Context,
	run *types.ExamRAGEvaluationRun,
	result *types.ExamRAGDiagnosticResult,
) error {
	snapshot, err := json.Marshal(result)
	if err != nil {
		return s.persistRunFailure(ctx, run, err)
	}
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{
		CompletedCases: result.Summary.Total, TotalCases: result.Summary.Total,
	})
	if err != nil {
		return s.persistRunFailure(ctx, run, err)
	}
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusCompleted
	run.ResultSnapshot = snapshot
	run.Progress = progress
	run.ErrorMessage = ""
	run.CompletedAt = &now
	run.UpdatedAt = now
	return s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, run)
}

func (s *examRAGEvaluationService) persistRunFailure(
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

func isTerminalRAGEvaluationRun(status types.ExamRAGEvaluationRunStatus) bool {
	return status == types.ExamRAGEvaluationRunStatusCompleted || status == types.ExamRAGEvaluationRunStatusFailed
}

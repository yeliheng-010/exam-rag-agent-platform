package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/examagent"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func (s *examAgentEvaluationService) ProcessRunTask(ctx context.Context, task *asynq.Task) error {
	var payload types.ExamAgentEvaluationTaskPayload
	if task == nil || json.Unmarshal(task.Payload(), &payload) != nil || strings.TrimSpace(payload.RunID) == "" {
		return fmt.Errorf("decode exam Agent evaluation payload: %w", ErrExamInvalidRequest)
	}
	run, err := s.repo.GetRunForTask(ctx, types.ExamEvaluationKindAgent, strings.TrimSpace(payload.RunID))
	if err != nil {
		return err
	}
	if isTerminalRAGEvaluationRun(run.Status) {
		return nil
	}
	snapshot, err := parseAgentEvaluationRequest(run.RequestSnapshot)
	if err != nil {
		return s.persistAgentRunFailure(ctx, run, err)
	}
	if s.runner == nil {
		return s.persistAgentRunFailure(ctx, run, errors.New("exam Agent evaluation runner is not configured"))
	}
	if err := s.markAgentRunRunning(ctx, run); err != nil {
		return err
	}
	workerCtx, err := s.buildAgentEvaluationWorkerContext(ctx, run)
	if err != nil {
		return s.persistAgentRunFailure(ctx, run, err)
	}
	result := &types.ExamAgentEvaluationResult{
		QuestionBankID: run.QuestionBankID, Agent: snapshot.Agent,
		Results: make([]types.ExamAgentEvaluationCaseResult, 0, len(snapshot.Cases)),
	}
	for _, scenario := range snapshot.Cases {
		started := time.Now()
		state, executionErr := s.runner.ExecuteAgentEvaluation(
			workerCtx, newAgentEvaluationQARequest(run, snapshot.Agent, scenario), event.NewEventBus(),
		)
		if state == nil && executionErr == nil {
			executionErr = errors.New("empty state")
		}
		caseResult := examagent.EvaluateScenario(scenario, state, executionErr, time.Since(started).Milliseconds())
		result.Results = append(result.Results, caseResult)
		result.Summary = examagent.Summarize(result.Results)
		if err := s.persistAgentRunProgress(workerCtx, run, result, len(result.Results), len(snapshot.Cases)); err != nil {
			return err
		}
	}
	return s.persistAgentRunCompletion(workerCtx, run, result)
}

func parseAgentEvaluationRequest(snapshot types.JSON) (*types.ExamAgentEvaluationRequestSnapshot, error) {
	var request types.ExamAgentEvaluationRequestSnapshot
	if err := json.Unmarshal(snapshot, &request); err != nil {
		return nil, fmt.Errorf("decode exam Agent evaluation request: %w", err)
	}
	if request.Agent.ID == "" || len(request.Cases) == 0 {
		return nil, ErrExamInvalidRequest
	}
	return &request, nil
}

func (s *examAgentEvaluationService) buildAgentEvaluationWorkerContext(
	ctx context.Context, run *types.ExamRAGEvaluationRun,
) (context.Context, error) {
	if s.tenantService == nil {
		return nil, errors.New("exam Agent evaluation tenant service is not configured")
	}
	tenant, err := s.tenantService.GetTenantByID(ctx, run.TenantID)
	if err != nil {
		return nil, fmt.Errorf("load exam Agent evaluation tenant: %w", err)
	}
	if tenant == nil {
		return nil, errors.New("exam Agent evaluation tenant was not found")
	}
	workerCtx := context.WithValue(ctx, types.TenantIDContextKey, run.TenantID)
	workerCtx = context.WithValue(workerCtx, types.TenantInfoContextKey, tenant)
	workerCtx = context.WithValue(workerCtx, types.UserIDContextKey, run.CreatedBy)
	return workerCtx, nil
}

func newAgentEvaluationQARequest(
	run *types.ExamRAGEvaluationRun,
	agentSnapshot types.ExamAgentSnapshot,
	scenario types.ExamAgentEvaluationCase,
) *types.QARequest {
	agent := &types.CustomAgent{
		ID: agentSnapshot.ID, Name: agentSnapshot.Name, TenantID: run.TenantID,
		CreatedBy: run.CreatedBy, Config: agentSnapshot.Config,
	}
	return &types.QARequest{
		Session: &types.Session{
			ID: uuid.NewString(), TenantID: run.TenantID, UserID: run.CreatedBy,
			Title: "Agent evaluation: " + scenario.Name,
		},
		Query: scenario.Input, AssistantMessageID: uuid.NewString(), UserMessageID: uuid.NewString(),
		CustomAgent: agent, SummaryModelID: agentSnapshot.ModelID,
		KnowledgeBaseIDs: append([]string(nil), agentSnapshot.KnowledgeBases...),
		WebSearchEnabled: false, EnableMemory: false, DisableHistory: true,
	}
}

func (s *examAgentEvaluationService) markAgentRunRunning(
	ctx context.Context, run *types.ExamRAGEvaluationRun,
) error {
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusRunning
	run.ErrorMessage = ""
	if run.StartedAt == nil {
		run.StartedAt = &now
	}
	run.UpdatedAt = now
	return s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, types.ExamEvaluationKindAgent, run)
}

func (s *examAgentEvaluationService) persistAgentRunProgress(
	ctx context.Context,
	run *types.ExamRAGEvaluationRun,
	result *types.ExamAgentEvaluationResult,
	completed int,
	total int,
) error {
	snapshot, err := json.Marshal(result)
	if err != nil {
		return err
	}
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{CompletedCases: completed, TotalCases: total})
	if err != nil {
		return err
	}
	run.ResultSnapshot = snapshot
	run.Progress = progress
	run.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, types.ExamEvaluationKindAgent, run)
}

func (s *examAgentEvaluationService) persistAgentRunCompletion(
	ctx context.Context, run *types.ExamRAGEvaluationRun, result *types.ExamAgentEvaluationResult,
) error {
	if err := s.persistAgentRunProgress(ctx, run, result, len(result.Results), len(result.Results)); err != nil {
		return err
	}
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusCompleted
	run.ErrorMessage = ""
	run.CompletedAt = &now
	run.UpdatedAt = now
	return s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, types.ExamEvaluationKindAgent, run)
}

func (s *examAgentEvaluationService) persistAgentRunFailure(
	ctx context.Context, run *types.ExamRAGEvaluationRun, cause error,
) error {
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusFailed
	run.ErrorMessage = cause.Error()
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, types.ExamEvaluationKindAgent, run); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

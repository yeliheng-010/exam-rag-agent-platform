package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestExamAgentEvaluationWorkerPersistsScenarioResultsAndProgress(t *testing.T) {
	t.Parallel()
	run := testQueuedAgentEvaluationRun(t, "agent-run-1")
	repo := &examAgentEvaluationRunRepoStub{
		examRAGEvaluationRunRepoStub: newExamRAGEvaluationRunRepoStub(),
	}
	repo.runs[run.ID] = run
	runner := &examAgentEvaluationRunnerStub{states: []*types.AgentState{
		{
			FinalAnswer: "question 21 comes from chunk-21",
			RoundSteps: []types.AgentStep{{ToolCalls: []types.ToolCall{
				{Name: "exam_class_diagnosis", Args: map[string]interface{}{"class_id": "class-1"}, Result: &types.ToolResult{Success: true, Output: "question 21"}},
				{Name: "exam_question_context", Result: &types.ToolResult{Success: true, Output: "question 21 chunk-21"}},
			}}},
		},
	}}
	svc := NewExamAgentEvaluationService(
		repo, testRAGQuestionService(), nil, testRAGTenantService(), runner, &examRAGEvaluationEnqueuerStub{},
	)

	err := svc.ProcessRunTask(context.Background(), testAgentEvaluationTask(t, run.ID))

	require.NoError(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusCompleted, run.Status)
	var progress types.ExamRAGEvaluationProgress
	require.NoError(t, json.Unmarshal(run.Progress, &progress))
	require.Equal(t, 1, progress.CompletedCases)
	require.Equal(t, 1, progress.TotalCases)
	var result types.ExamAgentEvaluationResult
	require.NoError(t, json.Unmarshal(run.ResultSnapshot, &result))
	require.Equal(t, 1, result.Summary.Total)
	require.Equal(t, 1, result.Summary.Passed)
	require.Len(t, result.Results[0].ActualToolCalls, 2)
	require.NotEmpty(t, repo.progressHistory)
	require.Len(t, runner.requests, 1)
	require.True(t, runner.requests[0].DisableHistory)
}

func TestExamAgentEvaluationWorkerHandlesEmptyAgentStateWithoutPanic(t *testing.T) {
	t.Parallel()
	run := testQueuedAgentEvaluationRun(t, "agent-run-empty")
	repo := &examAgentEvaluationRunRepoStub{examRAGEvaluationRunRepoStub: newExamRAGEvaluationRunRepoStub()}
	repo.runs[run.ID] = run
	svc := NewExamAgentEvaluationService(
		repo, testRAGQuestionService(), nil, testRAGTenantService(),
		&examAgentEvaluationRunnerStub{states: []*types.AgentState{nil}},
		&examRAGEvaluationEnqueuerStub{},
	)

	err := svc.ProcessRunTask(context.Background(), testAgentEvaluationTask(t, run.ID))

	require.NoError(t, err)
	var result types.ExamAgentEvaluationResult
	require.NoError(t, json.Unmarshal(run.ResultSnapshot, &result))
	require.Equal(t, 1, result.Summary.Failed)
	require.Contains(t, result.Results[0].Error, "empty state")
}

func TestExamAgentEvaluationWorkerContinuesAfterScenarioFailure(t *testing.T) {
	t.Parallel()
	run := testQueuedAgentEvaluationRun(t, "agent-run-continues")
	var snapshot types.ExamAgentEvaluationRequestSnapshot
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &snapshot))
	snapshot.Cases = append(snapshot.Cases, types.ExamAgentEvaluationCase{
		Name: "second", Input: "explain question 22",
		ExpectedToolCalls: []types.ExamAgentExpectedToolCall{{Name: "exam_question_context"}},
	})
	run.RequestSnapshot, _ = json.Marshal(snapshot)
	run.Progress, _ = json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: 2})
	repo := &examAgentEvaluationRunRepoStub{examRAGEvaluationRunRepoStub: newExamRAGEvaluationRunRepoStub()}
	repo.runs[run.ID] = run
	runner := &examAgentEvaluationRunnerStub{
		states: []*types.AgentState{nil, {
			FinalAnswer: "question 22", RoundSteps: []types.AgentStep{{ToolCalls: []types.ToolCall{{
				Name: "exam_question_context", Result: &types.ToolResult{Success: true, Output: "question 22"},
			}}}},
		}},
		errs: []error{errors.New("model timeout")},
	}
	svc := NewExamAgentEvaluationService(
		repo, testRAGQuestionService(), nil, testRAGTenantService(), runner, &examRAGEvaluationEnqueuerStub{},
	)

	err := svc.ProcessRunTask(context.Background(), testAgentEvaluationTask(t, run.ID))

	require.NoError(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusCompleted, run.Status)
	var result types.ExamAgentEvaluationResult
	require.NoError(t, json.Unmarshal(run.ResultSnapshot, &result))
	require.Len(t, result.Results, 2)
	require.Equal(t, 1, result.Summary.Failed)
	require.Equal(t, 1, result.Summary.Passed)
	require.Contains(t, result.Results[0].Error, "model timeout")
	require.Empty(t, result.Results[1].Error)
	require.Len(t, runner.requests, 2)
	require.NotEqual(t, runner.requests[0].Session.ID, runner.requests[1].Session.ID)
}

func testQueuedAgentEvaluationRun(t *testing.T, id string) *types.ExamRAGEvaluationRun {
	t.Helper()
	req := testAgentEvaluationRequest()
	agent := testEvaluationAgent()
	agent.Config = safeAgentEvaluationConfig(agent.Config)
	snapshot, err := json.Marshal(types.ExamAgentEvaluationRequestSnapshot{
		Agent: newExamAgentSnapshot(agent), Cases: req.Cases,
	})
	require.NoError(t, err)
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: len(req.Cases)})
	require.NoError(t, err)
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	return &types.ExamRAGEvaluationRun{
		ID: id, TenantID: 10000, QuestionBankID: "bank-1", EvaluationKind: types.ExamEvaluationKindAgent,
		AgentID: "agent-1", CreatedBy: "teacher-1", Status: types.ExamRAGEvaluationRunStatusQueued,
		RequestSnapshot: snapshot, Progress: progress, CreatedAt: now, UpdatedAt: now,
	}
}

func testAgentEvaluationTask(t *testing.T, runID string) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(types.ExamAgentEvaluationTaskPayload{RunID: runID})
	require.NoError(t, err)
	return asynq.NewTask(types.TypeExamAgentEvaluationRun, payload)
}

type examAgentEvaluationRunnerStub struct {
	interfaces.SessionService
	states   []*types.AgentState
	errs     []error
	requests []*types.QARequest
}

func (s *examAgentEvaluationRunnerStub) ExecuteAgentEvaluation(
	_ context.Context, req *types.QARequest, _ *event.EventBus,
) (*types.AgentState, error) {
	s.requests = append(s.requests, req)
	index := len(s.requests) - 1
	var err error
	if index < len(s.errs) {
		err = s.errs[index]
	}
	if index >= len(s.states) {
		return nil, err
	}
	return s.states[index], err
}

type examAgentEvaluationRunRepoStub struct {
	*examRAGEvaluationRunRepoStub
	progressHistory []types.ExamRAGEvaluationProgress
}

func (r *examAgentEvaluationRunRepoStub) UpdateRun(
	ctx context.Context, tenantID uint64, bankID string, kind types.ExamEvaluationKind,
	run *types.ExamRAGEvaluationRun,
) error {
	var progress types.ExamRAGEvaluationProgress
	if json.Unmarshal(run.Progress, &progress) == nil {
		r.progressHistory = append(r.progressHistory, progress)
	}
	return r.examRAGEvaluationRunRepoStub.UpdateRun(ctx, tenantID, bankID, kind, run)
}

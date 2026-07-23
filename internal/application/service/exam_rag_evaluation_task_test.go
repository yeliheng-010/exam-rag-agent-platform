package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestExamRAGEvaluationWorkerCompletesRunWithCaseFailures(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	repo.runs["run-1"] = testQueuedRAGEvaluationRun("run-1")
	diagnostic := &examRAGEvaluationDiagnosticStub{result: &types.ExamRAGDiagnosticResult{
		Summary: types.ExamRAGDiagnosticSummary{
			Total: 2, Passed: 1, FailedCaseCount: 1,
			Results: []types.ExamRAGDiagnosticResultItem{{Name: "case-1", Passed: true}, {Name: "case-2", Error: "search failed"}},
		},
	}}
	svc := NewExamRAGEvaluationService(repo, testRAGQuestionService(), diagnostic, testRAGTenantService(), newExamRAGChunkQualityRepoStub(), &examRAGEvaluationEnqueuerStub{})

	err := svc.ProcessRunTask(context.Background(), testRAGEvaluationTask(t, "run-1"))

	require.NoError(t, err)
	run := repo.runs["run-1"]
	require.Equal(t, types.ExamRAGEvaluationRunStatusCompleted, run.Status)
	require.NotNil(t, run.StartedAt)
	require.NotNil(t, run.CompletedAt)
	var result types.ExamRAGDiagnosticResult
	require.NoError(t, json.Unmarshal(run.ResultSnapshot, &result))
	require.Equal(t, 1, result.Summary.FailedCaseCount)
	require.True(t, result.UsedDefaultCases)
	require.NotNil(t, diagnostic.tenantInfo)
	require.Equal(t, uint64(10000), diagnostic.tenantInfo.ID)
	var progress types.ExamRAGEvaluationProgress
	require.NoError(t, json.Unmarshal(run.Progress, &progress))
	require.Equal(t, 2, progress.CompletedCases)
}

func TestExamRAGEvaluationWorkerPersistsTerminalFailure(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	repo.runs["run-1"] = testQueuedRAGEvaluationRun("run-1")
	diagnostic := &examRAGEvaluationDiagnosticStub{err: errRAGEvaluationWorker}
	svc := NewExamRAGEvaluationService(repo, testRAGQuestionService(), diagnostic, testRAGTenantService(), newExamRAGChunkQualityRepoStub(), &examRAGEvaluationEnqueuerStub{})

	err := svc.ProcessRunTask(context.Background(), testRAGEvaluationTask(t, "run-1"))

	require.ErrorIs(t, err, errRAGEvaluationWorker)
	run := repo.runs["run-1"]
	require.Equal(t, types.ExamRAGEvaluationRunStatusFailed, run.Status)
	require.Equal(t, errRAGEvaluationWorker.Error(), run.ErrorMessage)
	require.NotNil(t, run.CompletedAt)
}

func TestExamRAGEvaluationWorkerRejectsNilDiagnosticResult(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	repo.runs["run-1"] = testQueuedRAGEvaluationRun("run-1")
	diagnostic := &examRAGEvaluationDiagnosticStub{}
	svc := NewExamRAGEvaluationService(repo, testRAGQuestionService(), diagnostic, testRAGTenantService(), newExamRAGChunkQualityRepoStub(), &examRAGEvaluationEnqueuerStub{})

	err := svc.ProcessRunTask(context.Background(), testRAGEvaluationTask(t, "run-1"))

	require.Error(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusFailed, repo.runs["run-1"].Status)
	require.Contains(t, repo.runs["run-1"].ErrorMessage, "empty result")
}

func TestExamRAGEvaluationWorkerSkipsTerminalRun(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	run := testQueuedRAGEvaluationRun("run-1")
	run.Status = types.ExamRAGEvaluationRunStatusCompleted
	repo.runs[run.ID] = run
	diagnostic := &examRAGEvaluationDiagnosticStub{}
	svc := NewExamRAGEvaluationService(repo, testRAGQuestionService(), diagnostic, testRAGTenantService(), newExamRAGChunkQualityRepoStub(), &examRAGEvaluationEnqueuerStub{})

	require.NoError(t, svc.ProcessRunTask(context.Background(), testRAGEvaluationTask(t, "run-1")))
	require.Zero(t, diagnostic.calls)
}

func testQueuedRAGEvaluationRun(id string) *types.ExamRAGEvaluationRun {
	preparation := testRAGPreparation()
	request, _ := json.Marshal(examRAGEvaluationRequestSnapshot{
		RunExamRAGDiagnosticRequest: preparation.Request,
		UsedDefaultCases:            preparation.UsedDefaultCases,
	})
	progress, _ := json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: 2})
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	return &types.ExamRAGEvaluationRun{
		ID: id, TenantID: 10000, QuestionBankID: "bank-1", CreatedBy: "teacher-1",
		EvaluationKind: types.ExamEvaluationKindRAG,
		Status:         types.ExamRAGEvaluationRunStatusQueued, RequestSnapshot: request, Progress: progress,
		CreatedAt: now, UpdatedAt: now,
	}
}

func testRAGEvaluationTask(t *testing.T, runID string) *asynq.Task {
	t.Helper()
	payload, err := json.Marshal(types.ExamRAGEvaluationTaskPayload{RunID: runID})
	require.NoError(t, err)
	return asynq.NewTask(types.TypeExamRAGEvaluationRun, payload)
}

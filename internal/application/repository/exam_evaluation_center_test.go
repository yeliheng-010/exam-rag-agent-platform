package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestEvaluationCenterRepositoryFiltersVisibleBanksWithoutEmptyScopeLeak(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	runRepo := NewExamRAGEvaluationRepository(db)
	repo := NewExamEvaluationCenterRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	completed := testExamRAGEvaluationRun("run-visible", 10000, "bank-1", now)
	completed.Status = types.ExamRAGEvaluationRunStatusCompleted
	completed.ResultSnapshot = types.JSON(`{"summary":{"hit_rate":0.8}}`)
	require.NoError(t, runRepo.CreateRun(ctx, completed))
	require.NoError(t, runRepo.CreateRun(ctx, testExamRAGEvaluationRun("run-hidden", 10000, "bank-2", now.Add(time.Minute))))
	require.NoError(t, runRepo.CreateRun(ctx, testExamRAGEvaluationRun("run-other-tenant", 20000, "bank-1", now.Add(2*time.Minute))))

	visible, err := repo.ListCenterRuns(ctx, 10000, []string{"bank-1"}, types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKindRAG, Limit: 20,
	})
	require.NoError(t, err)
	require.Len(t, visible, 1)
	require.Equal(t, "run-visible", visible[0].ID)

	empty, err := repo.ListCenterRuns(ctx, 10000, []string{}, types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKindRAG, Limit: 20,
	})
	require.NoError(t, err)
	require.Empty(t, empty)

	allTenant, err := repo.ListCenterRuns(ctx, 10000, nil, types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKindRAG, Limit: 20,
	})
	require.NoError(t, err)
	require.Len(t, allTenant, 2)
}

func TestEvaluationCenterRepositoryReplacesBaselineWithinExactScope(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	runRepo := NewExamRAGEvaluationRepository(db)
	repo := NewExamEvaluationCenterRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 11, 0, 0, 0, time.UTC)

	for _, id := range []string{"run-old", "run-new"} {
		run := testExamRAGEvaluationRun(id, 10000, "bank-1", now)
		run.Status = types.ExamRAGEvaluationRunStatusCompleted
		require.NoError(t, runRepo.CreateRun(ctx, run))
	}
	otherKind := testExamRAGEvaluationRun("run-agent", 10000, "bank-1", now)
	otherKind.Status = types.ExamRAGEvaluationRunStatusCompleted
	otherKind.EvaluationKind = types.ExamEvaluationKindAgent
	otherKind.AgentID = "agent-1"
	require.NoError(t, runRepo.CreateRun(ctx, otherKind))

	require.NoError(t, repo.SetBaseline(ctx, 10000, "run-old"))
	require.NoError(t, repo.SetBaseline(ctx, 10000, "run-agent"))
	require.NoError(t, repo.SetBaseline(ctx, 10000, "run-new"))

	ragBaselines, err := repo.ListBaselines(ctx, 10000, nil, types.ExamEvaluationKindRAG, "")
	require.NoError(t, err)
	require.Len(t, ragBaselines, 1)
	require.Equal(t, "run-new", ragBaselines[0].ID)

	agentBaselines, err := repo.ListBaselines(ctx, 10000, nil, types.ExamEvaluationKindAgent, "agent-1")
	require.NoError(t, err)
	require.Len(t, agentBaselines, 1)
	require.Equal(t, "run-agent", agentBaselines[0].ID)

	queued := testExamRAGEvaluationRun("run-queued", 10000, "bank-1", now)
	require.NoError(t, runRepo.CreateRun(ctx, queued))
	require.ErrorIs(t, repo.SetBaseline(ctx, 10000, "run-queued"), ErrExamEvaluationRunNotCompleted)
	require.ErrorIs(t, repo.SetBaseline(ctx, 20000, "run-new"), ErrExamRAGEvaluationRunNotFound)
}

func TestEvaluationCenterRepositoryCreatesImmutableVersionSequence(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	repo := NewExamEvaluationCenterRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	set := &types.ExamEvaluationSet{
		ID: "set-1", TenantID: 10000, QuestionBankID: "bank-1",
		EvaluationKind: types.ExamEvaluationKindRAG, Name: "高考英语回归集",
		CurrentVersion: 1, CreatedBy: "teacher-1", Status: types.ExamEvaluationSetStatusActive,
		CreatedAt: now, UpdatedAt: now,
	}
	first := &types.ExamEvaluationSetVersion{
		ID: "version-1", TenantID: 10000, EvaluationSetID: "set-1", Version: 1,
		SourceRunID: "run-1", DefinitionSnapshot: types.JSON(`{"cases":[{"name":"A篇"}]}`),
		CreatedBy: "teacher-1", CreatedAt: now,
	}
	require.NoError(t, repo.CreateEvaluationSet(ctx, set, first))

	second, err := repo.CreateEvaluationSetVersion(ctx, 10000, "set-1", &types.ExamEvaluationSetVersion{
		ID: "version-2", SourceRunID: "run-2",
		DefinitionSnapshot: types.JSON(`{"cases":[{"name":"B篇"}]}`),
		CreatedBy:          "teacher-1", CreatedAt: now.Add(time.Minute),
	})
	require.NoError(t, err)
	require.Equal(t, 2, second.Version)

	storedSet, err := repo.GetEvaluationSet(ctx, 10000, "set-1")
	require.NoError(t, err)
	require.Equal(t, 2, storedSet.CurrentVersion)

	versions, err := repo.ListEvaluationSetVersions(ctx, 10000, "set-1")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	require.Equal(t, 2, versions[0].Version)
	require.Equal(t, 1, versions[1].Version)

	_, err = repo.CreateEvaluationSetVersion(ctx, 20000, "set-1", &types.ExamEvaluationSetVersion{
		ID: "version-cross-tenant", SourceRunID: "run-3", DefinitionSnapshot: types.JSON(`{}`),
		CreatedBy: "teacher-2", CreatedAt: now,
	})
	require.ErrorIs(t, err, ErrExamEvaluationSetNotFound)
}

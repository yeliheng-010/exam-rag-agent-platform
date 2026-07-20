package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamRAGEvaluationRepositoryScopesRunsByTenantAndBank(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	repo := NewExamRAGEvaluationRepository(db)
	ctx := context.Background()
	now := time.Date(2026, 7, 14, 16, 0, 0, 0, time.UTC)

	require.NoError(t, repo.CreateRun(ctx, testExamRAGEvaluationRun("run-visible", 10000, "bank-1", now)))
	require.NoError(t, repo.CreateRun(ctx, testExamRAGEvaluationRun("run-other-bank", 10000, "bank-2", now.Add(time.Minute))))
	require.NoError(t, repo.CreateRun(ctx, testExamRAGEvaluationRun("run-other-tenant", 20000, "bank-1", now.Add(2*time.Minute))))
	agentRun := testExamRAGEvaluationRun("run-agent", 10000, "bank-1", now.Add(3*time.Minute))
	agentRun.EvaluationKind = types.ExamEvaluationKindAgent
	agentRun.AgentID = "agent-1"
	require.NoError(t, repo.CreateRun(ctx, agentRun))

	runs, err := repo.ListRuns(ctx, 10000, "bank-1", types.ExamEvaluationKindRAG, 20)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, "run-visible", runs[0].ID)

	agentRuns, err := repo.ListRuns(ctx, 10000, "bank-1", types.ExamEvaluationKindAgent, 20)
	require.NoError(t, err)
	require.Len(t, agentRuns, 1)
	require.Equal(t, "run-agent", agentRuns[0].ID)

	visible, err := repo.GetRun(ctx, 10000, "bank-1", types.ExamEvaluationKindRAG, "run-visible")
	require.NoError(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusQueued, visible.Status)

	_, err = repo.GetRun(ctx, 10000, "bank-1", types.ExamEvaluationKindRAG, "run-agent")
	require.ErrorIs(t, err, ErrExamRAGEvaluationRunNotFound)
	_, err = repo.GetRun(ctx, 10000, "bank-2", types.ExamEvaluationKindRAG, "run-visible")
	require.ErrorIs(t, err, ErrExamRAGEvaluationRunNotFound)
	_, err = repo.GetRun(ctx, 20000, "bank-1", types.ExamEvaluationKindRAG, "run-visible")
	require.ErrorIs(t, err, ErrExamRAGEvaluationRunNotFound)
}

func TestExamRAGEvaluationRepositoryUpdateCannotCrossBankScope(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	repo := NewExamRAGEvaluationRepository(db)
	ctx := context.Background()
	run := testExamRAGEvaluationRun("run-1", 10000, "bank-1", time.Now().UTC())
	require.NoError(t, repo.CreateRun(ctx, run))

	run.Status = types.ExamRAGEvaluationRunStatusCompleted
	err := repo.UpdateRun(ctx, 10000, "bank-2", types.ExamEvaluationKindRAG, run)
	require.ErrorIs(t, err, ErrExamRAGEvaluationRunNotFound)

	stored, err := repo.GetRun(ctx, 10000, "bank-1", types.ExamEvaluationKindRAG, "run-1")
	require.NoError(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusQueued, stored.Status)
}

func TestExamRAGEvaluationRepositoryLeavesEvaluationSetColumnsNullForOrdinaryRun(t *testing.T) {
	t.Parallel()

	db := newExamRAGEvaluationRepositoryTestDB(t)
	repo := NewExamRAGEvaluationRepository(db)
	run := testExamRAGEvaluationRun("run-without-set", 10000, "bank-1", time.Now().UTC())

	require.NoError(t, repo.CreateRun(context.Background(), run))

	var stored struct {
		EvaluationSetID      *string
		EvaluationSetVersion *int
	}
	require.NoError(t, db.Table("exam_rag_evaluation_runs").
		Select("evaluation_set_id, evaluation_set_version").
		Where("id = ?", run.ID).Scan(&stored).Error)
	require.Nil(t, stored.EvaluationSetID)
	require.Nil(t, stored.EvaluationSetVersion)
}

func newExamRAGEvaluationRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE exam_rag_evaluation_runs (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			question_bank_id VARCHAR(36) NOT NULL,
			evaluation_kind VARCHAR(16) NOT NULL DEFAULT 'rag',
			agent_id VARCHAR(36),
			is_baseline BOOLEAN NOT NULL DEFAULT FALSE,
			evaluation_set_id VARCHAR(36),
			evaluation_set_version INTEGER,
			created_by VARCHAR(36) NOT NULL,
			status VARCHAR(32) NOT NULL,
			progress TEXT NOT NULL,
			request_snapshot TEXT NOT NULL,
			result_snapshot TEXT,
			error_message TEXT NOT NULL DEFAULT '',
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE exam_evaluation_sets (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			question_bank_id VARCHAR(36) NOT NULL,
			evaluation_kind VARCHAR(16) NOT NULL,
			agent_id VARCHAR(36),
			name VARCHAR(160) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			current_version INTEGER NOT NULL DEFAULT 1,
			created_by VARCHAR(36) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE exam_evaluation_set_versions (
			id VARCHAR(36) PRIMARY KEY,
			tenant_id BIGINT NOT NULL,
			evaluation_set_id VARCHAR(36) NOT NULL,
			version INTEGER NOT NULL,
			source_run_id VARCHAR(36) NOT NULL,
			definition_snapshot TEXT NOT NULL,
			created_by VARCHAR(36) NOT NULL,
			created_at DATETIME NOT NULL,
			UNIQUE(evaluation_set_id, version)
		)
	`).Error)
	return db
}

func testExamRAGEvaluationRun(
	id string,
	tenantID uint64,
	bankID string,
	createdAt time.Time,
) *types.ExamRAGEvaluationRun {
	return &types.ExamRAGEvaluationRun{
		ID:              id,
		TenantID:        tenantID,
		QuestionBankID:  bankID,
		EvaluationKind:  types.ExamEvaluationKindRAG,
		CreatedBy:       "teacher-1",
		Status:          types.ExamRAGEvaluationRunStatusQueued,
		Progress:        types.JSON(`{"completed_cases":0,"total_cases":2}`),
		RequestSnapshot: types.JSON(`{"match_count":8}`),
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
}

package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type ExamRAGEvaluationRepository interface {
	CreateRun(ctx context.Context, run *types.ExamRAGEvaluationRun) error
	GetRun(ctx context.Context, tenantID uint64, bankID string, runID string) (*types.ExamRAGEvaluationRun, error)
	GetRunForTask(ctx context.Context, runID string) (*types.ExamRAGEvaluationRun, error)
	ListRuns(ctx context.Context, tenantID uint64, bankID string, limit int) ([]*types.ExamRAGEvaluationRun, error)
	UpdateRun(ctx context.Context, tenantID uint64, bankID string, run *types.ExamRAGEvaluationRun) error
}

type ExamRAGEvaluationService interface {
	CreateRun(ctx context.Context, tenantID uint64, userID string, bankID string, req *types.RunExamRAGDiagnosticRequest) (*types.ExamRAGEvaluationRun, error)
	ListRuns(ctx context.Context, tenantID uint64, userID string, bankID string, limit int) ([]*types.ExamRAGEvaluationRun, error)
	GetRun(ctx context.Context, tenantID uint64, userID string, bankID string, runID string) (*types.ExamRAGEvaluationRun, error)
	ProcessRunTask(ctx context.Context, task *asynq.Task) error
}

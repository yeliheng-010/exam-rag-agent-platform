package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type ExamAgentEvaluationService interface {
	CreateRun(ctx context.Context, tenantID uint64, userID string, bankID string, req *types.RunExamAgentEvaluationRequest) (*types.ExamRAGEvaluationRun, error)
	ListRuns(ctx context.Context, tenantID uint64, userID string, bankID string, limit int) ([]*types.ExamRAGEvaluationRun, error)
	GetRun(ctx context.Context, tenantID uint64, userID string, bankID string, runID string) (*types.ExamRAGEvaluationRun, error)
	ProcessRunTask(ctx context.Context, task *asynq.Task) error
}

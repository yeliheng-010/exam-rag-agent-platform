package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamEvaluationCenterRepository interface {
	ListCenterQuestionBanks(ctx context.Context, tenantID uint64, bankIDs []string) ([]*types.QuestionBank, error)
	GetCenterRun(ctx context.Context, tenantID uint64, runID string) (*types.ExamRAGEvaluationRun, error)
	ListCenterRuns(ctx context.Context, tenantID uint64, bankIDs []string, filter types.ExamEvaluationCenterFilter) ([]*types.ExamRAGEvaluationRun, error)
	ListBaselines(ctx context.Context, tenantID uint64, bankIDs []string, kind types.ExamEvaluationKind, agentID string) ([]*types.ExamRAGEvaluationRun, error)
	SetBaseline(ctx context.Context, tenantID uint64, runID string) error
	AttachRunEvaluationSet(ctx context.Context, tenantID uint64, runID string, setID string, version int) error
	CreateEvaluationSet(ctx context.Context, set *types.ExamEvaluationSet, version *types.ExamEvaluationSetVersion) error
	ListEvaluationSets(ctx context.Context, tenantID uint64, bankIDs []string, filter types.ExamEvaluationCenterFilter) ([]*types.ExamEvaluationSet, error)
	GetEvaluationSet(ctx context.Context, tenantID uint64, setID string) (*types.ExamEvaluationSet, error)
	ListEvaluationSetVersions(ctx context.Context, tenantID uint64, setID string) ([]*types.ExamEvaluationSetVersion, error)
	GetEvaluationSetVersion(ctx context.Context, tenantID uint64, setID string, version int) (*types.ExamEvaluationSetVersion, error)
	CreateEvaluationSetVersion(ctx context.Context, tenantID uint64, setID string, version *types.ExamEvaluationSetVersion) (*types.ExamEvaluationSetVersion, error)
}

type ExamEvaluationCenterService interface {
	GetCenter(ctx context.Context, tenantID uint64, userID string, filter types.ExamEvaluationCenterFilter) (*types.ExamEvaluationCenterResult, error)
	SetBaseline(ctx context.Context, tenantID uint64, userID string, runID string) (*types.ExamRAGEvaluationRun, error)
	ListEvaluationSets(ctx context.Context, tenantID uint64, userID string, filter types.ExamEvaluationCenterFilter) ([]*types.ExamEvaluationSet, error)
	CreateEvaluationSet(ctx context.Context, tenantID uint64, userID string, req *types.CreateExamEvaluationSetRequest) (*types.ExamEvaluationSet, error)
	CreateEvaluationSetVersion(ctx context.Context, tenantID uint64, userID string, setID string, req *types.CreateExamEvaluationSetVersionRequest) (*types.ExamEvaluationSetVersion, error)
	RunEvaluationSet(ctx context.Context, tenantID uint64, userID string, setID string, req *types.RunExamEvaluationSetRequest) (*types.ExamRAGEvaluationRun, error)
}

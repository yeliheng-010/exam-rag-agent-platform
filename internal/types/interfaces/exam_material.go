package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamMaterialService interface {
	RegisterMaterial(ctx context.Context, tenantID uint64, userID string, req *types.RegisterExamMaterialRequest) (*types.ExamMaterialRegistrationResult, error)
	ListMaterials(ctx context.Context, tenantID uint64, userID string, filter types.ListExamMaterialsFilter) ([]*types.ExamMaterial, error)
	CreateStructuringTask(ctx context.Context, tenantID uint64, userID string, materialID string, req *types.CreateExamStructuringTaskRequest) (*types.ExamStructuringTask, error)
	ListStructuringTasks(ctx context.Context, tenantID uint64, userID string, filter types.ListExamStructuringTasksFilter) ([]*types.ExamStructuringTask, error)
}

type ExamMaterialRepository interface {
	UpsertMaterial(ctx context.Context, material *types.ExamMaterial) error
	GetMaterialByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamMaterial, error)
	GetMaterialByKnowledge(ctx context.Context, tenantID uint64, knowledgeID string) (*types.ExamMaterial, error)
	ListMaterials(ctx context.Context, tenantID uint64, filter types.ListExamMaterialsFilter, spaceIDs []string) ([]*types.ExamMaterial, error)
	CreateStructuringTask(ctx context.Context, task *types.ExamStructuringTask) error
	ListStructuringTasks(ctx context.Context, tenantID uint64, filter types.ListExamStructuringTasksFilter, spaceIDs []string) ([]*types.ExamStructuringTask, error)
}

type ExamMaterialKnowledgeBaseReader interface {
	GetKnowledgeBaseByID(ctx context.Context, id string) (*types.KnowledgeBase, error)
}

type ExamMaterialKnowledgeReader interface {
	GetKnowledgeByID(ctx context.Context, id string) (*types.Knowledge, error)
}

type ExamMaterialChunkReader interface {
	ListChunksByKnowledgeID(ctx context.Context, knowledgeID string) ([]*types.Chunk, error)
}

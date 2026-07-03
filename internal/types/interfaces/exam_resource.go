package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamResourceService interface {
	BindKnowledgeBase(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string, req *types.BindKnowledgeBaseResourceRequest) (*types.ExamSpaceResource, error)
	GetKnowledgeBaseBinding(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string) (*types.ExamSpaceResource, error)
	CanReadKnowledgeBase(ctx context.Context, tenantID uint64, userID string, knowledgeBaseID string) (bool, error)
	ListResources(ctx context.Context, tenantID uint64, userID string, filter types.ListExamResourcesFilter) ([]*types.ExamSpaceResource, error)
}

type ExamResourceRepository interface {
	Upsert(ctx context.Context, resource *types.ExamSpaceResource) error
	GetByResource(ctx context.Context, tenantID uint64, resourceType types.ExamResourceType, resourceID string) (*types.ExamSpaceResource, error)
	List(ctx context.Context, tenantID uint64, filter types.ListExamResourcesFilter, spaceIDs []string) ([]*types.ExamSpaceResource, error)
}

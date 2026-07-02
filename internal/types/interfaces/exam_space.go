package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamSpaceService interface {
	ListSpaces(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamSpace, error)
	EnsurePersonalSpace(ctx context.Context, tenantID uint64, userID string) (*types.ExamSpace, error)
	CanReadSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (bool, error)
	CanWriteSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (bool, error)
	GetSpace(ctx context.Context, tenantID uint64, userID string, spaceID string) (*types.ExamSpace, error)
}

type ExamSpaceRepository interface {
	Create(ctx context.Context, space *types.ExamSpace) error
	ListByTenant(ctx context.Context, tenantID uint64) ([]*types.ExamSpace, error)
	GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamSpace, error)
	GetPersonalByOwner(ctx context.Context, tenantID uint64, userID string) (*types.ExamSpace, error)
}

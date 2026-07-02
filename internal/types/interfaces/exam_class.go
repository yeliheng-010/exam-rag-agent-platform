package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamClassService interface {
	CreateClass(ctx context.Context, tenantID uint64, userID string, req *types.CreateExamClassRequest) (*types.ExamClass, error)
	ListClasses(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error)
	GetClass(ctx context.Context, tenantID uint64, userID string, classID string) (*types.ExamClass, error)
	CanAccessClass(ctx context.Context, tenantID uint64, userID string, classID string) (bool, error)
	CanWriteClass(ctx context.Context, tenantID uint64, userID string, classID string) (bool, error)
}

type ExamClassRepository interface {
	CreateClass(ctx context.Context, class *types.ExamClass) error
	AddMember(ctx context.Context, member *types.ExamClassMember) error
	ListByUser(ctx context.Context, tenantID uint64, userID string) ([]*types.ExamClass, error)
	GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamClass, error)
	GetBySpaceIDAndTenant(ctx context.Context, spaceID string, tenantID uint64) (*types.ExamClass, error)
	GetMember(ctx context.Context, classID string, tenantID uint64, userID string) (*types.ExamClassMember, error)
}

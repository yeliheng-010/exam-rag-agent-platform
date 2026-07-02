package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamTeacherApplicationService interface {
	Apply(ctx context.Context, tenantID uint64, userID string, req *types.ApplyExamTeacherRequest) (*types.ExamTeacherApplication, error)
	GetMine(ctx context.Context, tenantID uint64, userID string) (*types.ExamTeacherApplication, error)
	List(ctx context.Context, tenantID uint64, status *types.ExamTeacherApplicationStatus) ([]*types.ExamTeacherApplicationView, error)
	Approve(ctx context.Context, tenantID uint64, reviewerID string, applicationID string, req *types.ReviewExamTeacherApplicationRequest) (*types.ExamTeacherApplication, error)
	Reject(ctx context.Context, tenantID uint64, reviewerID string, applicationID string, req *types.ReviewExamTeacherApplicationRequest) (*types.ExamTeacherApplication, error)
	IsApprovedTeacher(ctx context.Context, tenantID uint64, userID string) (bool, error)
}

type ExamTeacherAccessService interface {
	IsApprovedTeacher(ctx context.Context, tenantID uint64, userID string) (bool, error)
}

type ExamTeacherApplicationRepository interface {
	UpsertPending(ctx context.Context, app *types.ExamTeacherApplication) error
	GetByTenantAndUser(ctx context.Context, tenantID uint64, userID string) (*types.ExamTeacherApplication, error)
	GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamTeacherApplication, error)
	ListByTenant(ctx context.Context, tenantID uint64, status *types.ExamTeacherApplicationStatus) ([]*types.ExamTeacherApplication, error)
	UpdateReview(ctx context.Context, id string, tenantID uint64, status types.ExamTeacherApplicationStatus, reviewerID string, reviewNote string) (*types.ExamTeacherApplication, error)
}

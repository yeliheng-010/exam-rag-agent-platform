package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamAssignmentService interface {
	CreateAssignment(ctx context.Context, tenantID uint64, userID string, classID string, req *types.CreateExamAssignmentRequest) (*types.ExamAssignmentSummary, error)
	UpdateAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string, req *types.UpdateExamAssignmentRequest) (*types.ExamAssignmentSummary, error)
	WithdrawAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string) (*types.ExamAssignmentSummary, error)
	RepublishAssignment(ctx context.Context, tenantID uint64, userID, classID, assignmentID string) (*types.ExamAssignmentSummary, error)
	ListClassAssignments(ctx context.Context, tenantID uint64, userID string, classID string, filter types.ListExamAssignmentsFilter) ([]*types.ExamAssignmentSummary, error)
	ListMyAssignments(ctx context.Context, tenantID uint64, userID string, filter types.ListExamAssignmentsFilter) ([]*types.ExamAssignmentSummary, error)
	CreateAssignmentAttempt(ctx context.Context, tenantID uint64, userID string, assignmentID string) (*types.CreatePracticeAttemptResult, error)
	GetAssignmentProgress(ctx context.Context, tenantID uint64, userID string, classID string, assignmentID string) (*types.ExamAssignmentProgressSummary, error)
}

type ExamAssignmentRepository interface {
	CreateAssignment(ctx context.Context, assignment *types.ExamClassAssignment) error
	GetAssignmentByIDAndTenant(ctx context.Context, tenantID uint64, assignmentID string) (*types.ExamClassAssignment, error)
	ListAssignmentsByClass(ctx context.Context, tenantID uint64, classID string, statuses []types.ExamAssignmentStatus, limit int) ([]*types.ExamClassAssignment, error)
	ListAssignmentsByUserClasses(ctx context.Context, tenantID uint64, userID string, limit int) ([]*types.ExamClassAssignment, error)
	UpdateAssignmentMetadata(ctx context.Context, tenantID uint64, classID, assignmentID string, allowed []types.ExamAssignmentStatus, title, instructions string, dueAt *time.Time, updatedAt time.Time) error
	TransitionAssignmentStatus(ctx context.Context, tenantID uint64, classID, assignmentID string, expected, next types.ExamAssignmentStatus, updatedAt time.Time) error
}

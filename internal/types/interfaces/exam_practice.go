package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamPracticeService interface {
	ListQuestionGroups(ctx context.Context, tenantID uint64, userID string, filter types.ListPracticeQuestionGroupsFilter) ([]*types.QuestionGroupPracticeSummary, error)
	GetQuestionGroupDetail(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.QuestionGroupDetail, error)
	CreateAttempt(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.CreatePracticeAttemptResult, error)
	SubmitAnswer(ctx context.Context, tenantID uint64, userID string, attemptID string, req *types.SubmitPracticeAnswerRequest) (*types.PracticeAnswerResult, error)
	CompleteAttempt(ctx context.Context, tenantID uint64, userID string, attemptID string) (*types.ExamPracticeAttempt, error)
}

type ExamPracticeRepository interface {
	CreateAttempt(ctx context.Context, attempt *types.ExamPracticeAttempt) error
	GetAttemptByIDAndTenant(ctx context.Context, tenantID uint64, attemptID string) (*types.ExamPracticeAttempt, error)
	UpdateAttempt(ctx context.Context, attempt *types.ExamPracticeAttempt) error
	UpsertAnswer(ctx context.Context, answer *types.ExamPracticeAnswer) error
	ListAnswersByAttempt(ctx context.Context, tenantID uint64, attemptID string) ([]*types.ExamPracticeAnswer, error)
	ListLatestAttemptsByGroups(ctx context.Context, tenantID uint64, userID string, groupIDs []string) (map[string]*types.ExamPracticeAttempt, error)
}

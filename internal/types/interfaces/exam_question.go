package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamQuestionService interface {
	CreateQuestionBank(ctx context.Context, tenantID uint64, userID string, req *types.CreateQuestionBankRequest) (*types.QuestionBank, error)
	ListQuestionBanks(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]*types.QuestionBank, error)
	GetQuestionBank(ctx context.Context, tenantID uint64, userID string, bankID string) (*types.QuestionBank, error)
	ListQuestionDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionDetail, error)
	GetQuestionDetail(ctx context.Context, tenantID uint64, userID string, questionID string) (*types.QuestionDetail, error)
	ListQuestionGroupDetails(ctx context.Context, tenantID uint64, userID string, bankID string) ([]*types.QuestionGroupDetail, error)
	GetQuestionGroupDetail(ctx context.Context, tenantID uint64, userID string, groupID string) (*types.QuestionGroupDetail, error)
}

type ExamQuestionRepository interface {
	CreateQuestionBank(ctx context.Context, bank *types.QuestionBank) error
	ListQuestionBanks(ctx context.Context, tenantID uint64, spaceIDs []string) ([]*types.QuestionBank, error)
	GetQuestionBankByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.QuestionBank, error)
	CreateQuestionDetail(ctx context.Context, detail *types.QuestionDetail) error
	ListQuestionDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionDetail, error)
	GetQuestionDetailByIDAndTenant(ctx context.Context, tenantID uint64, questionID string) (*types.QuestionDetail, error)
	CreateQuestionGroupDetail(ctx context.Context, detail *types.QuestionGroupDetail) error
	ListQuestionGroupDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionGroupDetail, error)
	GetQuestionGroupDetailByIDAndTenant(ctx context.Context, tenantID uint64, groupID string) (*types.QuestionGroupDetail, error)
}

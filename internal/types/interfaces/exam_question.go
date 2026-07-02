package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamQuestionService interface {
	CreateQuestionBank(ctx context.Context, tenantID uint64, userID string, req *types.CreateQuestionBankRequest) (*types.QuestionBank, error)
	ListQuestionBanks(ctx context.Context, tenantID uint64, userID string, spaceID string) ([]*types.QuestionBank, error)
	GetQuestionBank(ctx context.Context, tenantID uint64, userID string, bankID string) (*types.QuestionBank, error)
}

type ExamQuestionRepository interface {
	CreateQuestionBank(ctx context.Context, bank *types.QuestionBank) error
	ListQuestionBanks(ctx context.Context, tenantID uint64, spaceIDs []string) ([]*types.QuestionBank, error)
	GetQuestionBankByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.QuestionBank, error)
}

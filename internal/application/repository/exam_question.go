package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrQuestionBankNotFound = errors.New("question bank not found")

type examQuestionRepository struct {
	db *gorm.DB
}

func NewExamQuestionRepository(db *gorm.DB) interfaces.ExamQuestionRepository {
	return &examQuestionRepository{db: db}
}

func (r *examQuestionRepository) CreateQuestionBank(ctx context.Context, bank *types.QuestionBank) error {
	return r.db.WithContext(ctx).Create(bank).Error
}

func (r *examQuestionRepository) ListQuestionBanks(ctx context.Context, tenantID uint64, spaceIDs []string) ([]*types.QuestionBank, error) {
	if len(spaceIDs) == 0 {
		return []*types.QuestionBank{}, nil
	}
	var banks []*types.QuestionBank
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND space_id IN ? AND status = ?", tenantID, spaceIDs, "active").
		Order("created_at DESC").
		Find(&banks).Error
	return banks, err
}

func (r *examQuestionRepository) GetQuestionBankByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.QuestionBank, error) {
	var bank types.QuestionBank
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status = ?", id, tenantID, "active").
		First(&bank).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionBankNotFound
		}
		return nil, err
	}
	return &bank, nil
}

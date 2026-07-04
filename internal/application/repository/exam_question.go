package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrQuestionBankNotFound = errors.New("question bank not found")
var ErrQuestionNotFound = errors.New("question not found")

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

func (r *examQuestionRepository) CreateQuestionDetail(ctx context.Context, detail *types.QuestionDetail) error {
	if detail == nil || detail.Question == nil {
		return errors.New("question detail is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(detail.Question).Error; err != nil {
			return err
		}
		if len(detail.Options) > 0 {
			if err := tx.Create(&detail.Options).Error; err != nil {
				return err
			}
		}
		if len(detail.Answers) > 0 {
			if err := tx.Create(&detail.Answers).Error; err != nil {
				return err
			}
		}
		if len(detail.Explanations) > 0 {
			if err := tx.Create(&detail.Explanations).Error; err != nil {
				return err
			}
		}
		if len(detail.ChunkRefs) > 0 {
			if err := tx.Create(&detail.ChunkRefs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *examQuestionRepository) ListQuestionDetailsByBank(ctx context.Context, tenantID uint64, bankID string) ([]*types.QuestionDetail, error) {
	var questions []*types.Question
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND question_bank_id = ? AND status <> ?", tenantID, bankID, "deleted").
		Order("created_at DESC").
		Find(&questions).Error
	if err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return []*types.QuestionDetail{}, nil
	}
	ids := make([]string, 0, len(questions))
	details := make([]*types.QuestionDetail, 0, len(questions))
	byID := make(map[string]*types.QuestionDetail, len(questions))
	for _, question := range questions {
		ids = append(ids, question.ID)
		detail := &types.QuestionDetail{Question: question}
		details = append(details, detail)
		byID[question.ID] = detail
	}
	if err := r.loadQuestionChildren(ctx, ids, byID); err != nil {
		return nil, err
	}
	return details, nil
}

func (r *examQuestionRepository) GetQuestionDetailByIDAndTenant(ctx context.Context, tenantID uint64, questionID string) (*types.QuestionDetail, error) {
	var question types.Question
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status <> ?", questionID, tenantID, "deleted").
		First(&question).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, err
	}
	detail := &types.QuestionDetail{Question: &question}
	if err := r.loadQuestionChildren(ctx, []string{question.ID}, map[string]*types.QuestionDetail{question.ID: detail}); err != nil {
		return nil, err
	}
	return detail, nil
}

func (r *examQuestionRepository) loadQuestionChildren(ctx context.Context, questionIDs []string, byID map[string]*types.QuestionDetail) error {
	var options []*types.QuestionOption
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("sort_order ASC, option_key ASC").
		Find(&options).Error; err != nil {
		return err
	}
	for _, item := range options {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Options = append(detail.Options, item)
		}
	}

	var answers []*types.QuestionAnswer
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&answers).Error; err != nil {
		return err
	}
	for _, item := range answers {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Answers = append(detail.Answers, item)
		}
	}

	var explanations []*types.QuestionExplanation
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&explanations).Error; err != nil {
		return err
	}
	for _, item := range explanations {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.Explanations = append(detail.Explanations, item)
		}
	}

	var refs []*types.QuestionChunkRef
	if err := r.db.WithContext(ctx).
		Where("question_id IN ?", questionIDs).
		Order("created_at ASC").
		Find(&refs).Error; err != nil {
		return err
	}
	for _, item := range refs {
		if detail := byID[item.QuestionID]; detail != nil {
			detail.ChunkRefs = append(detail.ChunkRefs, item)
		}
	}
	return nil
}

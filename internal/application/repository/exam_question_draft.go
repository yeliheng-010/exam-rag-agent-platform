package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamQuestionDraftNotFound = errors.New("exam question draft not found")

type examQuestionDraftRepository struct {
	db *gorm.DB
}

func NewExamQuestionDraftRepository(db *gorm.DB) interfaces.ExamQuestionDraftRepository {
	return &examQuestionDraftRepository{db: db}
}

func (r *examQuestionDraftRepository) CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionDraft) error {
	if len(drafts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&drafts).Error
}

func (r *examQuestionDraftRepository) DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Delete(&types.ExamQuestionDraft{}).Error
}

func (r *examQuestionDraftRepository) GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionDraft, error) {
	var draft types.ExamQuestionDraft
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&draft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamQuestionDraftNotFound
		}
		return nil, err
	}
	return &draft, nil
}

func (r *examQuestionDraftRepository) ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionDraft, error) {
	var drafts []*types.ExamQuestionDraft
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Order("question_no ASC, created_at ASC").
		Find(&drafts).Error
	return drafts, err
}

func (r *examQuestionDraftRepository) CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionDraftStats, error) {
	var rows []struct {
		Status types.ExamQuestionDraftStatus
		Count  int
	}
	err := r.db.WithContext(ctx).
		Model(&types.ExamQuestionDraft{}).
		Select("status, COUNT(*) AS count").
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return types.ExamQuestionDraftStats{}, err
	}

	stats := types.ExamQuestionDraftStats{}
	for _, row := range rows {
		stats.Total += row.Count
		switch row.Status {
		case types.ExamQuestionDraftStatusPendingReview:
			stats.PendingReview = row.Count
		case types.ExamQuestionDraftStatusApproved:
			stats.Approved = row.Count
		case types.ExamQuestionDraftStatusRejected:
			stats.Rejected = row.Count
		}
	}
	return stats, nil
}

func (r *examQuestionDraftRepository) UpdateDraft(ctx context.Context, draft *types.ExamQuestionDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}

package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrQuestionGroupDraftNotFound = errors.New("question group draft not found")

type examQuestionGroupDraftRepository struct {
	db *gorm.DB
}

func NewExamQuestionGroupDraftRepository(db *gorm.DB) interfaces.ExamQuestionGroupDraftRepository {
	return &examQuestionGroupDraftRepository{db: db}
}

func (r *examQuestionGroupDraftRepository) CreateDrafts(ctx context.Context, drafts []*types.ExamQuestionGroupDraft) error {
	if len(drafts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&drafts).Error
}

func (r *examQuestionGroupDraftRepository) DeleteDraftsByTask(ctx context.Context, tenantID uint64, taskID string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Delete(&types.ExamQuestionGroupDraft{}).Error
}

func (r *examQuestionGroupDraftRepository) GetDraftByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamQuestionGroupDraft, error) {
	var draft types.ExamQuestionGroupDraft
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&draft).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrQuestionGroupDraftNotFound
		}
		return nil, err
	}
	return &draft, nil
}

func (r *examQuestionGroupDraftRepository) ListDraftsByTask(ctx context.Context, tenantID uint64, taskID string) ([]*types.ExamQuestionGroupDraft, error) {
	var drafts []*types.ExamQuestionGroupDraft
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Order("created_at ASC").
		Find(&drafts).Error
	return drafts, err
}

func (r *examQuestionGroupDraftRepository) CountDraftsByTask(ctx context.Context, tenantID uint64, taskID string) (types.ExamQuestionGroupDraftStats, error) {
	var rows []struct {
		Status types.ExamQuestionGroupDraftStatus
		Count  int
	}
	err := r.db.WithContext(ctx).
		Model(&types.ExamQuestionGroupDraft{}).
		Select("status, COUNT(*) AS count").
		Where("tenant_id = ? AND task_id = ?", tenantID, taskID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return types.ExamQuestionGroupDraftStats{}, err
	}
	stats := types.ExamQuestionGroupDraftStats{}
	for _, row := range rows {
		stats.Total += row.Count
		switch row.Status {
		case types.ExamQuestionGroupDraftStatusPendingReview:
			stats.PendingReview = row.Count
		case types.ExamQuestionGroupDraftStatusApproved:
			stats.Approved = row.Count
		case types.ExamQuestionGroupDraftStatusRejected:
			stats.Rejected = row.Count
		}
	}
	return stats, nil
}

func (r *examQuestionGroupDraftRepository) UpdateDraft(ctx context.Context, draft *types.ExamQuestionGroupDraft) error {
	return r.db.WithContext(ctx).Save(draft).Error
}

package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

var ErrExamTeacherApplicationNotFound = errors.New("exam teacher application not found")

type examTeacherApplicationRepository struct {
	db *gorm.DB
}

func NewExamTeacherApplicationRepository(db *gorm.DB) interfaces.ExamTeacherApplicationRepository {
	return &examTeacherApplicationRepository{db: db}
}

func (r *examTeacherApplicationRepository) UpsertPending(ctx context.Context, app *types.ExamTeacherApplication) error {
	var existing types.ExamTeacherApplication
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", app.TenantID, app.UserID).
		First(&existing).Error
	if err == nil {
		return r.db.WithContext(ctx).
			Model(&existing).
			Updates(map[string]any{
				"status":      types.ExamTeacherApplicationStatusPending,
				"reason":      app.Reason,
				"reviewer_id": nil,
				"review_note": "",
				"reviewed_at": nil,
				"updated_at":  app.UpdatedAt,
			}).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(app).Error
	}
	return err
}

func (r *examTeacherApplicationRepository) GetByTenantAndUser(ctx context.Context, tenantID uint64, userID string) (*types.ExamTeacherApplication, error) {
	var app types.ExamTeacherApplication
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamTeacherApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *examTeacherApplicationRepository) GetByIDAndTenant(ctx context.Context, id string, tenantID uint64) (*types.ExamTeacherApplication, error) {
	var app types.ExamTeacherApplication
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&app).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExamTeacherApplicationNotFound
		}
		return nil, err
	}
	return &app, nil
}

func (r *examTeacherApplicationRepository) ListByTenant(ctx context.Context, tenantID uint64, status *types.ExamTeacherApplicationStatus) ([]*types.ExamTeacherApplication, error) {
	var apps []*types.ExamTeacherApplication
	q := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	err := q.Order("created_at DESC").Find(&apps).Error
	return apps, err
}

func (r *examTeacherApplicationRepository) UpdateReview(ctx context.Context, id string, tenantID uint64, status types.ExamTeacherApplicationStatus, reviewerID string, reviewNote string) (*types.ExamTeacherApplication, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&types.ExamTeacherApplication{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]any{
			"status":      status,
			"reviewer_id": reviewerID,
			"review_note": reviewNote,
			"reviewed_at": now,
			"updated_at":  now,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrExamTeacherApplicationNotFound
	}
	return r.GetByIDAndTenant(ctx, id, tenantID)
}

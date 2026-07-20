package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
)

var ErrExamAssignmentNotificationNotFound = errors.New("exam assignment notification not found")

type examAssignmentNotificationRepository struct {
	db *gorm.DB
}

func NewExamAssignmentNotificationRepository(db *gorm.DB) *examAssignmentNotificationRepository {
	return &examAssignmentNotificationRepository{db: db}
}

func (r *examAssignmentNotificationRepository) ListForRecipient(
	ctx context.Context,
	tenantID uint64,
	userID string,
	limit int,
) ([]*types.ExamAssignmentNotification, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	var unreadCount int64
	if err := r.db.WithContext(ctx).
		Model(&types.ExamAssignmentNotification{}).
		Where("tenant_id = ? AND recipient_user_id = ? AND read_at IS NULL", tenantID, userID).
		Count(&unreadCount).Error; err != nil {
		return nil, 0, err
	}
	var notifications []*types.ExamAssignmentNotification
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND recipient_user_id = ?", tenantID, userID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, unreadCount, err
}

func (r *examAssignmentNotificationRepository) MarkRead(
	ctx context.Context,
	tenantID uint64,
	userID string,
	notificationID string,
	readAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&types.ExamAssignmentNotification{}).
		Where("tenant_id = ? AND recipient_user_id = ? AND id = ?", tenantID, userID, notificationID).
		Update("read_at", gorm.Expr("COALESCE(read_at, ?)", readAt))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrExamAssignmentNotificationNotFound
	}
	return nil
}

func (r *examAssignmentNotificationRepository) MarkAllRead(
	ctx context.Context,
	tenantID uint64,
	userID string,
	readAt time.Time,
) error {
	return r.db.WithContext(ctx).
		Model(&types.ExamAssignmentNotification{}).
		Where("tenant_id = ? AND recipient_user_id = ? AND read_at IS NULL", tenantID, userID).
		Update("read_at", readAt).Error
}

func (r *examAssignmentNotificationRepository) ListLatestReminders(
	ctx context.Context,
	tenantID uint64,
	assignmentID string,
	userIDs []string,
) (map[string]time.Time, error) {
	out := make(map[string]time.Time, len(userIDs))
	if assignmentID == "" || len(userIDs) == 0 {
		return out, nil
	}
	var notifications []*types.ExamAssignmentNotification
	err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND assignment_id = ? AND recipient_user_id IN ? AND kind = ?",
			tenantID, assignmentID, userIDs, types.ExamAssignmentNotificationKindReminder,
		).
		Order("created_at DESC, id DESC").
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		if _, exists := out[notification.RecipientUserID]; !exists {
			out[notification.RecipientUserID] = notification.CreatedAt
		}
	}
	return out, nil
}

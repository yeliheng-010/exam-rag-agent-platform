package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrExamAssignmentNotificationNotFound = errors.New("exam assignment notification not found")

type examAssignmentNotificationRepository struct {
	db *gorm.DB
}

func NewExamAssignmentNotificationRepository(db *gorm.DB) interfaces.ExamAssignmentNotificationRepository {
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

func (r *examAssignmentNotificationRepository) CreateRemindersIfEligible(
	ctx context.Context,
	tenantID uint64,
	assignmentID string,
	candidates []*types.ExamAssignmentNotification,
	now time.Time,
) (*types.SendExamAssignmentReminderResult, error) {
	result := &types.SendExamAssignmentReminderResult{SentUserIDs: []string{}}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockOpenAssignment(tx, tenantID, assignmentID, now); err != nil {
			return err
		}
		candidateByUser, userIDs, err := reminderCandidates(tenantID, assignmentID, candidates)
		if err != nil {
			return err
		}
		attempts, err := latestAssignmentAttempts(tx, tenantID, assignmentID, userIDs)
		if err != nil {
			return err
		}
		recent, err := recentAssignmentReminders(tx, tenantID, assignmentID, userIDs, now.Add(-24*time.Hour))
		if err != nil {
			return err
		}
		toCreate := eligibleReminders(candidateByUser, userIDs, attempts, recent, result)
		if len(toCreate) == 0 {
			return nil
		}
		return tx.Create(toCreate).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func lockOpenAssignment(tx *gorm.DB, tenantID uint64, assignmentID string, now time.Time) error {
	var assignment types.ExamClassAssignment
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where(
			"tenant_id = ? AND id = ? AND status = ? AND (due_at IS NULL OR due_at > ?)",
			tenantID, assignmentID, types.ExamAssignmentStatusPublished, now,
		).
		First(&assignment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrExamClassAssignmentStateConflict
	}
	return err
}

func reminderCandidates(
	tenantID uint64,
	assignmentID string,
	candidates []*types.ExamAssignmentNotification,
) (map[string]*types.ExamAssignmentNotification, []string, error) {
	byUser := make(map[string]*types.ExamAssignmentNotification, len(candidates))
	userIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.RecipientUserID == "" {
			continue
		}
		if candidate.TenantID != tenantID || candidate.AssignmentID != assignmentID || candidate.Kind != types.ExamAssignmentNotificationKindReminder {
			return nil, nil, ErrExamClassAssignmentStateConflict
		}
		if _, exists := byUser[candidate.RecipientUserID]; exists {
			continue
		}
		byUser[candidate.RecipientUserID] = candidate
		userIDs = append(userIDs, candidate.RecipientUserID)
	}
	return byUser, userIDs, nil
}

func latestAssignmentAttempts(
	tx *gorm.DB,
	tenantID uint64,
	assignmentID string,
	userIDs []string,
) (map[string]*types.ExamPracticeAttempt, error) {
	out := make(map[string]*types.ExamPracticeAttempt, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var attempts []*types.ExamPracticeAttempt
	err := tx.Where("tenant_id = ? AND assignment_id = ? AND user_id IN ?", tenantID, assignmentID, userIDs).
		Order("created_at DESC, id DESC").Find(&attempts).Error
	for _, attempt := range attempts {
		if attempt != nil && out[attempt.UserID] == nil {
			out[attempt.UserID] = attempt
		}
	}
	return out, err
}

func recentAssignmentReminders(
	tx *gorm.DB,
	tenantID uint64,
	assignmentID string,
	userIDs []string,
	since time.Time,
) (map[string]bool, error) {
	out := make(map[string]bool, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var notifications []*types.ExamAssignmentNotification
	err := tx.Where(
		"tenant_id = ? AND assignment_id = ? AND recipient_user_id IN ? AND kind = ? AND created_at > ?",
		tenantID, assignmentID, userIDs, types.ExamAssignmentNotificationKindReminder, since,
	).Find(&notifications).Error
	for _, notification := range notifications {
		if notification != nil {
			out[notification.RecipientUserID] = true
		}
	}
	return out, err
}

func eligibleReminders(
	candidates map[string]*types.ExamAssignmentNotification,
	userIDs []string,
	attempts map[string]*types.ExamPracticeAttempt,
	recent map[string]bool,
	result *types.SendExamAssignmentReminderResult,
) []*types.ExamAssignmentNotification {
	items := make([]*types.ExamAssignmentNotification, 0, len(userIDs))
	for _, userID := range userIDs {
		if attempt := attempts[userID]; attempt != nil && attempt.Status == types.ExamPracticeAttemptStatusCompleted {
			result.CompletedSkippedCount++
			continue
		}
		if recent[userID] {
			result.CooldownSkippedCount++
			continue
		}
		items = append(items, candidates[userID])
		result.SentUserIDs = append(result.SentUserIDs, userID)
		result.SentCount++
	}
	return items
}

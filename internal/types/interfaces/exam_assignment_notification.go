package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type ExamAssignmentNotificationRepository interface {
	ListForRecipient(ctx context.Context, tenantID uint64, userID string, limit int) ([]*types.ExamAssignmentNotification, int64, error)
	MarkRead(ctx context.Context, tenantID uint64, userID, notificationID string, readAt time.Time) error
	MarkAllRead(ctx context.Context, tenantID uint64, userID string, readAt time.Time) error
	ListLatestReminders(ctx context.Context, tenantID uint64, assignmentID string, userIDs []string) (map[string]time.Time, error)
	CreateRemindersIfEligible(ctx context.Context, tenantID uint64, assignmentID string, candidates []*types.ExamAssignmentNotification, now time.Time) (*types.SendExamAssignmentReminderResult, error)
}

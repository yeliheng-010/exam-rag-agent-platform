package types

import "time"

type ExamAssignmentNotificationKind string

const (
	ExamAssignmentNotificationKindPublished   ExamAssignmentNotificationKind = "published"
	ExamAssignmentNotificationKindRepublished ExamAssignmentNotificationKind = "republished"
	ExamAssignmentNotificationKindWithdrawn   ExamAssignmentNotificationKind = "withdrawn"
	ExamAssignmentNotificationKindReminder    ExamAssignmentNotificationKind = "reminder"
)

type ExamAssignmentNotification struct {
	ID              string                         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64                         `json:"tenant_id" gorm:"not null;index"`
	ClassID         string                         `json:"class_id" gorm:"type:varchar(36);not null;index"`
	AssignmentID    string                         `json:"assignment_id" gorm:"type:varchar(36);not null;index"`
	GroupID         string                         `json:"group_id" gorm:"type:varchar(36);not null;index"`
	RecipientUserID string                         `json:"recipient_user_id" gorm:"type:varchar(36);not null;index"`
	ActorUserID     string                         `json:"actor_user_id" gorm:"type:varchar(36);not null;index"`
	Kind            ExamAssignmentNotificationKind `json:"kind" gorm:"type:varchar(32);not null"`
	Title           string                         `json:"title" gorm:"type:varchar(255);not null"`
	Content         string                         `json:"content" gorm:"type:text;not null"`
	ReadAt          *time.Time                     `json:"read_at,omitempty"`
	CreatedAt       time.Time                      `json:"created_at"`
}

func (ExamAssignmentNotification) TableName() string {
	return "exam_assignment_notifications"
}

type ExamAssignmentNotificationItem struct {
	Notification     *ExamAssignmentNotification `json:"notification"`
	LastAttemptID    string                      `json:"last_attempt_id,omitempty"`
	AssignmentStatus ExamAssignmentStatus        `json:"assignment_status"`
	AssignmentDueAt  *time.Time                  `json:"assignment_due_at,omitempty"`
	CanStart         bool                        `json:"can_start"`
}

type ExamAssignmentNotificationList struct {
	Items       []*ExamAssignmentNotificationItem `json:"items"`
	UnreadCount int64                             `json:"unread_count"`
}

type SendExamAssignmentReminderRequest struct {
	RecipientUserIDs []string `json:"recipient_user_ids" binding:"omitempty,max=100,dive,required"`
}

type SendExamAssignmentReminderResult struct {
	SentCount             int      `json:"sent_count"`
	CompletedSkippedCount int      `json:"completed_skipped_count"`
	CooldownSkippedCount  int      `json:"cooldown_skipped_count"`
	SentUserIDs           []string `json:"sent_user_ids"`
}

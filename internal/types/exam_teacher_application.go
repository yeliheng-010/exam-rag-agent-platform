package types

import "time"

type ExamTeacherApplicationStatus string

const (
	ExamTeacherApplicationStatusPending  ExamTeacherApplicationStatus = "pending"
	ExamTeacherApplicationStatusApproved ExamTeacherApplicationStatus = "approved"
	ExamTeacherApplicationStatusRejected ExamTeacherApplicationStatus = "rejected"
)

type ExamTeacherApplication struct {
	ID         string                       `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID   uint64                       `json:"tenant_id" gorm:"not null;index"`
	UserID     string                       `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Status     ExamTeacherApplicationStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending';index"`
	Reason     string                       `json:"reason" gorm:"type:text;not null;default:''"`
	ReviewerID *string                      `json:"reviewer_id,omitempty" gorm:"type:varchar(36);index"`
	ReviewNote string                       `json:"review_note" gorm:"type:text;not null;default:''"`
	ReviewedAt *time.Time                   `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time                    `json:"created_at"`
	UpdatedAt  time.Time                    `json:"updated_at"`
}

func (ExamTeacherApplication) TableName() string {
	return "exam_teacher_applications"
}

type ExamTeacherApplicationView struct {
	*ExamTeacherApplication
	UserEmail    string `json:"user_email,omitempty"`
	Username     string `json:"username,omitempty"`
	ReviewerName string `json:"reviewer_name,omitempty"`
}

type ApplyExamTeacherRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=1000"`
}

type ReviewExamTeacherApplicationRequest struct {
	ReviewNote string `json:"review_note" binding:"omitempty,max=1000"`
}

package types

import "time"

type ExamAssignmentStatus string

const (
	ExamAssignmentStatusPublished ExamAssignmentStatus = "published"
	ExamAssignmentStatusArchived  ExamAssignmentStatus = "archived"
)

type ExamClassAssignment struct {
	ID              string               `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64               `json:"tenant_id" gorm:"not null;index"`
	ClassID         string               `json:"class_id" gorm:"type:varchar(36);not null;index"`
	SpaceID         string               `json:"space_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID  string               `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	GroupID         string               `json:"group_id" gorm:"type:varchar(36);not null;index"`
	Title           string               `json:"title" gorm:"type:varchar(255);not null"`
	Instructions    string               `json:"instructions" gorm:"type:text;not null;default:''"`
	Status          ExamAssignmentStatus `json:"status" gorm:"type:varchar(32);not null;default:'published'"`
	DueAt           *time.Time           `json:"due_at,omitempty"`
	CreatedByUserID string               `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

func (ExamClassAssignment) TableName() string {
	return "exam_class_assignments"
}

type CreateExamAssignmentRequest struct {
	GroupID      string     `json:"group_id" binding:"required"`
	Title        string     `json:"title" binding:"omitempty,max=255"`
	Instructions string     `json:"instructions" binding:"omitempty,max=2000"`
	DueAt        *time.Time `json:"due_at"`
}

type ListExamAssignmentsFilter struct {
	Limit int
}

type ExamAssignmentSummary struct {
	Assignment    *ExamClassAssignment `json:"assignment"`
	Group         *QuestionGroup       `json:"group,omitempty"`
	BankName      string               `json:"bank_name"`
	QuestionCount int                  `json:"question_count"`
	LastAttempt   *ExamPracticeAttempt `json:"last_attempt,omitempty"`
}

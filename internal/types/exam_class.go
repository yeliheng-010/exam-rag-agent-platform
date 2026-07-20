package types

import "time"

type ExamClassRole string

const (
	ExamClassRoleTeacher   ExamClassRole = "teacher"
	ExamClassRoleAssistant ExamClassRole = "assistant"
	ExamClassRoleStudent   ExamClassRole = "student"
)

func (r ExamClassRole) CanWrite() bool {
	return r == ExamClassRoleTeacher || r == ExamClassRoleAssistant
}

type ExamClassStatus string

const (
	ExamClassStatusActive   ExamClassStatus = "active"
	ExamClassStatusArchived ExamClassStatus = "archived"
)

type ExamClassMemberStatus string

const (
	ExamClassMemberStatusPending ExamClassMemberStatus = "pending"
	ExamClassMemberStatusActive  ExamClassMemberStatus = "active"
	ExamClassMemberStatusRemoved ExamClassMemberStatus = "removed"
)

type ExamClass struct {
	ID                  string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID            uint64          `json:"tenant_id" gorm:"not null;index"`
	OwnerUserID         string          `json:"owner_user_id" gorm:"type:varchar(36);not null;index"`
	SpaceID             string          `json:"space_id" gorm:"type:varchar(36);not null;index"`
	DomainID            *string         `json:"domain_id,omitempty" gorm:"type:varchar(36);index"`
	Name                string          `json:"name" gorm:"type:varchar(255);not null"`
	Description         string          `json:"description" gorm:"type:text;not null;default:''"`
	InviteCode          *string         `json:"invite_code,omitempty" gorm:"type:varchar(32);uniqueIndex"`
	InviteCodeExpiresAt *time.Time      `json:"invite_code_expires_at,omitempty"`
	MemberLimit         int             `json:"member_limit" gorm:"not null;default:50"`
	Status              ExamClassStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

func (ExamClass) TableName() string {
	return "exam_classes"
}

type ExamClassMember struct {
	ID        string                `json:"id" gorm:"type:varchar(36);primaryKey"`
	ClassID   string                `json:"class_id" gorm:"type:varchar(36);not null;index"`
	UserID    string                `json:"user_id" gorm:"type:varchar(36);not null;index"`
	TenantID  uint64                `json:"tenant_id" gorm:"not null;index"`
	Role      ExamClassRole         `json:"role" gorm:"type:varchar(32);not null;default:'student'"`
	Status    ExamClassMemberStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	JoinedAt  time.Time             `json:"joined_at"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`

	DisplayID   string `json:"display_id,omitempty" gorm:"-"`
	DisplayName string `json:"display_name,omitempty" gorm:"-"`
}

func (ExamClassMember) TableName() string {
	return "exam_class_members"
}

type CreateExamClassRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
	DomainID    *string `json:"domain_id"`
	MemberLimit *int    `json:"member_limit"`
}

type UpdateExamClassRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberLimit int    `json:"member_limit"`
}

type ListExamClassesFilter struct {
	IncludeArchived bool
}

type JoinExamClassRequest struct {
	InviteCode string `json:"invite_code" binding:"required,min=1,max=32"`
}

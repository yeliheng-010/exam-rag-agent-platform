package types

import "time"

type ExamSpaceType string

const (
	ExamSpaceTypePublic   ExamSpaceType = "public"
	ExamSpaceTypePersonal ExamSpaceType = "personal"
	ExamSpaceTypeClass    ExamSpaceType = "class"
)

type ExamSpaceStatus string

const (
	ExamSpaceStatusActive   ExamSpaceStatus = "active"
	ExamSpaceStatusArchived ExamSpaceStatus = "archived"
)

type ExamSpace struct {
	ID          string          `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID    uint64          `json:"tenant_id" gorm:"not null;index"`
	OwnerUserID *string         `json:"owner_user_id,omitempty" gorm:"type:varchar(36);index"`
	SpaceType   ExamSpaceType   `json:"space_type" gorm:"type:varchar(32);not null;index"`
	Name        string          `json:"name" gorm:"type:varchar(255);not null"`
	Description string          `json:"description" gorm:"type:text;not null;default:''"`
	Status      ExamSpaceStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (ExamSpace) TableName() string {
	return "exam_spaces"
}

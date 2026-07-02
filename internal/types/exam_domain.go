package types

import "time"

type ExamDomainStatus string

const (
	ExamDomainStatusActive   ExamDomainStatus = "active"
	ExamDomainStatusDisabled ExamDomainStatus = "disabled"
)

type ExamDomain struct {
	ID          string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	Code        string           `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Name        string           `json:"name" gorm:"type:varchar(255);not null"`
	Description string           `json:"description" gorm:"type:text;not null;default:''"`
	Status      ExamDomainStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func (ExamDomain) TableName() string {
	return "exam_domains"
}

type ExamSubject struct {
	ID        string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	DomainID  string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	Code      string           `json:"code" gorm:"type:varchar(64);not null"`
	Name      string           `json:"name" gorm:"type:varchar(255);not null"`
	SortOrder int              `json:"sort_order" gorm:"not null;default:0"`
	Status    ExamDomainStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func (ExamSubject) TableName() string {
	return "exam_subjects"
}

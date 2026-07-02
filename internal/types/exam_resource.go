package types

import "time"

type ExamResourceType string

const (
	ExamResourceTypeKnowledgeBase ExamResourceType = "knowledge_base"
)

type ExamMaterialType string

const (
	ExamMaterialTypeLearningMaterial ExamMaterialType = "learning_material"
	ExamMaterialTypeExamPaper        ExamMaterialType = "exam_paper"
	ExamMaterialTypeAnswerKey        ExamMaterialType = "answer_key"
	ExamMaterialTypeExplanation      ExamMaterialType = "explanation"
)

type ExamSpaceResourceStatus string

const (
	ExamSpaceResourceStatusActive   ExamSpaceResourceStatus = "active"
	ExamSpaceResourceStatusArchived ExamSpaceResourceStatus = "archived"
)

type ExamSpaceResource struct {
	ID              string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64                  `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string                  `json:"space_id" gorm:"type:varchar(36);not null;index"`
	ResourceType    ExamResourceType        `json:"resource_type" gorm:"type:varchar(64);not null;index"`
	ResourceID      string                  `json:"resource_id" gorm:"type:varchar(36);not null;index"`
	DomainID        string                  `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       *string                 `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	MaterialType    ExamMaterialType        `json:"material_type" gorm:"type:varchar(64);not null;default:'learning_material'"`
	ReviewStatus    ExamReviewStatus        `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status          ExamSpaceResourceStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string                  `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func (ExamSpaceResource) TableName() string {
	return "exam_space_resources"
}

type BindKnowledgeBaseResourceRequest struct {
	SpaceID      string           `json:"space_id" binding:"required"`
	DomainID     string           `json:"domain_id" binding:"required"`
	SubjectID    *string          `json:"subject_id"`
	MaterialType ExamMaterialType `json:"material_type"`
}

type ListExamResourcesFilter struct {
	ResourceType ExamResourceType
	SpaceID      string
	DomainID     string
	SubjectID    string
	MaterialType ExamMaterialType
}

package types

import "time"

type ExamMaterialIngestStatus string

const (
	ExamMaterialIngestStatusPending    ExamMaterialIngestStatus = "pending"
	ExamMaterialIngestStatusProcessing ExamMaterialIngestStatus = "processing"
	ExamMaterialIngestStatusCompleted  ExamMaterialIngestStatus = "completed"
	ExamMaterialIngestStatusFailed     ExamMaterialIngestStatus = "failed"
	ExamMaterialIngestStatusCancelled  ExamMaterialIngestStatus = "cancelled"
	ExamMaterialIngestStatusUnknown    ExamMaterialIngestStatus = "unknown"
)

type ExamStructuringTaskStatus string

const (
	ExamStructuringTaskStatusPending        ExamStructuringTaskStatus = "pending"
	ExamStructuringTaskStatusReadyForReview ExamStructuringTaskStatus = "ready_for_review"
	ExamStructuringTaskStatusBlocked        ExamStructuringTaskStatus = "blocked"
	ExamStructuringTaskStatusExtracting     ExamStructuringTaskStatus = "extracting"
	ExamStructuringTaskStatusReviewing      ExamStructuringTaskStatus = "reviewing"
	ExamStructuringTaskStatusCompleted      ExamStructuringTaskStatus = "completed"
	ExamStructuringTaskStatusFailed         ExamStructuringTaskStatus = "failed"
)

type ExamStructuringStrategy string

const (
	ExamStructuringStrategyManualReview ExamStructuringStrategy = "manual_review"
)

type ExamMaterial struct {
	ID              string                   `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64                   `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string                   `json:"space_id" gorm:"type:varchar(36);not null;index"`
	KnowledgeBaseID string                   `json:"knowledge_base_id" gorm:"type:varchar(36);not null;index"`
	KnowledgeID     string                   `json:"knowledge_id" gorm:"type:varchar(36);not null;index"`
	DomainID        string                   `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       *string                  `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	MaterialType    ExamMaterialType         `json:"material_type" gorm:"type:varchar(64);not null;default:'learning_material'"`
	Title           string                   `json:"title" gorm:"type:varchar(255);not null"`
	Description     string                   `json:"description" gorm:"type:text;not null;default:''"`
	SourceYear      *int                     `json:"source_year,omitempty"`
	SourceRegion    string                   `json:"source_region" gorm:"type:varchar(128);not null;default:''"`
	PaperType       string                   `json:"paper_type" gorm:"type:varchar(128);not null;default:''"`
	IngestStatus    ExamMaterialIngestStatus `json:"ingest_status" gorm:"type:varchar(32);not null;default:'unknown'"`
	ReviewStatus    ExamReviewStatus         `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status          string                   `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string                   `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

func (ExamMaterial) TableName() string {
	return "exam_materials"
}

type ExamStructuringTask struct {
	ID                      string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID                uint64                    `json:"tenant_id" gorm:"not null;index"`
	MaterialID              string                    `json:"material_id" gorm:"type:varchar(36);not null;index"`
	SpaceID                 string                    `json:"space_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID          string                    `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	Status                  ExamStructuringTaskStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending'"`
	Strategy                ExamStructuringStrategy   `json:"strategy" gorm:"type:varchar(64);not null;default:'manual_review'"`
	SourceChunkCount        int                       `json:"source_chunk_count" gorm:"not null;default:0"`
	StructuredQuestionCount int                       `json:"structured_question_count" gorm:"not null;default:0"`
	ReviewRequired          bool                      `json:"review_required" gorm:"not null;default:true"`
	ErrorMessage            string                    `json:"error_message" gorm:"type:text;not null;default:''"`
	Progress                ExamStructuringProgress   `json:"progress" gorm:"type:jsonb;not null;default:'{}'"`
	CreatedByUserID         string                    `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt               time.Time                 `json:"created_at"`
	UpdatedAt               time.Time                 `json:"updated_at"`
}

func (ExamStructuringTask) TableName() string {
	return "exam_structuring_tasks"
}

type RegisterExamMaterialRequest struct {
	SpaceID         string           `json:"space_id" binding:"required"`
	KnowledgeBaseID string           `json:"knowledge_base_id" binding:"required"`
	KnowledgeID     string           `json:"knowledge_id" binding:"required"`
	DomainID        string           `json:"domain_id" binding:"required"`
	SubjectID       *string          `json:"subject_id"`
	MaterialType    ExamMaterialType `json:"material_type"`
	Title           string           `json:"title" binding:"omitempty,max=255"`
	Description     string           `json:"description" binding:"omitempty,max=1000"`
	SourceYear      *int             `json:"source_year"`
	SourceRegion    string           `json:"source_region" binding:"omitempty,max=128"`
	PaperType       string           `json:"paper_type" binding:"omitempty,max=128"`
	QuestionBankID  string           `json:"question_bank_id"`
	CreateTask      bool             `json:"create_task"`
}

type ExamMaterialRegistrationResult struct {
	Material        *ExamMaterial        `json:"material"`
	StructuringTask *ExamStructuringTask `json:"structuring_task,omitempty"`
}

type ListExamMaterialsFilter struct {
	SpaceID      string
	DomainID     string
	SubjectID    string
	MaterialType ExamMaterialType
}

type CreateExamStructuringTaskRequest struct {
	QuestionBankID string                  `json:"question_bank_id"`
	Strategy       ExamStructuringStrategy `json:"strategy"`
}

type ListExamStructuringTasksFilter struct {
	SpaceID    string
	MaterialID string
	Status     ExamStructuringTaskStatus
}

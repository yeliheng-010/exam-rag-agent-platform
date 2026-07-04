package types

import "time"

type ExamQuestionDraftStatus string

const (
	ExamQuestionDraftStatusPendingReview ExamQuestionDraftStatus = "pending_review"
	ExamQuestionDraftStatusApproved      ExamQuestionDraftStatus = "approved"
	ExamQuestionDraftStatusRejected      ExamQuestionDraftStatus = "rejected"
)

type ExamQuestionDraftOption struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type ExamQuestionDraft struct {
	ID                 string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64                  `json:"tenant_id" gorm:"not null;index"`
	SpaceID            string                  `json:"space_id" gorm:"type:varchar(36);not null;index"`
	TaskID             string                  `json:"task_id" gorm:"type:varchar(36);not null;index"`
	MaterialID         string                  `json:"material_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID     string                  `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID           string                  `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID          *string                 `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	SourceChunkIDs     JSON                    `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	QuestionNo         string                  `json:"question_no" gorm:"type:varchar(64);not null;default:''"`
	QuestionTypeCode   string                  `json:"question_type_code" gorm:"type:varchar(64);not null;default:''"`
	Stem               string                  `json:"stem" gorm:"type:text;not null"`
	OptionsJSON        JSON                    `json:"options_json" gorm:"type:jsonb;not null"`
	AnswerJSON         JSONMap                 `json:"answer_json" gorm:"type:jsonb;not null"`
	Explanation        string                  `json:"explanation" gorm:"type:text;not null;default:''"`
	Difficulty         string                  `json:"difficulty" gorm:"type:varchar(32);not null;default:'unknown'"`
	Confidence         float64                 `json:"confidence" gorm:"type:numeric(5,4);not null;default:0"`
	Status             ExamQuestionDraftStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending_review'"`
	RawModelOutput     string                  `json:"raw_model_output" gorm:"type:text;not null;default:''"`
	ErrorMessage       string                  `json:"error_message" gorm:"type:text;not null;default:''"`
	ApprovedQuestionID string                  `json:"approved_question_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedByUserID   string                  `json:"reviewed_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedAt         *time.Time              `json:"reviewed_at,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

func (ExamQuestionDraft) TableName() string {
	return "exam_question_drafts"
}

type ExtractExamQuestionDraftsRequest struct {
	Force bool `json:"force"`
}

type ExamQuestionDraftCandidate struct {
	QuestionNo       string                    `json:"question_no"`
	QuestionTypeCode string                    `json:"question_type_code"`
	Stem             string                    `json:"stem"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Difficulty       string                    `json:"difficulty"`
	Confidence       float64                   `json:"confidence"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
	RawModelOutput   string                    `json:"raw_model_output"`
}

type ExamQuestionDraftExtractionResult struct {
	Task   *ExamStructuringTask   `json:"task"`
	Drafts []*ExamQuestionDraft   `json:"drafts"`
	Stats  ExamQuestionDraftStats `json:"stats"`
}

type ExamQuestionDraftStats struct {
	Total         int `json:"total"`
	PendingReview int `json:"pending_review"`
	Approved      int `json:"approved"`
	Rejected      int `json:"rejected"`
}

type ListExamQuestionDraftsResult struct {
	Task   *ExamStructuringTask   `json:"task"`
	Drafts []*ExamQuestionDraft   `json:"drafts"`
	Stats  ExamQuestionDraftStats `json:"stats"`
}

type UpdateExamQuestionDraftRequest struct {
	QuestionNo       string                    `json:"question_no" binding:"omitempty,max=64"`
	QuestionTypeCode string                    `json:"question_type_code" binding:"omitempty,max=64"`
	Stem             string                    `json:"stem" binding:"required"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Difficulty       string                    `json:"difficulty"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
}

type ApproveExamQuestionDraftResult struct {
	Draft    *ExamQuestionDraft `json:"draft"`
	Question *QuestionDetail    `json:"question"`
}

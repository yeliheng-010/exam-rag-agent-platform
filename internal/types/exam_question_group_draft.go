package types

import "time"

type ExamQuestionGroupDraftStatus string

const (
	ExamQuestionGroupDraftStatusPendingReview ExamQuestionGroupDraftStatus = "pending_review"
	ExamQuestionGroupDraftStatusApproved      ExamQuestionGroupDraftStatus = "approved"
	ExamQuestionGroupDraftStatusRejected      ExamQuestionGroupDraftStatus = "rejected"
)

type ExamQuestionGroupDraft struct {
	ID               string                         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64                         `json:"tenant_id" gorm:"not null;index"`
	SpaceID          string                         `json:"space_id" gorm:"type:varchar(36);not null;index"`
	TaskID           string                         `json:"task_id" gorm:"type:varchar(36);not null;index"`
	MaterialID       string                         `json:"material_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID   string                         `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID         string                         `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID        *string                        `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	GroupType        string                         `json:"group_type" gorm:"type:varchar(64);not null"`
	Title            string                         `json:"title" gorm:"type:varchar(255);not null;default:''"`
	MaterialText     string                         `json:"material_text" gorm:"type:text;not null;default:''"`
	MaterialFormat   string                         `json:"material_format" gorm:"type:varchar(32);not null;default:'plain_text'"`
	QuestionsJSON    JSON                           `json:"questions_json" gorm:"type:jsonb;not null"`
	AssetsJSON       JSON                           `json:"assets_json" gorm:"type:jsonb;not null"`
	SourceChunkIDs   JSON                           `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	StrategyCode     string                         `json:"strategy_code" gorm:"type:varchar(64);not null;default:''"`
	Confidence       float64                        `json:"confidence" gorm:"type:numeric(5,4);not null;default:0"`
	Status           ExamQuestionGroupDraftStatus   `json:"status" gorm:"type:varchar(32);not null;default:'pending_review'"`
	RawModelOutput   string                         `json:"raw_model_output" gorm:"type:text;not null;default:''"`
	ErrorMessage     string                         `json:"error_message" gorm:"type:text;not null;default:''"`
	ApprovedGroupID  string                         `json:"approved_group_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedByUserID string                         `json:"reviewed_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	ReviewedAt       *time.Time                     `json:"reviewed_at,omitempty"`
	QualityReport    ExamQuestionGroupQualityReport `json:"quality_report" gorm:"-"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
}

func (ExamQuestionGroupDraft) TableName() string {
	return "exam_question_group_drafts"
}

type ExamQuestionGroupDraftQuestionCandidate struct {
	QuestionNo       string                    `json:"question_no"`
	QuestionTypeCode string                    `json:"question_type_code"`
	Stem             string                    `json:"stem"`
	Options          []ExamQuestionDraftOption `json:"options"`
	Answer           JSONMap                   `json:"answer"`
	Explanation      string                    `json:"explanation"`
	Evidence         []JSONMap                 `json:"evidence"`
	Metadata         JSONMap                   `json:"metadata"`
	Difficulty       string                    `json:"difficulty"`
	Confidence       float64                   `json:"confidence"`
	OrderInGroup     int                       `json:"order_in_group"`
	SourceChunkIDs   []string                  `json:"source_chunk_ids"`
}

type ExamQuestionGroupDraftAssetCandidate struct {
	AssetType     string  `json:"asset_type"`
	StorageURI    string  `json:"storage_uri"`
	AltText       string  `json:"alt_text"`
	SourceChunkID string  `json:"source_chunk_id"`
	BBox          JSONMap `json:"bbox"`
	Metadata      JSONMap `json:"metadata"`
	SortOrder     int     `json:"sort_order"`
}

type ExamQuestionGroupDraftCandidate struct {
	GroupNo        string                                    `json:"group_no"`
	GroupType      string                                    `json:"group_type"`
	Title          string                                    `json:"title"`
	MaterialText   string                                    `json:"material_text"`
	MaterialFormat string                                    `json:"material_format"`
	SourceChunkIDs []string                                  `json:"source_chunk_ids"`
	Assets         []ExamQuestionGroupDraftAssetCandidate    `json:"assets"`
	Questions      []ExamQuestionGroupDraftQuestionCandidate `json:"questions"`
	StrategyCode   string                                    `json:"strategy_code"`
	Confidence     float64                                   `json:"confidence"`
	RawModelOutput string                                    `json:"raw_model_output"`
}

type ExtractExamQuestionGroupDraftsRequest struct {
	Force bool `json:"force"`
}

type ExamQuestionGroupDraftStats struct {
	Total         int `json:"total"`
	PendingReview int `json:"pending_review"`
	Approved      int `json:"approved"`
	Rejected      int `json:"rejected"`
}

type ListExamQuestionGroupDraftsResult struct {
	Task   *ExamStructuringTask        `json:"task"`
	Drafts []*ExamQuestionGroupDraft   `json:"drafts"`
	Stats  ExamQuestionGroupDraftStats `json:"stats"`
}

type ExamQuestionGroupDraftExtractionResult = ListExamQuestionGroupDraftsResult

type UpdateExamQuestionGroupDraftRequest struct {
	GroupType      string                                    `json:"group_type" binding:"required"`
	Title          string                                    `json:"title" binding:"omitempty,max=255"`
	MaterialText   string                                    `json:"material_text"`
	MaterialFormat string                                    `json:"material_format"`
	Questions      []ExamQuestionGroupDraftQuestionCandidate `json:"questions" binding:"required"`
	Assets         []ExamQuestionGroupDraftAssetCandidate    `json:"assets"`
	SourceChunkIDs []string                                  `json:"source_chunk_ids"`
}

type ApproveExamQuestionGroupDraftResult struct {
	Draft *ExamQuestionGroupDraft `json:"draft"`
	Group *QuestionGroupDetail    `json:"group"`
}

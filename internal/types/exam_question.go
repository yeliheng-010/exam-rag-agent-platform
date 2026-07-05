package types

import "time"

type ExamReviewStatus string

const (
	ExamReviewStatusPrivate  ExamReviewStatus = "private"
	ExamReviewStatusPending  ExamReviewStatus = "pending"
	ExamReviewStatusApproved ExamReviewStatus = "approved"
	ExamReviewStatusRejected ExamReviewStatus = "rejected"
)

type QuestionBank struct {
	ID              string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64           `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string           `json:"space_id" gorm:"type:varchar(36);not null;index"`
	DomainID        string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	Name            string           `json:"name" gorm:"type:varchar(255);not null"`
	Description     string           `json:"description" gorm:"type:text;not null;default:''"`
	SourceType      string           `json:"source_type" gorm:"type:varchar(64);not null;default:'manual'"`
	ReviewStatus    ExamReviewStatus `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status          string           `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string           `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (QuestionBank) TableName() string {
	return "question_banks"
}

type QuestionType struct {
	ID         string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	DomainID   string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID  *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	Code       string           `json:"code" gorm:"type:varchar(64);not null"`
	Name       string           `json:"name" gorm:"type:varchar(255);not null"`
	AnswerMode string           `json:"answer_mode" gorm:"type:varchar(64);not null;default:'objective'"`
	SortOrder  int              `json:"sort_order" gorm:"not null;default:0"`
	Status     ExamDomainStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

func (QuestionType) TableName() string {
	return "question_types"
}

type KnowledgePoint struct {
	ID        string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	DomainID  string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	ParentID  *string          `json:"parent_id,omitempty" gorm:"type:varchar(36);index"`
	Code      string           `json:"code" gorm:"type:varchar(128);not null"`
	Name      string           `json:"name" gorm:"type:varchar(255);not null"`
	Path      string           `json:"path" gorm:"type:text;not null;default:''"`
	Level     int              `json:"level" gorm:"not null;default:1"`
	SortOrder int              `json:"sort_order" gorm:"not null;default:0"`
	Status    ExamDomainStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func (KnowledgePoint) TableName() string {
	return "knowledge_points"
}

type Question struct {
	ID               string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64           `json:"tenant_id" gorm:"not null;index"`
	QuestionBankID   string           `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID         string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID        *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	GroupID          *string          `json:"group_id,omitempty" gorm:"type:varchar(36);index"`
	QuestionTypeID   *string          `json:"question_type_id,omitempty" gorm:"type:varchar(36);index"`
	QuestionNo       string           `json:"question_no" gorm:"type:varchar(64);not null;default:''"`
	OrderInGroup     int              `json:"order_in_group" gorm:"not null;default:0"`
	Stem             string           `json:"stem" gorm:"type:text;not null"`
	QuestionMetadata JSONMap          `json:"question_metadata" gorm:"type:jsonb;not null;default:'{}'"`
	Difficulty       string           `json:"difficulty" gorm:"type:varchar(32);not null;default:'unknown'"`
	SourceYear       *int             `json:"source_year,omitempty"`
	SourceRegion     string           `json:"source_region" gorm:"type:varchar(128);not null;default:''"`
	ReviewStatus     ExamReviewStatus `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status           string           `json:"status" gorm:"type:varchar(32);not null;default:'draft'"`
	CreatedByUserID  string           `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func (Question) TableName() string {
	return "questions"
}

type QuestionGroup struct {
	ID              string           `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64           `json:"tenant_id" gorm:"not null;index"`
	SpaceID         string           `json:"space_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID  string           `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	DomainID        string           `json:"domain_id" gorm:"type:varchar(36);not null;index"`
	SubjectID       *string          `json:"subject_id,omitempty" gorm:"type:varchar(36);index"`
	GroupType       string           `json:"group_type" gorm:"type:varchar(64);not null;default:'single_question'"`
	Title           string           `json:"title" gorm:"type:varchar(255);not null;default:''"`
	MaterialText    string           `json:"material_text" gorm:"type:text;not null;default:''"`
	MaterialFormat  string           `json:"material_format" gorm:"type:varchar(32);not null;default:'plain_text'"`
	AssetRefs       JSON             `json:"asset_refs" gorm:"type:jsonb;not null"`
	SourceChunkIDs  JSON             `json:"source_chunk_ids" gorm:"type:jsonb;not null"`
	SourceYear      *int             `json:"source_year,omitempty"`
	SourceRegion    string           `json:"source_region" gorm:"type:varchar(128);not null;default:''"`
	PaperType       string           `json:"paper_type" gorm:"type:varchar(128);not null;default:''"`
	SortOrder       int              `json:"sort_order" gorm:"not null;default:0"`
	ReviewStatus    ExamReviewStatus `json:"review_status" gorm:"type:varchar(32);not null;default:'private'"`
	Status          string           `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string           `json:"created_by_user_id" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (QuestionGroup) TableName() string {
	return "question_groups"
}

type QuestionGroupAsset struct {
	ID            string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID      uint64    `json:"tenant_id" gorm:"not null;index"`
	GroupID       string    `json:"group_id" gorm:"type:varchar(36);not null;index"`
	AssetType     string    `json:"asset_type" gorm:"type:varchar(64);not null"`
	StorageURI    string    `json:"storage_uri" gorm:"type:text;not null;default:''"`
	AltText       string    `json:"alt_text" gorm:"type:text;not null;default:''"`
	SourceChunkID string    `json:"source_chunk_id" gorm:"type:varchar(36);not null;default:''"`
	BBox          JSONMap   `json:"bbox" gorm:"type:jsonb;not null"`
	Metadata      JSONMap   `json:"metadata" gorm:"type:jsonb;not null"`
	SortOrder     int       `json:"sort_order" gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

func (QuestionGroupAsset) TableName() string {
	return "question_group_assets"
}

type QuestionOption struct {
	ID         string `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID string `json:"question_id" gorm:"type:varchar(36);not null;index"`
	OptionKey  string `json:"option_key" gorm:"type:varchar(16);not null"`
	Content    string `json:"content" gorm:"type:text;not null"`
	SortOrder  int    `json:"sort_order" gorm:"not null;default:0"`
}

func (QuestionOption) TableName() string {
	return "question_options"
}

type QuestionAnswer struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID string    `json:"question_id" gorm:"type:varchar(36);not null;index"`
	AnswerText string    `json:"answer_text" gorm:"type:text;not null"`
	IsCorrect  bool      `json:"is_correct" gorm:"not null;default:true"`
	CreatedAt  time.Time `json:"created_at"`
}

func (QuestionAnswer) TableName() string {
	return "question_answers"
}

type QuestionExplanation struct {
	ID              string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	QuestionID      string    `json:"question_id" gorm:"type:varchar(36);not null;index"`
	ExplanationText string    `json:"explanation_text" gorm:"type:text;not null"`
	SourceType      string    `json:"source_type" gorm:"type:varchar(64);not null;default:'manual'"`
	CreatedAt       time.Time `json:"created_at"`
}

func (QuestionExplanation) TableName() string {
	return "question_explanations"
}

type QuestionChunkRef struct {
	QuestionID string    `json:"question_id" gorm:"type:varchar(36);primaryKey"`
	ChunkID    string    `json:"chunk_id" gorm:"type:varchar(36);primaryKey"`
	RefType    string    `json:"ref_type" gorm:"type:varchar(64);primaryKey"`
	Confidence float64   `json:"confidence" gorm:"type:numeric(5,4);not null;default:1"`
	CreatedAt  time.Time `json:"created_at"`
}

func (QuestionChunkRef) TableName() string {
	return "question_chunk_refs"
}

type QuestionDetail struct {
	Question     *Question              `json:"question"`
	Options      []*QuestionOption      `json:"options"`
	Answers      []*QuestionAnswer      `json:"answers"`
	Explanations []*QuestionExplanation `json:"explanations"`
	ChunkRefs    []*QuestionChunkRef    `json:"chunk_refs"`
}

type QuestionGroupDetail struct {
	Group     *QuestionGroup        `json:"group"`
	Assets    []*QuestionGroupAsset `json:"assets"`
	Questions []*QuestionDetail     `json:"questions"`
}

type CreateQuestionBankRequest struct {
	SpaceID     string  `json:"space_id" binding:"required"`
	DomainID    string  `json:"domain_id" binding:"required"`
	SubjectID   *string `json:"subject_id"`
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	Description string  `json:"description" binding:"omitempty,max=1000"`
}

package types

import "time"

type ExamPracticeAttemptStatus string

const (
	ExamPracticeAttemptStatusInProgress ExamPracticeAttemptStatus = "in_progress"
	ExamPracticeAttemptStatusCompleted  ExamPracticeAttemptStatus = "completed"
)

type ExamPracticeAttempt struct {
	ID             string                    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID       uint64                    `json:"tenant_id" gorm:"not null;index"`
	UserID         string                    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	SpaceID        string                    `json:"space_id" gorm:"type:varchar(36);not null;index"`
	QuestionBankID string                    `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	GroupID        string                    `json:"group_id" gorm:"type:varchar(36);not null;index"`
	Status         ExamPracticeAttemptStatus `json:"status" gorm:"type:varchar(32);not null;default:'in_progress'"`
	QuestionCount  int                       `json:"question_count" gorm:"not null;default:0"`
	AnsweredCount  int                       `json:"answered_count" gorm:"not null;default:0"`
	CorrectCount   int                       `json:"correct_count" gorm:"not null;default:0"`
	StartedAt      time.Time                 `json:"started_at"`
	CompletedAt    *time.Time                `json:"completed_at,omitempty"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

func (ExamPracticeAttempt) TableName() string {
	return "exam_practice_attempts"
}

type ExamPracticeAnswer struct {
	ID                  string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID            uint64    `json:"tenant_id" gorm:"not null;index"`
	AttemptID           string    `json:"attempt_id" gorm:"type:varchar(36);not null;index;uniqueIndex:uniq_exam_practice_answer_question"`
	QuestionID          string    `json:"question_id" gorm:"type:varchar(36);not null;index;uniqueIndex:uniq_exam_practice_answer_question"`
	QuestionNo          string    `json:"question_no" gorm:"type:varchar(64);not null;default:''"`
	AnswerText          string    `json:"answer_text" gorm:"type:text;not null;default:''"`
	IsCorrect           bool      `json:"is_correct" gorm:"not null;default:false"`
	CorrectAnswer       string    `json:"correct_answer" gorm:"type:text;not null;default:''"`
	QuestionSnapshot    JSONMap   `json:"question_snapshot" gorm:"type:jsonb;not null"`
	AnswerSnapshot      JSON      `json:"answer_snapshot" gorm:"type:jsonb;not null"`
	ExplanationSnapshot JSON      `json:"explanation_snapshot" gorm:"type:jsonb;not null"`
	AnsweredAt          time.Time `json:"answered_at"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func (ExamPracticeAnswer) TableName() string {
	return "exam_practice_answers"
}

type ListPracticeQuestionGroupsFilter struct {
	SpaceID   string
	DomainID  string
	SubjectID string
	Limit     int
}

type ListPracticeAttemptsFilter struct {
	SpaceID string
	GroupID string
	Limit   int
}

type ListWrongQuestionsFilter struct {
	SpaceID string
	GroupID string
	Limit   int
}

type QuestionGroupPracticeSummary struct {
	Group         *QuestionGroup        `json:"group"`
	BankName      string                `json:"bank_name"`
	QuestionCount int                   `json:"question_count"`
	LastAttempt   *ExamPracticeAttempt  `json:"last_attempt,omitempty"`
	Assets        []*QuestionGroupAsset `json:"assets,omitempty"`
}

type PracticeAttemptSummary struct {
	Attempt  *ExamPracticeAttempt `json:"attempt"`
	Group    *QuestionGroup       `json:"group,omitempty"`
	BankName string               `json:"bank_name"`
}

type PracticeAttemptDetail struct {
	Attempt *ExamPracticeAttempt  `json:"attempt"`
	Group   *QuestionGroupDetail  `json:"group"`
	Answers []*ExamPracticeAnswer `json:"answers"`
}

type WrongQuestionItem struct {
	Attempt  *ExamPracticeAttempt `json:"attempt"`
	Answer   *ExamPracticeAnswer  `json:"answer"`
	Group    *QuestionGroup       `json:"group,omitempty"`
	BankName string               `json:"bank_name"`
}

type CreatePracticeAttemptResult struct {
	Attempt *ExamPracticeAttempt `json:"attempt"`
	Group   *QuestionGroupDetail `json:"group"`
}

type SubmitPracticeAnswerRequest struct {
	QuestionID string `json:"question_id" binding:"required"`
	AnswerText string `json:"answer_text" binding:"required"`
}

type PracticeAnswerResult struct {
	Attempt        *ExamPracticeAttempt   `json:"attempt"`
	Answer         *ExamPracticeAnswer    `json:"answer"`
	CorrectAnswers []string               `json:"correct_answers"`
	Explanations   []*QuestionExplanation `json:"explanations"`
	ChunkRefs      []*QuestionChunkRef    `json:"chunk_refs"`
}

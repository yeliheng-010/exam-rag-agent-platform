package types

import "time"

type ExamRAGEvaluationRunStatus string

const (
	ExamRAGEvaluationRunStatusQueued    ExamRAGEvaluationRunStatus = "queued"
	ExamRAGEvaluationRunStatusRunning   ExamRAGEvaluationRunStatus = "running"
	ExamRAGEvaluationRunStatusCompleted ExamRAGEvaluationRunStatus = "completed"
	ExamRAGEvaluationRunStatusFailed    ExamRAGEvaluationRunStatus = "failed"
)

type ExamRAGEvaluationRun struct {
	ID              string                     `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64                     `json:"tenant_id" gorm:"not null;index"`
	QuestionBankID  string                     `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	CreatedBy       string                     `json:"created_by" gorm:"type:varchar(36);not null;index"`
	Status          ExamRAGEvaluationRunStatus `json:"status" gorm:"type:varchar(32);not null;default:'queued'"`
	Progress        JSON                       `json:"progress" gorm:"type:jsonb;not null;default:'{}'"`
	RequestSnapshot JSON                       `json:"request_snapshot" gorm:"type:jsonb;not null"`
	ResultSnapshot  JSON                       `json:"result_snapshot" gorm:"type:jsonb"`
	ErrorMessage    string                     `json:"error_message" gorm:"type:text;not null;default:''"`
	StartedAt       *time.Time                 `json:"started_at,omitempty"`
	CompletedAt     *time.Time                 `json:"completed_at,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

func (ExamRAGEvaluationRun) TableName() string {
	return "exam_rag_evaluation_runs"
}

type ExamRAGEvaluationProgress struct {
	CompletedCases int `json:"completed_cases"`
	TotalCases     int `json:"total_cases"`
}

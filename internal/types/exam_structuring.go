package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type ExamStructuringPhase string

const (
	ExamStructuringPhaseQueued       ExamStructuringPhase = "queued"
	ExamStructuringPhasePreflight    ExamStructuringPhase = "preflight"
	ExamStructuringPhaseExtracting   ExamStructuringPhase = "extracting"
	ExamStructuringPhaseQualityCheck ExamStructuringPhase = "quality_check"
	ExamStructuringPhaseCompleted    ExamStructuringPhase = "completed"
	ExamStructuringPhaseFailed       ExamStructuringPhase = "failed"
)

type ExamQualitySeverity string

const (
	ExamQualitySeverityError   ExamQualitySeverity = "error"
	ExamQualitySeverityWarning ExamQualitySeverity = "warning"
)

type ExamStructuringWarning struct {
	Code           string              `json:"code"`
	Severity       ExamQualitySeverity `json:"severity"`
	Message        string              `json:"message"`
	Recommendation string              `json:"recommendation,omitempty"`
	ReferenceCount int                 `json:"reference_count,omitempty"`
}

type ExamQuestionGroupQualityIssue struct {
	Code       string              `json:"code"`
	Severity   ExamQualitySeverity `json:"severity"`
	Message    string              `json:"message"`
	QuestionNo string              `json:"question_no,omitempty"`
}

type ExamQuestionGroupQualityReport struct {
	Blocking     bool                            `json:"blocking"`
	ErrorCount   int                             `json:"error_count"`
	WarningCount int                             `json:"warning_count"`
	Issues       []ExamQuestionGroupQualityIssue `json:"issues"`
}

type ExamStructuringQualitySummary struct {
	TotalDrafts       int `json:"total_drafts"`
	DraftsWithErrors  int `json:"drafts_with_errors"`
	DraftsWithWarning int `json:"drafts_with_warnings"`
	ErrorCount        int `json:"error_count"`
	WarningCount      int `json:"warning_count"`
}

type ExamStructuringProgress struct {
	Phase            ExamStructuringPhase          `json:"phase,omitempty"`
	TotalBatches     int                           `json:"total_batches"`
	CompletedBatches int                           `json:"completed_batches"`
	CurrentBatch     int                           `json:"current_batch"`
	FailedBatch      int                           `json:"failed_batch,omitempty"`
	Percent          int                           `json:"percent"`
	Message          string                        `json:"message,omitempty"`
	Warnings         []ExamStructuringWarning      `json:"warnings"`
	QualitySummary   ExamStructuringQualitySummary `json:"quality_summary"`
	StartedAt        *time.Time                    `json:"started_at,omitempty"`
	FinishedAt       *time.Time                    `json:"finished_at,omitempty"`
}

func (p *ExamStructuringProgress) Scan(value interface{}) error {
	if value == nil {
		*p = ExamStructuringProgress{}
		return nil
	}
	var data []byte
	switch typed := value.(type) {
	case []byte:
		data = typed
	case string:
		data = []byte(typed)
	default:
		return errors.New("exam structuring progress must be JSON bytes or string")
	}
	if len(data) == 0 {
		*p = ExamStructuringProgress{}
		return nil
	}
	return json.Unmarshal(data, p)
}

func (p ExamStructuringProgress) Value() (driver.Value, error) {
	return json.Marshal(p)
}

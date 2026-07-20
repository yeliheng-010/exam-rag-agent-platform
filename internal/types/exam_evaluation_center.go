package types

import "time"

type ExamEvaluationComparisonStatus string

const (
	ExamEvaluationComparisonBaseline   ExamEvaluationComparisonStatus = "baseline"
	ExamEvaluationComparisonRegressed  ExamEvaluationComparisonStatus = "regressed"
	ExamEvaluationComparisonStable     ExamEvaluationComparisonStatus = "stable"
	ExamEvaluationComparisonUncompared ExamEvaluationComparisonStatus = "uncompared"
)

type ExamEvaluationRegressionFilter string

const (
	ExamEvaluationRegressionAll        ExamEvaluationRegressionFilter = "all"
	ExamEvaluationRegressionRegressed  ExamEvaluationRegressionFilter = "regressed"
	ExamEvaluationRegressionStable     ExamEvaluationRegressionFilter = "stable"
	ExamEvaluationRegressionUncompared ExamEvaluationRegressionFilter = "uncompared"
)

type ExamEvaluationCenterFilter struct {
	Kind       ExamEvaluationKind
	BankID     string
	AgentID    string
	Status     ExamRAGEvaluationRunStatus
	Regression ExamEvaluationRegressionFilter
	Limit      int
}

type ExamEvaluationRegressionReason struct {
	Metric   string  `json:"metric"`
	Label    string  `json:"label"`
	Baseline float64 `json:"baseline"`
	Current  float64 `json:"current"`
	Delta    float64 `json:"delta"`
}

type ExamEvaluationCenterItem struct {
	Run               *ExamRAGEvaluationRun            `json:"run"`
	QuestionBankName  string                           `json:"question_bank_name"`
	AgentName         string                           `json:"agent_name,omitempty"`
	EvaluationSetName string                           `json:"evaluation_set_name,omitempty"`
	Metrics           map[string]float64               `json:"metrics"`
	BaselineRunID     string                           `json:"baseline_run_id,omitempty"`
	BaselineMetrics   map[string]float64               `json:"baseline_metrics,omitempty"`
	MetricDeltas      map[string]float64               `json:"metric_deltas,omitempty"`
	ComparisonStatus  ExamEvaluationComparisonStatus   `json:"comparison_status"`
	RegressionReasons []ExamEvaluationRegressionReason `json:"regression_reasons"`
}

type ExamEvaluationCenterSummary struct {
	TotalRuns       int     `json:"total_runs"`
	CompletedRuns   int     `json:"completed_runs"`
	BaselineRuns    int     `json:"baseline_runs"`
	RegressionRuns  int     `json:"regression_runs"`
	AveragePassRate float64 `json:"average_pass_rate"`
}

type ExamEvaluationAgentOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExamEvaluationCenterResult struct {
	Summary       ExamEvaluationCenterSummary `json:"summary"`
	Items         []*ExamEvaluationCenterItem `json:"items"`
	QuestionBanks []*QuestionBank             `json:"question_banks"`
	Agents        []ExamEvaluationAgentOption `json:"agents"`
}

type ExamEvaluationSetStatus string

const ExamEvaluationSetStatusActive ExamEvaluationSetStatus = "active"

type ExamEvaluationSet struct {
	ID               string                      `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID         uint64                      `json:"tenant_id" gorm:"not null;index"`
	QuestionBankID   string                      `json:"question_bank_id" gorm:"type:varchar(36);not null;index"`
	EvaluationKind   ExamEvaluationKind          `json:"evaluation_kind" gorm:"type:varchar(16);not null;index"`
	AgentID          string                      `json:"agent_id,omitempty" gorm:"type:varchar(36);index"`
	Name             string                      `json:"name" gorm:"type:varchar(160);not null"`
	Description      string                      `json:"description" gorm:"type:text;not null;default:''"`
	CurrentVersion   int                         `json:"current_version" gorm:"not null;default:1"`
	CreatedBy        string                      `json:"created_by" gorm:"type:varchar(36);not null;index"`
	Status           ExamEvaluationSetStatus     `json:"status" gorm:"type:varchar(32);not null;default:'active';index"`
	CreatedAt        time.Time                   `json:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at"`
	QuestionBankName string                      `json:"question_bank_name,omitempty" gorm:"-"`
	AgentName        string                      `json:"agent_name,omitempty" gorm:"-"`
	Versions         []*ExamEvaluationSetVersion `json:"versions,omitempty" gorm:"-"`
}

func (ExamEvaluationSet) TableName() string { return "exam_evaluation_sets" }

type ExamEvaluationSetVersion struct {
	ID                 string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64    `json:"tenant_id" gorm:"not null;index"`
	EvaluationSetID    string    `json:"evaluation_set_id" gorm:"type:varchar(36);not null;index"`
	Version            int       `json:"version" gorm:"not null"`
	SourceRunID        string    `json:"source_run_id" gorm:"type:varchar(36);not null;index"`
	DefinitionSnapshot JSON      `json:"definition_snapshot" gorm:"type:jsonb;not null"`
	CreatedBy          string    `json:"created_by" gorm:"type:varchar(36);not null"`
	CreatedAt          time.Time `json:"created_at"`
}

func (ExamEvaluationSetVersion) TableName() string { return "exam_evaluation_set_versions" }

type CreateExamEvaluationSetRequest struct {
	RunID       string `json:"run_id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type CreateExamEvaluationSetVersionRequest struct {
	RunID string `json:"run_id" binding:"required"`
}

type RunExamEvaluationSetRequest struct {
	Version int `json:"version"`
}

type SetExamEvaluationBaselineRequest struct {
	RunID string `json:"run_id" binding:"required"`
}

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const defaultExamClassDiagnosisLimit = 10
const maxExamClassDiagnosisLimit = 20

var examClassDiagnosisTool = BaseTool{
	name: ToolExamClassDiagnosis,
	description: `Return teacher-facing class learning analytics for exam diagnosis.

Use this tool when a teacher or assistant asks about class progress, mastery trends, frequent wrong questions, students needing attention, or teaching priorities.
If class_id is unknown, call without it to list analyzable classes. The tool enforces active teacher/assistant membership. Combine with exam_question_context when full question or passage content is needed.`,
	schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "class_id": {
      "type": "string",
      "description": "Optional class ID. Omit to list classes or auto-select the only analyzable class."
    },
    "top_wrong_questions": {
      "type": "integer",
      "description": "Maximum frequent wrong questions to include.",
      "minimum": 1,
      "maximum": 20,
      "default": 10
    },
    "at_risk_students": {
      "type": "integer",
      "description": "Maximum students needing attention to include.",
      "minimum": 1,
      "maximum": 20,
      "default": 10
    }
  }
}`),
}

type ExamClassDiagnosisInput struct {
	ClassID           string `json:"class_id,omitempty"`
	TopWrongQuestions int    `json:"top_wrong_questions,omitempty"`
	AtRiskStudents    int    `json:"at_risk_students,omitempty"`
}

type ExamClassDiagnosisTool struct {
	BaseTool
	analyticsService interfaces.ExamAnalyticsService
}

func NewExamClassDiagnosisTool(analyticsService interfaces.ExamAnalyticsService) *ExamClassDiagnosisTool {
	return &ExamClassDiagnosisTool{BaseTool: examClassDiagnosisTool, analyticsService: analyticsService}
}

func (t *ExamClassDiagnosisTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	input, err := parseExamClassDiagnosisInput(args)
	if err != nil {
		return failedExamClassDiagnosisResult(err.Error()), err
	}
	scope, err := t.resolveClassDiagnosisScope(ctx)
	if err != nil {
		return failedExamClassDiagnosisResult(err.Error()), err
	}
	classID, selection, err := t.resolveClassID(ctx, scope, input.ClassID)
	if err != nil {
		return failedExamClassDiagnosisResult(err.Error()), err
	}
	if selection != nil {
		return formatExamClassSelectionResult(selection), nil
	}
	summary, err := t.analyticsService.GetClassAnalytics(ctx, scope.tenantID, scope.userID, classID)
	if err != nil {
		maskedErr := errors.New("class is unavailable or permission denied")
		return failedExamClassDiagnosisResult(maskedErr.Error()), maskedErr
	}
	input.ClassID = classID
	input.TopWrongQuestions = normalizeExamClassDiagnosisLimit(input.TopWrongQuestions)
	input.AtRiskStudents = normalizeExamClassDiagnosisLimit(input.AtRiskStudents)
	return formatExamClassDiagnosisResult(summary, input), nil
}

type examClassDiagnosisScope struct {
	tenantID uint64
	userID   string
}

func (t *ExamClassDiagnosisTool) resolveClassDiagnosisScope(ctx context.Context) (examClassDiagnosisScope, error) {
	if t.analyticsService == nil {
		return examClassDiagnosisScope{}, fmt.Errorf("exam analytics service is unavailable")
	}
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return examClassDiagnosisScope{}, fmt.Errorf("tenant scope is required")
	}
	userID := strings.TrimSpace(types.SessionOwnerIDFromContext(ctx))
	if userID == "" {
		return examClassDiagnosisScope{}, fmt.Errorf("user scope is required")
	}
	return examClassDiagnosisScope{tenantID: tenantID, userID: userID}, nil
}

func (t *ExamClassDiagnosisTool) resolveClassID(
	ctx context.Context,
	scope examClassDiagnosisScope,
	requested string,
) (string, []*types.ExamClass, error) {
	if classID := strings.TrimSpace(requested); classID != "" {
		return classID, nil, nil
	}
	classes, err := t.analyticsService.ListAnalyzableClasses(ctx, scope.tenantID, scope.userID)
	if err != nil {
		return "", nil, err
	}
	if len(classes) == 1 && classes[0] != nil {
		return classes[0].ID, nil, nil
	}
	return "", classes, nil
}

func parseExamClassDiagnosisInput(args json.RawMessage) (ExamClassDiagnosisInput, error) {
	var input ExamClassDiagnosisInput
	if len(args) == 0 {
		return input, nil
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return input, fmt.Errorf("failed to parse args: %w", err)
	}
	return input, nil
}

func normalizeExamClassDiagnosisLimit(limit int) int {
	if limit <= 0 {
		return defaultExamClassDiagnosisLimit
	}
	if limit > maxExamClassDiagnosisLimit {
		return maxExamClassDiagnosisLimit
	}
	return limit
}

func failedExamClassDiagnosisResult(message string) *types.ToolResult {
	return &types.ToolResult{Success: false, Error: message}
}

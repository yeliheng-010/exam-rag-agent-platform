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

var examPracticeRecommendationTool = BaseTool{
	name: ToolExamPracticeRecommendation,
	description: `Recommend existing structured exam question groups from diagnosis evidence.

Use scope=student for the current student's unmastered wrong questions. Use scope=class with class_id after exam_class_diagnosis for teacher-facing intervention. Recommendations are read-only and never publish assignments. Do not invent group IDs; only use IDs returned by this tool.`,
	schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "scope": {
      "type": "string",
      "enum": ["student", "class"],
      "default": "student"
    },
    "class_id": {
      "type": "string",
      "description": "Required when scope is class."
    },
    "space_id": {
      "type": "string",
      "description": "Optional student practice space scope."
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 10,
      "default": 5
    },
    "include_assigned": {
      "type": "boolean",
      "description": "Class scope only: include groups already published to the class.",
      "default": false
    }
  }
}`),
}

type ExamPracticeRecommendationInput struct {
	Scope           types.ExamPracticeRecommendationScope `json:"scope,omitempty"`
	ClassID         string                                `json:"class_id,omitempty"`
	SpaceID         string                                `json:"space_id,omitempty"`
	Limit           int                                   `json:"limit,omitempty"`
	IncludeAssigned bool                                  `json:"include_assigned,omitempty"`
}

type ExamPracticeRecommendationTool struct {
	BaseTool
	interventionService interfaces.ExamInterventionService
}

func NewExamPracticeRecommendationTool(
	interventionService interfaces.ExamInterventionService,
) *ExamPracticeRecommendationTool {
	return &ExamPracticeRecommendationTool{
		BaseTool:            examPracticeRecommendationTool,
		interventionService: interventionService,
	}
}

func (t *ExamPracticeRecommendationTool) Execute(
	ctx context.Context,
	args json.RawMessage,
) (*types.ToolResult, error) {
	input, err := parseExamPracticeRecommendationInput(args)
	if err != nil {
		return failedExamPracticeRecommendationResult(err.Error()), err
	}
	scope, err := t.resolvePracticeRecommendationScope(ctx)
	if err != nil {
		return failedExamPracticeRecommendationResult(err.Error()), err
	}
	request := types.ExamPracticeRecommendationRequest{
		SpaceID: strings.TrimSpace(input.SpaceID), Limit: input.Limit, IncludeAssigned: input.IncludeAssigned,
	}
	result, err := t.recommend(ctx, scope, input, request)
	if err != nil {
		if input.Scope == types.ExamPracticeRecommendationScopeClass {
			err = errors.New("practice recommendations are unavailable or permission denied")
		}
		return failedExamPracticeRecommendationResult(err.Error()), err
	}
	return formatExamPracticeRecommendationResult(result), nil
}

type practiceRecommendationScope struct {
	tenantID uint64
	userID   string
}

func (t *ExamPracticeRecommendationTool) resolvePracticeRecommendationScope(
	ctx context.Context,
) (practiceRecommendationScope, error) {
	if t.interventionService == nil {
		return practiceRecommendationScope{}, fmt.Errorf("exam intervention service is unavailable")
	}
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return practiceRecommendationScope{}, fmt.Errorf("tenant scope is required")
	}
	userID := strings.TrimSpace(types.SessionOwnerIDFromContext(ctx))
	if userID == "" {
		return practiceRecommendationScope{}, fmt.Errorf("user scope is required")
	}
	return practiceRecommendationScope{tenantID: tenantID, userID: userID}, nil
}

func (t *ExamPracticeRecommendationTool) recommend(
	ctx context.Context,
	scope practiceRecommendationScope,
	input ExamPracticeRecommendationInput,
	request types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	if input.Scope == types.ExamPracticeRecommendationScopeClass {
		classID := strings.TrimSpace(input.ClassID)
		if classID == "" {
			return nil, fmt.Errorf("class_id is required for class scope")
		}
		return t.interventionService.RecommendClassPractice(ctx, scope.tenantID, scope.userID, classID, request)
	}
	return t.interventionService.RecommendStudentPractice(ctx, scope.tenantID, scope.userID, request)
}

func parseExamPracticeRecommendationInput(args json.RawMessage) (ExamPracticeRecommendationInput, error) {
	input := ExamPracticeRecommendationInput{Scope: types.ExamPracticeRecommendationScopeStudent}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return input, fmt.Errorf("failed to parse args: %w", err)
		}
	}
	if input.Scope == "" {
		input.Scope = types.ExamPracticeRecommendationScopeStudent
	}
	if input.Scope != types.ExamPracticeRecommendationScopeStudent && input.Scope != types.ExamPracticeRecommendationScopeClass {
		return input, fmt.Errorf("scope must be student or class")
	}
	if input.Scope == types.ExamPracticeRecommendationScopeClass && strings.TrimSpace(input.ClassID) == "" {
		return input, fmt.Errorf("class_id is required for class scope")
	}
	return input, nil
}

func failedExamPracticeRecommendationResult(message string) *types.ToolResult {
	return &types.ToolResult{Success: false, Error: message}
}

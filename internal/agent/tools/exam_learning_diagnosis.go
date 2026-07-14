package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const defaultExamLearningDiagnosisLimit = 10
const maxExamLearningDiagnosisLimit = 30

var examLearningDiagnosisTool = BaseTool{
	name: ToolExamLearningDiagnosis,
	description: `Return the current student's wrong-question review context for exam learning diagnosis.

Use this tool when the user asks about:
- their wrong questions, weak points, mastery status, or review plan
- which questions still need review
- a diagnosis based on completed practice attempts

The output includes recent wrong questions, student answers, correct answers, explanations, review status, and concise summary counts. If the user asks for the original passage or full grouped question context, combine this tool with exam_question_context.`,
	schema: json.RawMessage(`{
  "type": "object",
  "properties": {
    "space_id": {
      "type": "string",
      "description": "Optional class/space scope."
    },
    "group_id": {
      "type": "string",
      "description": "Optional question group ID to focus the diagnosis."
    },
    "limit": {
      "type": "integer",
      "description": "Maximum wrong questions to include.",
      "minimum": 1,
      "maximum": 30,
      "default": 10
    },
    "include_mastered": {
      "type": "boolean",
      "description": "Whether to include questions already marked as mastered.",
      "default": false
    }
  }
}`),
}

type ExamLearningDiagnosisInput struct {
	SpaceID         string `json:"space_id,omitempty"`
	GroupID         string `json:"group_id,omitempty"`
	Limit           int    `json:"limit,omitempty"`
	IncludeMastered bool   `json:"include_mastered,omitempty"`
}

type ExamLearningDiagnosisTool struct {
	BaseTool
	practiceService interfaces.ExamPracticeService
}

func NewExamLearningDiagnosisTool(practiceService interfaces.ExamPracticeService) *ExamLearningDiagnosisTool {
	return &ExamLearningDiagnosisTool{
		BaseTool:        examLearningDiagnosisTool,
		practiceService: practiceService,
	}
}

func (t *ExamLearningDiagnosisTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input ExamLearningDiagnosisInput
	if len(args) > 0 {
		if err := json.Unmarshal(args, &input); err != nil {
			return failedExamLearningDiagnosisResult(fmt.Sprintf("Failed to parse args: %v", err)), err
		}
	}
	scope, err := t.resolveScope(ctx)
	if err != nil {
		return failedExamLearningDiagnosisResult(err.Error()), err
	}
	limit := normalizeExamDiagnosisLimit(input.Limit)
	items, err := t.practiceService.ListWrongQuestions(ctx, scope.tenantID, scope.userID, types.ListWrongQuestionsFilter{
		SpaceID: strings.TrimSpace(input.SpaceID),
		GroupID: strings.TrimSpace(input.GroupID),
		Limit:   limit,
	})
	if err != nil {
		return failedExamLearningDiagnosisResult(err.Error()), err
	}
	summary := buildExamLearningDiagnosisSummary(items, input.IncludeMastered, limit)
	return formatExamLearningDiagnosisResult(summary, input), nil
}

type examLearningDiagnosisScope struct {
	tenantID uint64
	userID   string
}

func (t *ExamLearningDiagnosisTool) resolveScope(ctx context.Context) (examLearningDiagnosisScope, error) {
	if t.practiceService == nil {
		return examLearningDiagnosisScope{}, fmt.Errorf("exam practice service is unavailable")
	}
	tenantID, ok := types.SessionTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return examLearningDiagnosisScope{}, fmt.Errorf("tenant scope is required")
	}
	userID := strings.TrimSpace(types.SessionOwnerIDFromContext(ctx))
	if userID == "" {
		return examLearningDiagnosisScope{}, fmt.Errorf("user scope is required")
	}
	return examLearningDiagnosisScope{tenantID: tenantID, userID: userID}, nil
}

type examLearningDiagnosisSummary struct {
	Items           []examLearningDiagnosisItem
	Returned        int
	Considered      int
	OmittedMastered int
	StatusCounts    map[types.PracticeAnswerReviewStatus]int
}

type examLearningDiagnosisItem struct {
	AnswerID      string
	AttemptID     string
	GroupID       string
	GroupTitle    string
	GroupType     string
	BankName      string
	QuestionNo    string
	Stem          string
	Options       []string
	StudentAnswer string
	CorrectAnswer string
	Explanation   string
	ReviewStatus  types.PracticeAnswerReviewStatus
	ReviewNote    string
	AnsweredAt    time.Time
}

func buildExamLearningDiagnosisSummary(
	wrongItems []*types.WrongQuestionItem,
	includeMastered bool,
	limit int,
) examLearningDiagnosisSummary {
	summary := examLearningDiagnosisSummary{Returned: len(wrongItems), StatusCounts: newDiagnosisStatusCounts()}
	for _, wrong := range wrongItems {
		item, ok := normalizeWrongQuestionItem(wrong)
		if !ok {
			continue
		}
		if item.ReviewStatus == types.PracticeAnswerReviewStatusMastered && !includeMastered {
			summary.OmittedMastered++
			continue
		}
		if len(summary.Items) >= limit {
			continue
		}
		summary.Items = append(summary.Items, item)
		summary.Considered++
		summary.StatusCounts[item.ReviewStatus]++
	}
	return summary
}

func newDiagnosisStatusCounts() map[types.PracticeAnswerReviewStatus]int {
	return map[types.PracticeAnswerReviewStatus]int{
		types.PracticeAnswerReviewStatusUnreviewed: 0,
		types.PracticeAnswerReviewStatusReviewing:  0,
		types.PracticeAnswerReviewStatusMastered:   0,
	}
}

func normalizeWrongQuestionItem(wrong *types.WrongQuestionItem) (examLearningDiagnosisItem, bool) {
	if wrong == nil || wrong.Answer == nil {
		return examLearningDiagnosisItem{}, false
	}
	answer := wrong.Answer
	item := examLearningDiagnosisItem{
		AnswerID:      answer.ID,
		QuestionNo:    firstNonEmpty(answer.QuestionNo, snapshotString(answer.QuestionSnapshot, "question_no")),
		Stem:          snapshotString(answer.QuestionSnapshot, "stem"),
		Options:       snapshotOptions(answer.QuestionSnapshot),
		StudentAnswer: strings.TrimSpace(answer.AnswerText),
		CorrectAnswer: strings.TrimSpace(answer.CorrectAnswer),
		Explanation:   firstExplanation(answer.ExplanationSnapshot),
		ReviewStatus:  normalizeReviewStatus(answer.ReviewStatus),
		ReviewNote:    strings.TrimSpace(answer.ReviewNote),
		AnsweredAt:    answer.AnsweredAt,
	}
	attachWrongQuestionScope(&item, wrong)
	return item, true
}

func attachWrongQuestionScope(item *examLearningDiagnosisItem, wrong *types.WrongQuestionItem) {
	if wrong.Attempt != nil {
		item.AttemptID = wrong.Attempt.ID
		item.GroupID = wrong.Attempt.GroupID
	}
	if wrong.Group != nil {
		item.GroupID = firstNonEmpty(wrong.Group.ID, item.GroupID)
		item.GroupTitle = strings.TrimSpace(wrong.Group.Title)
		item.GroupType = strings.TrimSpace(wrong.Group.GroupType)
	}
	item.BankName = strings.TrimSpace(wrong.BankName)
}

func normalizeExamDiagnosisLimit(limit int) int {
	if limit <= 0 {
		return defaultExamLearningDiagnosisLimit
	}
	if limit > maxExamLearningDiagnosisLimit {
		return maxExamLearningDiagnosisLimit
	}
	return limit
}

func normalizeReviewStatus(status types.PracticeAnswerReviewStatus) types.PracticeAnswerReviewStatus {
	switch status {
	case types.PracticeAnswerReviewStatusReviewing, types.PracticeAnswerReviewStatusMastered:
		return status
	default:
		return types.PracticeAnswerReviewStatusUnreviewed
	}
}

func failedExamLearningDiagnosisResult(message string) *types.ToolResult {
	return &types.ToolResult{Success: false, Error: message}
}

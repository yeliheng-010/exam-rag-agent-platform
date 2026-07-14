package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubExamLearningPracticeService struct {
	interfaces.ExamPracticeService
	items     []*types.WrongQuestionItem
	gotTenant uint64
	gotUser   string
	gotFilter types.ListWrongQuestionsFilter
}

func (s *stubExamLearningPracticeService) ListWrongQuestions(
	_ context.Context,
	tenantID uint64,
	userID string,
	filter types.ListWrongQuestionsFilter,
) ([]*types.WrongQuestionItem, error) {
	s.gotTenant = tenantID
	s.gotUser = userID
	s.gotFilter = filter
	return s.items, nil
}

func TestExamLearningDiagnosisToolSummarizesWrongQuestions(t *testing.T) {
	now := time.Date(2026, 7, 9, 10, 30, 0, 0, time.UTC)
	practice := &stubExamLearningPracticeService{
		items: []*types.WrongQuestionItem{
			newWrongQuestionItem("answer-21", "21", "B", "A", types.PracticeAnswerReviewStatusUnreviewed, "", now),
			newWrongQuestionItem("answer-22", "22", "D", "C", types.PracticeAnswerReviewStatusReviewing, "Need reread hotel table.", now.Add(-time.Minute)),
			newWrongQuestionItem("answer-23", "23", "A", "C", types.PracticeAnswerReviewStatusMastered, "", now.Add(-2*time.Minute)),
		},
	}
	tool := NewExamLearningDiagnosisTool(practice)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")
	args := json.RawMessage(`{"space_id":"space-1","group_id":"group-reading-a","limit":5}`)

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	if practice.gotTenant != 10000 || practice.gotUser != "student-1" {
		t.Fatalf("wrong scope tenant=%d user=%q", practice.gotTenant, practice.gotUser)
	}
	if practice.gotFilter.SpaceID != "space-1" || practice.gotFilter.GroupID != "group-reading-a" {
		t.Fatalf("filter = %#v", practice.gotFilter)
	}
	assertDiagnosisContains(t, result.Output, `<exam_learning_diagnosis`)
	assertDiagnosisContains(t, result.Output, `total_considered="2"`)
	assertDiagnosisContains(t, result.Output, `omitted_mastered="1"`)
	assertDiagnosisContains(t, result.Output, `reviewing="1"`)
	assertDiagnosisContains(t, result.Output, `question_no="21"`)
	assertDiagnosisContains(t, result.Output, `student_answer="A"`)
	assertDiagnosisContains(t, result.Output, `correct_answer="B"`)
	assertDiagnosisContains(t, result.Output, `A. Dallas Cowboys`)
	assertDiagnosisContains(t, result.Output, `B. Los Angeles Rams`)
	assertDiagnosisContains(t, result.Output, `Evidence comes from the stadium schedule.`)
	assertDiagnosisContains(t, result.Output, `Need reread hotel table.`)
	if strings.Contains(result.Output, "question_no=\"23\"") {
		t.Fatalf("mastered question should be omitted by default:\n%s", result.Output)
	}
	if result.Data["display_type"] != ToolExamLearningDiagnosis {
		t.Fatalf("display_type = %#v", result.Data["display_type"])
	}
}

func TestExamLearningDiagnosisToolIncludesMasteredWhenRequested(t *testing.T) {
	practice := &stubExamLearningPracticeService{
		items: []*types.WrongQuestionItem{
			newWrongQuestionItem("answer-23", "23", "C", "A", types.PracticeAnswerReviewStatusMastered, "", time.Now()),
		},
	}
	tool := NewExamLearningDiagnosisTool(practice)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{"include_mastered":true}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	assertDiagnosisContains(t, result.Output, `total_considered="1"`)
	assertDiagnosisContains(t, result.Output, `mastered="1"`)
	assertDiagnosisContains(t, result.Output, `question_no="23"`)
}

func TestExamLearningDiagnosisToolReturnsEmptyContext(t *testing.T) {
	tool := NewExamLearningDiagnosisTool(&stubExamLearningPracticeService{})
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful empty result, got %#v", result)
	}
	assertDiagnosisContains(t, result.Output, `empty="true"`)
	assertDiagnosisContains(t, result.Output, `no wrong questions`)
}

func TestExamLearningDiagnosisToolIsRegisteredByDefault(t *testing.T) {
	var foundDefinition bool
	for _, tool := range AvailableToolDefinitions() {
		if tool.Name == ToolExamLearningDiagnosis {
			foundDefinition = true
			break
		}
	}
	if !foundDefinition {
		t.Fatalf("%s missing from AvailableToolDefinitions()", ToolExamLearningDiagnosis)
	}
	if !strings.Contains(strings.Join(DefaultAllowedTools(), ","), ToolExamLearningDiagnosis) {
		t.Fatalf("%s missing from DefaultAllowedTools()", ToolExamLearningDiagnosis)
	}
	req, ok := ToolCapabilityRequirements[ToolExamLearningDiagnosis]
	if !ok {
		t.Fatalf("%s missing from ToolCapabilityRequirements", ToolExamLearningDiagnosis)
	}
	if len(req.AnyOf) != 0 || len(req.AllOf) != 0 || req.ConsumesFiles {
		t.Fatalf("learning diagnosis should not require KB or files: %#v", req)
	}
}

func newWrongQuestionItem(
	answerID string,
	questionNo string,
	correctAnswer string,
	studentAnswer string,
	status types.PracticeAnswerReviewStatus,
	note string,
	answeredAt time.Time,
) *types.WrongQuestionItem {
	return &types.WrongQuestionItem{
		Attempt: &types.ExamPracticeAttempt{
			ID:             "attempt-" + questionNo,
			TenantID:       10000,
			UserID:         "student-1",
			SpaceID:        "space-1",
			QuestionBankID: "bank-english",
			GroupID:        "group-reading-a",
			QuestionCount:  3,
			CorrectCount:   1,
			AnsweredCount:  3,
		},
		Answer: &types.ExamPracticeAnswer{
			ID:                  answerID,
			TenantID:            10000,
			AttemptID:           "attempt-" + questionNo,
			QuestionID:          "question-" + questionNo,
			QuestionNo:          questionNo,
			AnswerText:          studentAnswer,
			IsCorrect:           false,
			CorrectAnswer:       correctAnswer,
			QuestionSnapshot:    newLearningQuestionSnapshot(questionNo),
			ExplanationSnapshot: types.JSON(`[{"explanation_text":"Evidence comes from the stadium schedule."}]`),
			ReviewStatus:        status,
			ReviewNote:          note,
			AnsweredAt:          answeredAt,
		},
		Group: &types.QuestionGroup{
			ID:           "group-reading-a",
			GroupType:    "reading_passage",
			Title:        "SoFi Stadium Events",
			MaterialText: "SoFi Stadium is the go-to destination.",
			SourceRegion: "National",
		},
		BankName: "2026 Gaokao English",
	}
}

func newLearningQuestionSnapshot(questionNo string) types.JSONMap {
	return types.JSONMap{
		"id":          "question-" + questionNo,
		"question_no": questionNo,
		"stem":        "Which team will play the most games?",
		"options": []map[string]any{
			{"option_key": "A", "content": "Dallas Cowboys"},
			{"option_key": "B", "content": "Los Angeles Rams"},
		},
	}
}

func assertDiagnosisContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("missing %q in:\n%s", want, got)
	}
}

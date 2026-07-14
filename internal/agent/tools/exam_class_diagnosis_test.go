package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type stubExamClassAnalyticsService struct {
	classes   []*types.ExamClass
	summary   *types.ExamClassAnalyticsSummary
	err       error
	gotTenant uint64
	gotUser   string
	gotClass  string
}

func (s *stubExamClassAnalyticsService) ListAnalyzableClasses(
	_ context.Context,
	tenantID uint64,
	userID string,
) ([]*types.ExamClass, error) {
	s.gotTenant = tenantID
	s.gotUser = userID
	return s.classes, s.err
}

func (s *stubExamClassAnalyticsService) GetClassAnalytics(
	_ context.Context,
	tenantID uint64,
	userID string,
	classID string,
) (*types.ExamClassAnalyticsSummary, error) {
	s.gotTenant = tenantID
	s.gotUser = userID
	s.gotClass = classID
	return s.summary, s.err
}

func TestExamClassDiagnosisToolDiagnosesOnlyAnalyzableClass(t *testing.T) {
	now := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)
	class := &types.ExamClass{ID: "class-1", Name: "高三一班"}
	analytics := &stubExamClassAnalyticsService{
		classes: []*types.ExamClass{class},
		summary: newExamClassDiagnosisSummary(class, now),
	}
	tool := NewExamClassDiagnosisTool(analytics)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "teacher-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{"top_wrong_questions":1,"at_risk_students":1}`))

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	if analytics.gotTenant != 10000 || analytics.gotUser != "teacher-1" || analytics.gotClass != class.ID {
		t.Fatalf("analytics scope = tenant %d user %q class %q", analytics.gotTenant, analytics.gotUser, analytics.gotClass)
	}
	assertClassDiagnosisContains(t, result.Output, `<exam_class_diagnosis`)
	assertClassDiagnosisContains(t, result.Output, `class_name="高三一班"`)
	assertClassDiagnosisContains(t, result.Output, `<class_summary total_students="2"`)
	assertClassDiagnosisContains(t, result.Output, `direction="improving"`)
	assertClassDiagnosisContains(t, result.Output, `question_no="21"`)
	assertClassDiagnosisContains(t, result.Output, `wrong_rate="100.0%"`)
	assertClassDiagnosisContains(t, result.Output, `student_name="测试学生"`)
	assertClassDiagnosisContains(t, result.Output, `reason="not_started,low_completion"`)
	if result.Data["display_type"] != ToolExamClassDiagnosis {
		t.Fatalf("display_type = %#v", result.Data["display_type"])
	}
}

func TestExamClassDiagnosisToolReturnsClassSelectionWhenAmbiguous(t *testing.T) {
	analytics := &stubExamClassAnalyticsService{classes: []*types.ExamClass{
		{ID: "class-1", Name: "高三一班"},
		{ID: "class-2", Name: "高三二班"},
	}}
	tool := NewExamClassDiagnosisTool(analytics)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "teacher-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{}`))

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected successful result, got %#v", result)
	}
	assertClassDiagnosisContains(t, result.Output, `<exam_class_selection count="2">`)
	assertClassDiagnosisContains(t, result.Output, `class_id="class-1"`)
	assertClassDiagnosisContains(t, result.Output, `class_id="class-2"`)
	if analytics.gotClass != "" {
		t.Fatalf("ambiguous selection must not diagnose a class, got %q", analytics.gotClass)
	}
}

func TestExamClassDiagnosisToolMasksUnavailableClassReason(t *testing.T) {
	for _, serviceErr := range []error{errors.New("exam not found"), errors.New("exam permission denied")} {
		analytics := &stubExamClassAnalyticsService{err: serviceErr}
		tool := NewExamClassDiagnosisTool(analytics)
		ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")

		result, err := tool.Execute(ctx, json.RawMessage(`{"class_id":"class-1"}`))

		if err == nil || result == nil || result.Success {
			t.Fatalf("expected unavailable class failure, result=%#v err=%v", result, err)
		}
		const masked = "class is unavailable or permission denied"
		if err.Error() != masked || result.Error != masked {
			t.Fatalf("unmasked class error: result=%q err=%q", result.Error, err.Error())
		}
	}
}

func TestExamClassDiagnosisToolIsRegisteredByDefault(t *testing.T) {
	var found bool
	for _, definition := range AvailableToolDefinitions() {
		if definition.Name == ToolExamClassDiagnosis {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("%s missing from AvailableToolDefinitions", ToolExamClassDiagnosis)
	}
	if !strings.Contains(strings.Join(DefaultAllowedTools(), ","), ToolExamClassDiagnosis) {
		t.Fatalf("%s missing from DefaultAllowedTools", ToolExamClassDiagnosis)
	}
	requirement, ok := ToolCapabilityRequirements[ToolExamClassDiagnosis]
	if !ok || requirement.ConsumesFiles || len(requirement.AnyOf) != 0 || len(requirement.AllOf) != 0 {
		t.Fatalf("class diagnosis capability requirement = %#v, present=%t", requirement, ok)
	}
}

func newExamClassDiagnosisSummary(class *types.ExamClass, now time.Time) *types.ExamClassAnalyticsSummary {
	oldAssignment := &types.ExamClassAssignment{ID: "assignment-old", Title: "第一次练习", CreatedAt: now.Add(-2 * time.Hour)}
	newAssignment := &types.ExamClassAssignment{ID: "assignment-new", Title: "第二次练习", CreatedAt: now.Add(-time.Hour)}
	return &types.ExamClassAnalyticsSummary{
		Class:                class,
		TotalStudents:        2,
		AssignmentCount:      2,
		TotalAssignmentSlots: 4,
		StartedCount:         2,
		CompletedCount:       2,
		CompletionRate:       0.5,
		AverageCorrectRate:   0.6,
		Members: []*types.ExamClassAnalyticsMember{
			{
				Member:          &types.ExamClassMember{UserID: "student-1", DisplayName: "测试学生"},
				AssignmentCount: 2,
			},
			{
				Member:             &types.ExamClassMember{UserID: "student-2", DisplayName: "进步学生"},
				AssignmentCount:    2,
				StartedCount:       2,
				CompletedCount:     2,
				CompletionRate:     1,
				AverageCorrectRate: 0.9,
			},
		},
		Assignments: []*types.ExamClassAnalyticsAssignment{
			{Assignment: newAssignment, StartedCount: 1, CompletedCount: 1, CompletionRate: 0.5, AverageCorrectRate: 0.8},
			{Assignment: oldAssignment, StartedCount: 1, CompletedCount: 1, CompletionRate: 0.5, AverageCorrectRate: 0.4},
		},
		FrequentWrongQuestions: []*types.ExamClassFrequentWrongQuestion{
			{QuestionID: "question-21", QuestionNo: "21", Stem: "Which team will play the most games?", AnswerCount: 2, WrongCount: 2, WrongRate: 1, AffectedStudentCount: 2},
			{QuestionID: "question-22", QuestionNo: "22", Stem: "Which hotel is nearest?", AnswerCount: 2, WrongCount: 1, WrongRate: 0.5, AffectedStudentCount: 1},
		},
	}
}

func assertClassDiagnosisContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("missing %q in:\n%s", want, got)
	}
}

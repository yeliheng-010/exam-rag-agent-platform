package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubExamInterventionService struct {
	interfaces.ExamInterventionService
	result   *types.ExamPracticeRecommendationResult
	gotScope types.ExamPracticeRecommendationScope
	gotClass string
	gotReq   types.ExamPracticeRecommendationRequest
}

func (s *stubExamInterventionService) RecommendClassPractice(
	_ context.Context,
	_ uint64,
	_ string,
	classID string,
	req types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	s.gotScope = types.ExamPracticeRecommendationScopeClass
	s.gotClass = classID
	s.gotReq = req
	return s.result, nil
}

func (s *stubExamInterventionService) RecommendStudentPractice(
	_ context.Context,
	_ uint64,
	_ string,
	req types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	s.gotScope = types.ExamPracticeRecommendationScopeStudent
	s.gotReq = req
	return s.result, nil
}

func TestExamPracticeRecommendationToolFormatsClassRecommendations(t *testing.T) {
	service := &stubExamInterventionService{result: newToolPracticeRecommendationResult(types.ExamPracticeRecommendationScopeClass)}
	tool := NewExamPracticeRecommendationTool(service)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "teacher-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{"scope":"class","class_id":"class-1","limit":3}`))

	if err != nil || result == nil || !result.Success {
		t.Fatalf("Execute result=%#v err=%v", result, err)
	}
	if service.gotScope != types.ExamPracticeRecommendationScopeClass || service.gotClass != "class-1" || service.gotReq.Limit != 3 {
		t.Fatalf("service call scope=%q class=%q req=%#v", service.gotScope, service.gotClass, service.gotReq)
	}
	assertPracticeRecommendationContains(t, result.Output, `<exam_practice_recommendations scope="class"`)
	assertPracticeRecommendationContains(t, result.Output, `group_id="group-reading"`)
	assertPracticeRecommendationContains(t, result.Output, `code="same_subject"`)
	assertPracticeRecommendationContains(t, result.Output, `question_no="21"`)
	assertPracticeRecommendationContains(t, result.Output, `Teacher confirmation is required`)
	if result.Data["display_type"] != ToolExamPracticeRecommendation {
		t.Fatalf("display_type = %#v", result.Data["display_type"])
	}
}

func TestExamPracticeRecommendationToolDefaultsToStudentScope(t *testing.T) {
	service := &stubExamInterventionService{result: newToolPracticeRecommendationResult(types.ExamPracticeRecommendationScopeStudent)}
	tool := NewExamPracticeRecommendationTool(service)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{"space_id":"space-1"}`))

	if err != nil || result == nil || !result.Success {
		t.Fatalf("Execute result=%#v err=%v", result, err)
	}
	if service.gotScope != types.ExamPracticeRecommendationScopeStudent || service.gotReq.SpaceID != "space-1" {
		t.Fatalf("service call scope=%q req=%#v", service.gotScope, service.gotReq)
	}
	assertPracticeRecommendationContains(t, result.Output, `scope="student"`)
}

func TestExamPracticeRecommendationToolRequiresClassID(t *testing.T) {
	tool := NewExamPracticeRecommendationTool(&stubExamInterventionService{})
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10000))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "teacher-1")

	result, err := tool.Execute(ctx, json.RawMessage(`{"scope":"class"}`))

	if err == nil || result == nil || result.Success || !strings.Contains(result.Error, "class_id") {
		t.Fatalf("expected class_id failure, result=%#v err=%v", result, err)
	}
}

func TestExamPracticeRecommendationToolIsRegisteredByDefault(t *testing.T) {
	var found bool
	for _, definition := range AvailableToolDefinitions() {
		if definition.Name == ToolExamPracticeRecommendation {
			found = true
			break
		}
	}
	if !found || !strings.Contains(strings.Join(DefaultAllowedTools(), ","), ToolExamPracticeRecommendation) {
		t.Fatalf("%s is not registered by default", ToolExamPracticeRecommendation)
	}
	requirement, ok := ToolCapabilityRequirements[ToolExamPracticeRecommendation]
	if !ok || requirement.ConsumesFiles || len(requirement.AnyOf) > 0 || len(requirement.AllOf) > 0 {
		t.Fatalf("capability requirement=%#v present=%t", requirement, ok)
	}
}

func newToolPracticeRecommendationResult(scope types.ExamPracticeRecommendationScope) *types.ExamPracticeRecommendationResult {
	subjectID := "subject-english"
	group := &types.QuestionGroup{ID: "group-reading", SubjectID: &subjectID, GroupType: "reading_passage", Title: "阅读迁移"}
	return &types.ExamPracticeRecommendationResult{
		Scope: scope,
		Class: &types.ExamClass{ID: "class-1", Name: "高三一班"},
		Diagnosis: []types.ExamPracticeDiagnosisEvidence{
			{QuestionID: "question-21", GroupID: "weak-group", QuestionNo: "21", Stem: "Which team plays most?", WrongRate: 0.8},
		},
		Recommendations: []*types.ExamPracticeRecommendation{
			{
				Group: &types.QuestionGroupPracticeSummary{Group: group, BankName: "高考英语", QuestionCount: 3},
				Score: 80,
				Reasons: []types.ExamPracticeRecommendationReason{
					{Code: types.ExamPracticeRecommendationReasonSameSubject, Score: 40},
					{Code: types.ExamPracticeRecommendationReasonSameGroupType, Score: 25},
				},
				Evidence: []types.ExamPracticeDiagnosisEvidence{{QuestionID: "question-21", QuestionNo: "21"}},
			},
		},
		Warnings: []string{},
	}
}

func assertPracticeRecommendationContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("missing %q in:\n%s", want, got)
	}
}

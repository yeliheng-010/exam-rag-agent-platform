package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubInterventionAnalyticsService struct {
	interfaces.ExamAnalyticsService
	summary *types.ExamClassAnalyticsSummary
	err     error
}

func TestExamInterventionConstructorUsesRegisteredAssignmentRepositoryContract(t *testing.T) {
	var constructor func(
		interfaces.ExamAnalyticsService,
		interfaces.ExamPracticeService,
		interfaces.ExamAssignmentRepository,
	) interfaces.ExamInterventionService = NewExamInterventionService
	if constructor == nil {
		t.Fatal("NewExamInterventionService constructor is nil")
	}
}

func (s *stubInterventionAnalyticsService) GetClassAnalytics(
	context.Context,
	uint64,
	string,
	string,
) (*types.ExamClassAnalyticsSummary, error) {
	return s.summary, s.err
}

type stubInterventionPracticeService struct {
	interfaces.ExamPracticeService
	groups       []*types.QuestionGroupPracticeSummary
	wrong        []*types.WrongQuestionItem
	details      map[string]*types.QuestionGroupDetail
	gotGroupList types.ListPracticeQuestionGroupsFilter
}

func (s *stubInterventionPracticeService) ListQuestionGroups(
	_ context.Context,
	_ uint64,
	_ string,
	filter types.ListPracticeQuestionGroupsFilter,
) ([]*types.QuestionGroupPracticeSummary, error) {
	s.gotGroupList = filter
	return s.groups, nil
}

func (s *stubInterventionPracticeService) ListWrongQuestions(
	context.Context,
	uint64,
	string,
	types.ListWrongQuestionsFilter,
) ([]*types.WrongQuestionItem, error) {
	return s.wrong, nil
}

func (s *stubInterventionPracticeService) GetQuestionGroupDetail(
	_ context.Context,
	_ uint64,
	_ string,
	groupID string,
) (*types.QuestionGroupDetail, error) {
	if detail := s.details[groupID]; detail != nil {
		return detail, nil
	}
	return nil, errors.New("group not found")
}

type stubInterventionAssignmentRepo struct {
	interfaces.ExamAssignmentRepository
	assignments     []*types.ExamClassAssignment
	publishedGroups []*types.QuestionGroup
}

func (s *stubInterventionAssignmentRepo) ListPublishedGroupsByClass(
	_ context.Context,
	_ uint64,
	_ string,
) ([]*types.QuestionGroup, error) {
	out := append([]*types.QuestionGroup(nil), s.publishedGroups...)
	seen := make(map[string]bool, len(out))
	for _, group := range out {
		if group != nil {
			seen[group.ID] = true
		}
	}
	for _, assignment := range s.assignments {
		if assignment != nil && assignment.GroupID != "" && !seen[assignment.GroupID] {
			out = append(out, &types.QuestionGroup{ID: assignment.GroupID})
			seen[assignment.GroupID] = true
		}
	}
	return out, nil
}

func (s *stubInterventionAssignmentRepo) ListAssignmentsByClass(
	_ context.Context,
	_ uint64,
	_ string,
	limit int,
) ([]*types.ExamClassAssignment, error) {
	if limit > 0 && len(s.assignments) > limit {
		return s.assignments[:limit], nil
	}
	return s.assignments, nil
}

func TestExamInterventionRecommendsUnassignedClassGroupsByStructuredSignals(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	weak := interventionGroup("weak-group", "personal-space", "domain-gaokao", "subject-english", "reading_passage", "阅读 A")
	assigned := interventionGroup("assigned-group", "personal-space", "domain-gaokao", "subject-english", "reading_passage", "已发布阅读")
	reading := interventionGroup("reading-transfer", "personal-space", "domain-gaokao", "subject-english", "reading_passage", "迁移阅读")
	math := interventionGroup("math-group", "personal-space", "domain-gaokao", "subject-math", "math_problem", "数学练习")
	analytics := &stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{
		Class: class,
		FrequentWrongQuestions: []*types.ExamClassFrequentWrongQuestion{
			{QuestionID: "q-21", GroupID: weak.ID, QuestionNo: "21", Stem: "Which team plays most?", WrongRate: 0.8, AffectedStudentCount: 8},
		},
	}}
	practice := &stubInterventionPracticeService{
		groups: []*types.QuestionGroupPracticeSummary{
			interventionSummary(weak), interventionSummary(assigned), interventionSummary(reading), interventionSummary(math),
		},
		details: map[string]*types.QuestionGroupDetail{weak.ID: {Group: weak}},
	}
	assignments := &stubInterventionAssignmentRepo{assignments: []*types.ExamClassAssignment{
		{ID: "assignment-weak", GroupID: weak.ID},
		{ID: "assignment-existing", GroupID: assigned.ID},
	}}
	svc := NewExamInterventionService(analytics, practice, assignments)

	result, err := svc.RecommendClassPractice(context.Background(), 10000, "teacher-1", class.ID, types.ExamPracticeRecommendationRequest{Limit: 5})

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	if result.Scope != types.ExamPracticeRecommendationScopeClass || result.Class == nil || result.Class.ID != class.ID {
		t.Fatalf("result scope/class = %#v", result)
	}
	if len(result.Recommendations) == 0 || result.Recommendations[0].Group.Group.ID != reading.ID {
		t.Fatalf("recommendations = %#v, want reading transfer first", result.Recommendations)
	}
	assertInterventionReason(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonSameSubject)
	assertInterventionReason(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonSameGroupType)
	for _, recommendation := range result.Recommendations {
		if recommendation.Group.Group.ID == weak.ID || recommendation.Group.Group.ID == assigned.ID {
			t.Fatalf("assigned group leaked into default recommendations: %#v", recommendation)
		}
	}
}

func TestExamInterventionStudentRecommendationIgnoresMasteredWrongQuestions(t *testing.T) {
	weak := interventionGroup("weak-group", "student-space", "domain-gaokao", "subject-english", "reading_passage", "阅读 A")
	mastered := interventionGroup("mastered-group", "student-space", "domain-gaokao", "subject-math", "math_problem", "已掌握数学")
	transfer := interventionGroup("reading-transfer", "student-space", "domain-gaokao", "subject-english", "reading_passage", "阅读迁移")
	practice := &stubInterventionPracticeService{
		groups: []*types.QuestionGroupPracticeSummary{interventionSummary(mastered), interventionSummary(transfer), interventionSummary(weak)},
		wrong: []*types.WrongQuestionItem{
			interventionWrongQuestion(weak, "answer-1", types.PracticeAnswerReviewStatusUnreviewed),
			interventionWrongQuestion(mastered, "answer-2", types.PracticeAnswerReviewStatusMastered),
		},
		details: map[string]*types.QuestionGroupDetail{
			weak.ID:     {Group: weak},
			mastered.ID: {Group: mastered},
		},
	}
	svc := NewExamInterventionService(&stubInterventionAnalyticsService{}, practice, &stubInterventionAssignmentRepo{})

	result, err := svc.RecommendStudentPractice(context.Background(), 10000, "student-1", types.ExamPracticeRecommendationRequest{Limit: 5})

	if err != nil {
		t.Fatalf("RecommendStudentPractice returned error: %v", err)
	}
	if result.Scope != types.ExamPracticeRecommendationScopeStudent || len(result.Diagnosis) != 1 {
		t.Fatalf("student diagnosis = %#v", result)
	}
	if len(result.Recommendations) == 0 || result.Recommendations[0].Group.Group.ID != weak.ID {
		t.Fatalf("recommendations = %#v, want targeted weak group first", result.Recommendations)
	}
	assertInterventionReason(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonTargetedReview)
	if result.Diagnosis[0].GroupID == mastered.ID {
		t.Fatalf("mastered wrong question must not be used as diagnosis evidence")
	}
}

func TestExamInterventionFallsBackWithoutActiveWrongQuestions(t *testing.T) {
	candidate := interventionGroup("candidate", "student-space", "domain-ielts", "subject-reading", "reading_passage", "IELTS Reading")
	practice := &stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{interventionSummary(candidate)}}
	svc := NewExamInterventionService(&stubInterventionAnalyticsService{}, practice, &stubInterventionAssignmentRepo{})

	result, err := svc.RecommendStudentPractice(context.Background(), 10000, "student-1", types.ExamPracticeRecommendationRequest{Limit: 3})

	if err != nil {
		t.Fatalf("RecommendStudentPractice returned error: %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("recommendations = %#v, want one fallback", result.Recommendations)
	}
	assertInterventionReason(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonSupplemental)
	if len(result.Warnings) == 0 || result.Warnings[0] != "no_active_wrong_questions" {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestExamInterventionClassRecommendationIgnoresTeacherPracticeAttempt(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	candidate := interventionGroup("candidate", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "教师做过的题组")
	summary := interventionSummary(candidate)
	summary.LastAttempt = &types.ExamPracticeAttempt{QuestionCount: 10, CorrectCount: 2}
	practice := &stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{summary}}
	analytics := &stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{Class: class}}
	svc := NewExamInterventionService(analytics, practice, &stubInterventionAssignmentRepo{})

	result, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"teacher-1",
		class.ID,
		types.ExamPracticeRecommendationRequest{Limit: 3},
	)

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("recommendations = %#v, want one fallback", result.Recommendations)
	}
	assertInterventionReason(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonSupplemental)
	assertInterventionReasonAbsent(t, result.Recommendations[0], types.ExamPracticeRecommendationReasonLowAccuracyRetry)
}

func TestExamInterventionExcludesPublishedGroupBeyondRecentAssignmentLimit(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	oldPublished := interventionGroup("old-published", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "历史已发布题组")
	candidate := interventionGroup("candidate", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "新候选题组")
	practice := &stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{
		interventionSummary(oldPublished),
		interventionSummary(candidate),
	}}
	assignments := make([]*types.ExamClassAssignment, 0, 101)
	for i := 0; i < 100; i++ {
		assignments = append(assignments, &types.ExamClassAssignment{ID: fmt.Sprintf("assignment-%03d", i), GroupID: fmt.Sprintf("other-group-%03d", i)})
	}
	assignments = append(assignments, &types.ExamClassAssignment{ID: "assignment-old", GroupID: oldPublished.ID})
	svc := NewExamInterventionService(
		&stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{Class: class}},
		practice,
		&stubInterventionAssignmentRepo{assignments: assignments},
	)

	result, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"teacher-1",
		class.ID,
		types.ExamPracticeRecommendationRequest{Limit: 5},
	)

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	for _, recommendation := range result.Recommendations {
		if recommendation.Group.Group.ID == oldPublished.ID {
			t.Fatalf("old published group leaked into recommendations: %#v", recommendation)
		}
	}
}

func TestExamInterventionIncludesAssignedGroupWhenRequested(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	assigned := interventionGroup("assigned-group", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "原题复习")
	svc := NewExamInterventionService(
		&stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{Class: class}},
		&stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{interventionSummary(assigned)}},
		&stubInterventionAssignmentRepo{assignments: []*types.ExamClassAssignment{{ID: "assignment-1", GroupID: assigned.ID}}},
	)

	result, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"teacher-1",
		class.ID,
		types.ExamPracticeRecommendationRequest{Limit: 5, IncludeAssigned: true},
	)

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	if len(result.Recommendations) != 1 || !result.Recommendations[0].Assigned {
		t.Fatalf("recommendations = %#v, want assigned group included and marked", result.Recommendations)
	}
}

func TestExamInterventionPreservesClassPermissionDenied(t *testing.T) {
	svc := NewExamInterventionService(
		&stubInterventionAnalyticsService{err: ErrExamPermissionDenied},
		&stubInterventionPracticeService{},
		&stubInterventionAssignmentRepo{},
	)

	_, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"student-1",
		"class-1",
		types.ExamPracticeRecommendationRequest{},
	)

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("RecommendClassPractice error = %v, want permission denied", err)
	}
}

func TestExamInterventionExcludesSourceGroupAfterCrossSpaceImport(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	source := interventionGroup("source-group", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "跨空间阅读")
	imported := *source
	imported.ID = "imported-group"
	imported.SpaceID = class.SpaceID
	imported.QuestionBankID = "class-bank"
	practice := &stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{interventionSummary(source)}}
	assignments := &stubInterventionAssignmentRepo{
		assignments:     []*types.ExamClassAssignment{{ID: "assignment-1", GroupID: imported.ID}},
		publishedGroups: []*types.QuestionGroup{&imported},
	}
	svc := NewExamInterventionService(
		&stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{Class: class}},
		practice,
		assignments,
	)

	result, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"teacher-1",
		class.ID,
		types.ExamPracticeRecommendationRequest{Limit: 5},
	)

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	if len(result.Recommendations) != 0 {
		t.Fatalf("recommendations = %#v, want imported source group excluded", result.Recommendations)
	}
}

func TestExamInterventionDeduplicatesCrossSpaceCloneWhenAssignedIncluded(t *testing.T) {
	class := &types.ExamClass{ID: "class-1", TenantID: 10000, Name: "高三一班", SpaceID: "class-space"}
	source := interventionGroup("source-group", "teacher-space", "domain-gaokao", "subject-english", "reading_passage", "跨空间阅读")
	imported := *source
	imported.ID = "imported-group"
	imported.SpaceID = class.SpaceID
	imported.QuestionBankID = "class-bank"
	practice := &stubInterventionPracticeService{groups: []*types.QuestionGroupPracticeSummary{
		interventionSummary(source),
		interventionSummary(&imported),
	}}
	svc := NewExamInterventionService(
		&stubInterventionAnalyticsService{summary: &types.ExamClassAnalyticsSummary{Class: class}},
		practice,
		&stubInterventionAssignmentRepo{
			assignments:     []*types.ExamClassAssignment{{ID: "assignment-1", GroupID: imported.ID}},
			publishedGroups: []*types.QuestionGroup{&imported},
		},
	)

	result, err := svc.RecommendClassPractice(
		context.Background(),
		10000,
		"teacher-1",
		class.ID,
		types.ExamPracticeRecommendationRequest{Limit: 5, IncludeAssigned: true},
	)

	if err != nil {
		t.Fatalf("RecommendClassPractice returned error: %v", err)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("recommendations = %#v, want one deduplicated group", result.Recommendations)
	}
	if result.Recommendations[0].Group.Group.ID != imported.ID || !result.Recommendations[0].Assigned {
		t.Fatalf("recommendation = %#v, want assigned class-space clone", result.Recommendations[0])
	}
}

func interventionGroup(id, spaceID, domainID, subjectID, groupType, title string) *types.QuestionGroup {
	return &types.QuestionGroup{
		ID: id, TenantID: 10000, SpaceID: spaceID, QuestionBankID: "bank-" + id,
		DomainID: domainID, SubjectID: &subjectID, GroupType: groupType, Title: title,
		Status: "active", ReviewStatus: types.ExamReviewStatusApproved, UpdatedAt: time.Now(),
	}
}

func interventionSummary(group *types.QuestionGroup) *types.QuestionGroupPracticeSummary {
	return &types.QuestionGroupPracticeSummary{Group: group, BankName: "题库 " + group.ID, QuestionCount: 3}
}

func interventionWrongQuestion(
	group *types.QuestionGroup,
	answerID string,
	status types.PracticeAnswerReviewStatus,
) *types.WrongQuestionItem {
	return &types.WrongQuestionItem{
		Group: group,
		Answer: &types.ExamPracticeAnswer{
			ID: answerID, QuestionID: "question-" + answerID, QuestionNo: "21",
			QuestionSnapshot: types.JSONMap{"stem": "Which option is correct?"}, ReviewStatus: status,
		},
	}
}

func assertInterventionReason(
	t *testing.T,
	recommendation *types.ExamPracticeRecommendation,
	want types.ExamPracticeRecommendationReasonCode,
) {
	t.Helper()
	for _, reason := range recommendation.Reasons {
		if reason.Code == want {
			return
		}
	}
	t.Fatalf("recommendation reasons = %#v, want %q", recommendation.Reasons, want)
}

func assertInterventionReasonAbsent(
	t *testing.T,
	recommendation *types.ExamPracticeRecommendation,
	unwanted types.ExamPracticeRecommendationReasonCode,
) {
	t.Helper()
	for _, reason := range recommendation.Reasons {
		if reason.Code == unwanted {
			t.Fatalf("recommendation reasons = %#v, must not include %q", recommendation.Reasons, unwanted)
		}
	}
}

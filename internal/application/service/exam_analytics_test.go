package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestExamClassAnalyticsSummarizesAssignmentsAndStudents(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	practiceRepo := newStubPracticeRepo()
	class := seedExamClass(classRepo, "class-analytics", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "assistant-1", types.ExamClassRoleAssistant, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-high", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-low", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-pending", types.ExamClassRoleStudent, types.ExamClassMemberStatusPending)
	now := time.Now()
	firstAssignment := seedAnalyticsAssignment(assignRepo, class, "assignment-1", "group-1", now.Add(-2*time.Hour))
	secondAssignment := seedAnalyticsAssignment(assignRepo, class, "assignment-2", "group-2", now.Add(-1*time.Hour))
	seedAnalyticsAttempt(practiceRepo, firstAssignment, "student-high", 5, 4, true, now.Add(-80*time.Minute))
	seedAnalyticsAttempt(practiceRepo, secondAssignment, "student-high", 5, 5, true, now.Add(-20*time.Minute))
	seedAnalyticsAttempt(practiceRepo, firstAssignment, "student-low", 5, 2, false, now.Add(-60*time.Minute))
	seedAnalyticsAttempt(practiceRepo, firstAssignment, "student-pending", 5, 5, true, now.Add(-50*time.Minute))
	svc := NewExamAnalyticsService(classRepo, assignRepo, practiceRepo)

	analytics, err := svc.GetClassAnalytics(ctx, 10000, "teacher-1", class.ID)

	if err != nil {
		t.Fatalf("GetClassAnalytics returned error: %v", err)
	}
	if analytics.TotalStudents != 2 || analytics.AssignmentCount != 2 || analytics.TotalAssignmentSlots != 4 {
		t.Fatalf("summary counts = students %d assignments %d slots %d, want 2/2/4",
			analytics.TotalStudents, analytics.AssignmentCount, analytics.TotalAssignmentSlots)
	}
	if analytics.StartedCount != 3 || analytics.CompletedCount != 2 {
		t.Fatalf("progress counts = started %d completed %d, want 3/2", analytics.StartedCount, analytics.CompletedCount)
	}
	if analytics.CompletionRate != 0.5 {
		t.Fatalf("completion rate = %.2f, want 0.50", analytics.CompletionRate)
	}
	if analytics.AverageCorrectRate != 0.7333333333333334 {
		t.Fatalf("average correct rate = %.4f, want 0.7333", analytics.AverageCorrectRate)
	}
	if len(analytics.Members) != 2 {
		t.Fatalf("member analytics count = %d, want 2", len(analytics.Members))
	}
	high := findAnalyticsMember(analytics.Members, "student-high")
	if high == nil || high.StartedCount != 2 || high.CompletedCount != 2 || high.AverageCorrectRate != 0.9 {
		t.Fatalf("student-high analytics = %#v, want 2 started, 2 completed, 0.90 average", high)
	}
	low := findAnalyticsMember(analytics.Members, "student-low")
	if low == nil || low.StartedCount != 1 || low.CompletedCount != 0 || low.AverageCorrectRate != 0.4 {
		t.Fatalf("student-low analytics = %#v, want 1 started, 0 completed, 0.40 average", low)
	}
	if findAnalyticsMember(analytics.Members, "student-pending") != nil {
		t.Fatalf("pending student should not appear in analytics members")
	}
	if len(analytics.Assignments) != 2 {
		t.Fatalf("assignment analytics count = %d, want 2", len(analytics.Assignments))
	}
	second := findAnalyticsAssignment(analytics.Assignments, secondAssignment.ID)
	if second == nil || second.StartedCount != 1 || second.CompletedCount != 1 || second.AverageCorrectRate != 1 {
		t.Fatalf("assignment-2 analytics = %#v, want 1 started, 1 completed, 1.00 average", second)
	}
}

func TestExamClassAnalyticsRequiresClassWriteRole(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	practiceRepo := newStubPracticeRepo()
	class := seedExamClass(classRepo, "class-analytics", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	svc := NewExamAnalyticsService(classRepo, assignRepo, practiceRepo)

	_, err := svc.GetClassAnalytics(ctx, 10000, "student-1", class.ID)

	if !errors.Is(err, ErrExamPermissionDenied) {
		t.Fatalf("GetClassAnalytics error = %v, want ErrExamPermissionDenied", err)
	}
}

func TestExamClassAnalyticsExcludesWithdrawnAssignments(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	practiceRepo := newStubPracticeRepo()
	class := seedExamClass(classRepo, "class-analytics", 10000, "teacher-1", "CLASSCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	published := seedAnalyticsAssignment(assignRepo, class, "assignment-published", "group-1", time.Now().Add(-time.Hour))
	withdrawn := seedAnalyticsAssignment(assignRepo, class, "assignment-withdrawn", "group-2", time.Now())
	withdrawn.Status = types.ExamAssignmentStatusWithdrawn
	seedAnalyticsAttempt(practiceRepo, published, "student-1", 5, 4, true, time.Now())
	seedAnalyticsAttempt(practiceRepo, withdrawn, "student-1", 5, 5, true, time.Now())
	svc := NewExamAnalyticsService(classRepo, assignRepo, practiceRepo)

	analytics, err := svc.GetClassAnalytics(ctx, 10000, "teacher-1", class.ID)

	if err != nil {
		t.Fatalf("GetClassAnalytics returned error: %v", err)
	}
	if analytics.AssignmentCount != 1 || len(analytics.Assignments) != 1 {
		t.Fatalf("withdrawn assignment leaked into analytics: %#v", analytics.Assignments)
	}
	if analytics.Assignments[0].Assignment.ID != published.ID {
		t.Fatalf("analytics assignment = %s, want %s", analytics.Assignments[0].Assignment.ID, published.ID)
	}
}

func TestExamClassAnalyticsAggregatesFrequentWrongQuestionsFromLatestAttempts(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	assignRepo := newStubExamAssignmentRepo(classRepo)
	practiceRepo := newStubPracticeRepo()
	class := seedExamClass(classRepo, "class-wrong", 10000, "teacher-1", "WRONGCODE")
	seedExamClassMember(classRepo, class.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, class.ID, 10000, "student-2", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	now := time.Now()
	assignment := seedAnalyticsAssignment(assignRepo, class, "assignment-wrong", "group-reading-a", now.Add(-time.Hour))
	oldAttempt := seedAnalyticsAttemptWithID(practiceRepo, assignment, "attempt-old", "student-1", 3, 0, true, now.Add(-50*time.Minute))
	latestFirst := seedAnalyticsAttemptWithID(practiceRepo, assignment, "attempt-latest-1", "student-1", 3, 2, true, now.Add(-30*time.Minute))
	latestSecond := seedAnalyticsAttemptWithID(practiceRepo, assignment, "attempt-latest-2", "student-2", 3, 2, true, now.Add(-20*time.Minute))
	seedAnalyticsAnswer(practiceRepo, oldAttempt, "question-old", "20", "Old attempt question", false)
	seedAnalyticsAnswer(practiceRepo, latestFirst, "question-21", "21", "Which team will play the most games?", false)
	seedAnalyticsAnswer(practiceRepo, latestFirst, "question-22", "22", "Which hotel is nearest?", true)
	seedAnalyticsAnswer(practiceRepo, latestSecond, "question-21", "21", "Which team will play the most games?", false)
	seedAnalyticsAnswer(practiceRepo, latestSecond, "question-22", "22", "Which hotel is nearest?", true)
	svc := NewExamAnalyticsService(classRepo, assignRepo, practiceRepo)

	analytics, err := svc.GetClassAnalytics(ctx, 10000, "teacher-1", class.ID)

	if err != nil {
		t.Fatalf("GetClassAnalytics returned error: %v", err)
	}
	if len(analytics.FrequentWrongQuestions) != 1 {
		t.Fatalf("frequent wrong count = %d, want 1: %#v", len(analytics.FrequentWrongQuestions), analytics.FrequentWrongQuestions)
	}
	wrong := analytics.FrequentWrongQuestions[0]
	if wrong.QuestionID != "question-21" || wrong.GroupID != assignment.GroupID || wrong.AnswerCount != 2 || wrong.WrongCount != 2 || wrong.AffectedStudentCount != 2 {
		t.Fatalf("frequent wrong = %#v, want question-21 with 2/2 answers and 2 students", wrong)
	}
	if wrong.WrongRate != 1 || !strings.Contains(wrong.Stem, "most games") {
		t.Fatalf("frequent wrong rate/stem = %#v", wrong)
	}
}

func TestExamAnalyticsListAnalyzableClassesFiltersStudentMembership(t *testing.T) {
	ctx := context.Background()
	classRepo := newFakeExamClassRepo()
	teacherClass := seedExamClass(classRepo, "class-teacher", 10000, "teacher-1", "TEACHER")
	assistantClass := seedExamClass(classRepo, "class-assistant", 10000, "owner-2", "ASSIST")
	studentClass := seedExamClass(classRepo, "class-student", 10000, "owner-3", "STUDENT")
	seedExamClassMember(classRepo, teacherClass.ID, 10000, "teacher-1", types.ExamClassRoleTeacher, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, assistantClass.ID, 10000, "teacher-1", types.ExamClassRoleAssistant, types.ExamClassMemberStatusActive)
	seedExamClassMember(classRepo, studentClass.ID, 10000, "teacher-1", types.ExamClassRoleStudent, types.ExamClassMemberStatusActive)
	svc := NewExamAnalyticsService(classRepo, newStubExamAssignmentRepo(classRepo), newStubPracticeRepo())

	classes, err := svc.ListAnalyzableClasses(ctx, 10000, "teacher-1")

	if err != nil {
		t.Fatalf("ListAnalyzableClasses returned error: %v", err)
	}
	if len(classes) != 2 {
		t.Fatalf("analyzable classes = %#v, want teacher and assistant classes", classes)
	}
	for _, class := range classes {
		if class.ID == studentClass.ID {
			t.Fatalf("student membership must not be analyzable: %#v", class)
		}
	}
}

func TestAggregateFrequentWrongQuestionsSortsQuestionNumbersDescendingNaturally(t *testing.T) {
	attemptUsers := map[string]analyticsAttemptContext{"attempt-1": {UserID: "student-1"}}
	answers := []*types.ExamPracticeAnswer{
		{AttemptID: "attempt-1", QuestionID: "question-2", QuestionNo: "2", IsCorrect: false},
		{AttemptID: "attempt-1", QuestionID: "question-3", QuestionNo: "3", IsCorrect: false},
		{AttemptID: "attempt-1", QuestionID: "question-10", QuestionNo: "10", IsCorrect: false},
	}

	items := aggregateFrequentWrongQuestions(answers, attemptUsers)

	if len(items) != 3 || items[0].QuestionNo != "10" || items[1].QuestionNo != "3" || items[2].QuestionNo != "2" {
		t.Fatalf("question order = %#v, want 10 then 3 then 2", items)
	}
}

func seedAnalyticsAssignment(repo *stubExamAssignmentRepo, class *types.ExamClass, id string, groupID string, createdAt time.Time) *types.ExamClassAssignment {
	assignment := &types.ExamClassAssignment{
		ID:              id,
		TenantID:        class.TenantID,
		ClassID:         class.ID,
		SpaceID:         class.SpaceID,
		QuestionBankID:  "bank-" + groupID,
		GroupID:         groupID,
		Title:           "Assignment " + id,
		Status:          types.ExamAssignmentStatusPublished,
		CreatedByUserID: class.OwnerUserID,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	repo.assignments = append(repo.assignments, assignment)
	return assignment
}

func seedAnalyticsAttempt(repo *stubPracticeRepo, assignment *types.ExamClassAssignment, userID string, questionCount int, correctCount int, completed bool, createdAt time.Time) {
	seedAnalyticsAttemptWithID(repo, assignment, assignment.ID+"-"+userID, userID, questionCount, correctCount, completed, createdAt)
}

func seedAnalyticsAttemptWithID(repo *stubPracticeRepo, assignment *types.ExamClassAssignment, attemptID string, userID string, questionCount int, correctCount int, completed bool, createdAt time.Time) *types.ExamPracticeAttempt {
	assignmentID := assignment.ID
	status := types.ExamPracticeAttemptStatusInProgress
	var completedAt *time.Time
	if completed {
		status = types.ExamPracticeAttemptStatusCompleted
		doneAt := createdAt.Add(15 * time.Minute)
		completedAt = &doneAt
	}
	attempt := &types.ExamPracticeAttempt{
		ID:             attemptID,
		TenantID:       assignment.TenantID,
		UserID:         userID,
		SpaceID:        assignment.SpaceID,
		QuestionBankID: assignment.QuestionBankID,
		GroupID:        assignment.GroupID,
		AssignmentID:   &assignmentID,
		Status:         status,
		QuestionCount:  questionCount,
		AnsweredCount:  questionCount,
		CorrectCount:   correctCount,
		StartedAt:      createdAt,
		CompletedAt:    completedAt,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	repo.attempts = append(repo.attempts, attempt)
	return attempt
}

func seedAnalyticsAnswer(repo *stubPracticeRepo, attempt *types.ExamPracticeAttempt, questionID string, questionNo string, stem string, correct bool) {
	repo.answers = append(repo.answers, &types.ExamPracticeAnswer{
		ID:         attempt.ID + "-" + questionID,
		TenantID:   attempt.TenantID,
		AttemptID:  attempt.ID,
		QuestionID: questionID,
		QuestionNo: questionNo,
		IsCorrect:  correct,
		QuestionSnapshot: types.JSONMap{
			"question_no": questionNo,
			"stem":        stem,
		},
		AnsweredAt: attempt.UpdatedAt,
	})
}

func findAnalyticsMember(items []*types.ExamClassAnalyticsMember, userID string) *types.ExamClassAnalyticsMember {
	for _, item := range items {
		if item != nil && item.Member != nil && item.Member.UserID == userID {
			return item
		}
	}
	return nil
}

func findAnalyticsAssignment(items []*types.ExamClassAnalyticsAssignment, assignmentID string) *types.ExamClassAnalyticsAssignment {
	for _, item := range items {
		if item != nil && item.Assignment != nil && item.Assignment.ID == assignmentID {
			return item
		}
	}
	return nil
}

package service

import (
	"context"
	"errors"
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
	assignmentID := assignment.ID
	status := types.ExamPracticeAttemptStatusInProgress
	var completedAt *time.Time
	if completed {
		status = types.ExamPracticeAttemptStatusCompleted
		doneAt := createdAt.Add(15 * time.Minute)
		completedAt = &doneAt
	}
	repo.attempts = append(repo.attempts, &types.ExamPracticeAttempt{
		ID:             assignment.ID + "-" + userID,
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

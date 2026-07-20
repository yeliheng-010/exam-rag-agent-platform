package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExamAssignmentAttemptArchivedClassRejectsNewAttempt(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	fixture.assignmentRepo.classRepo.classes["class-1"].Status = types.ExamClassStatusArchived

	result, err := fixture.svc.CreateAssignmentAttempt(
		context.Background(), 10000, "student-1", "assignment-1",
	)

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrExamStateConflict)
	require.Empty(t, fixture.practiceRepo.attempts)
}

func TestExamAssignmentNotificationArchivedClassCannotStart(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	fixture.assignmentRepo.classRepo.classes["class-1"].Status = types.ExamClassStatusArchived
	fixture.notificationRepo.notifications = append(
		fixture.notificationRepo.notifications,
		archivedAssignmentNotification(fixture),
	)

	result, err := fixture.svc.ListAssignmentNotifications(
		context.Background(), 10000, "student-1", 50,
	)

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.False(t, result.Items[0].CanStart)
	require.Empty(t, result.Items[0].LastAttemptID)
	require.Equal(t, 1, fixture.assignmentRepo.classRepo.listByIDsCalls)
}

func TestExamAssignmentNotificationArchivedClassKeepsExistingAttempt(t *testing.T) {
	fixture := newAssignmentLifecycleFixture(t)
	fixture.assignmentRepo.classRepo.classes["class-1"].Status = types.ExamClassStatusArchived
	fixture.notificationRepo.notifications = append(
		fixture.notificationRepo.notifications,
		archivedAssignmentNotification(fixture),
	)
	assignmentID := "assignment-1"
	fixture.practiceRepo.attempts = append(fixture.practiceRepo.attempts, &types.ExamPracticeAttempt{
		ID: "attempt-1", TenantID: 10000, UserID: "student-1", AssignmentID: &assignmentID,
		GroupID: "group-1", Status: types.ExamPracticeAttemptStatusInProgress, CreatedAt: fixture.now,
	})

	result, err := fixture.svc.ListAssignmentNotifications(
		context.Background(), 10000, "student-1", 50,
	)

	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.False(t, result.Items[0].CanStart)
	require.Equal(t, "attempt-1", result.Items[0].LastAttemptID)
}

func archivedAssignmentNotification(fixture *assignmentLifecycleFixture) *types.ExamAssignmentNotification {
	return &types.ExamAssignmentNotification{
		ID: "notification-1", TenantID: 10000, ClassID: "class-1", AssignmentID: "assignment-1",
		GroupID: "group-1", RecipientUserID: "student-1", ActorUserID: "teacher-1",
		Kind: types.ExamAssignmentNotificationKindPublished, Title: "New assignment", Content: "Start now",
		CreatedAt: fixture.now,
	}
}

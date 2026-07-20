package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamAssignmentNotificationRepositoryListsRecipientScope(t *testing.T) {
	db, repo := setupAssignmentNotificationRepository(t, "list")
	now := time.Now().UTC()
	seedAssignmentNotification(t, db, "older", 10000, "student-1", types.ExamAssignmentNotificationKindPublished, now.Add(-time.Hour), nil)
	readAt := now.Add(-time.Minute)
	seedAssignmentNotification(t, db, "newer", 10000, "student-1", types.ExamAssignmentNotificationKindWithdrawn, now, &readAt)
	seedAssignmentNotification(t, db, "other-user", 10000, "student-2", types.ExamAssignmentNotificationKindPublished, now, nil)
	seedAssignmentNotification(t, db, "other-tenant", 20000, "student-1", types.ExamAssignmentNotificationKindPublished, now, nil)

	items, unreadCount, err := repo.ListForRecipient(context.Background(), 10000, "student-1", 50)

	require.NoError(t, err)
	require.EqualValues(t, 1, unreadCount)
	require.Len(t, items, 2)
	require.Equal(t, "newer", items[0].ID)
	require.Equal(t, "older", items[1].ID)
}

func TestExamAssignmentNotificationRepositoryMarksOnlyRecipientRows(t *testing.T) {
	db, repo := setupAssignmentNotificationRepository(t, "read")
	now := time.Now().UTC()
	seedAssignmentNotification(t, db, "notification-1", 10000, "student-1", types.ExamAssignmentNotificationKindPublished, now, nil)
	seedAssignmentNotification(t, db, "notification-2", 10000, "student-1", types.ExamAssignmentNotificationKindReminder, now, nil)

	err := repo.MarkRead(context.Background(), 10000, "student-2", "notification-1", now.Add(time.Minute))
	require.ErrorIs(t, err, ErrExamAssignmentNotificationNotFound)

	require.NoError(t, repo.MarkRead(context.Background(), 10000, "student-1", "notification-1", now.Add(time.Minute)))
	require.NoError(t, repo.MarkAllRead(context.Background(), 10000, "student-1", now.Add(2*time.Minute)))

	items, unreadCount, err := repo.ListForRecipient(context.Background(), 10000, "student-1", 50)
	require.NoError(t, err)
	require.Zero(t, unreadCount)
	require.Len(t, items, 2)
	require.NotNil(t, items[0].ReadAt)
	require.NotNil(t, items[1].ReadAt)
}

func TestExamAssignmentNotificationRepositoryListsLatestReminders(t *testing.T) {
	db, repo := setupAssignmentNotificationRepository(t, "latest-reminders")
	now := time.Now().UTC()
	seedAssignmentNotification(t, db, "student-1-old", 10000, "student-1", types.ExamAssignmentNotificationKindReminder, now.Add(-time.Hour), nil)
	seedAssignmentNotification(t, db, "student-1-new", 10000, "student-1", types.ExamAssignmentNotificationKindReminder, now, nil)
	seedAssignmentNotification(t, db, "student-2-publish", 10000, "student-2", types.ExamAssignmentNotificationKindPublished, now, nil)

	latest, err := repo.ListLatestReminders(
		context.Background(), 10000, "assignment-1", []string{"student-1", "student-2"},
	)

	require.NoError(t, err)
	require.Len(t, latest, 1)
	require.WithinDuration(t, now, latest["student-1"], time.Second)
}

func TestExamAssignmentNotificationRepositoryCreatesOnlyEligibleReminders(t *testing.T) {
	db, repo, assignment, now := setupReminderRepository(t, "eligible", types.ExamAssignmentStatusPublished, time.Hour)
	assignmentID := assignment.ID
	require.NoError(t, db.Create(&types.ExamPracticeAttempt{
		ID: "completed-attempt", TenantID: 10000, UserID: "student-completed",
		SpaceID: assignment.SpaceID, QuestionBankID: assignment.QuestionBankID,
		GroupID: assignment.GroupID, AssignmentID: &assignmentID,
		Status: types.ExamPracticeAttemptStatusCompleted, CreatedAt: now,
	}).Error)
	seedAssignmentNotification(
		t, db, "recent-reminder", 10000, "student-cooldown",
		types.ExamAssignmentNotificationKindReminder, now.Add(-time.Hour), nil,
	)
	candidates := testReminderCandidates(assignment, now, "student-new", "student-completed", "student-cooldown")

	result, err := repo.CreateRemindersIfEligible(context.Background(), 10000, assignment.ID, candidates, now)

	require.NoError(t, err)
	require.Equal(t, 1, result.SentCount)
	require.Equal(t, 1, result.CompletedSkippedCount)
	require.Equal(t, 1, result.CooldownSkippedCount)
	require.Equal(t, []string{"student-new"}, result.SentUserIDs)
	var stored []*types.ExamAssignmentNotification
	require.NoError(t, db.Where("kind = ?", types.ExamAssignmentNotificationKindReminder).Order("created_at").Find(&stored).Error)
	require.Len(t, stored, 2)
	require.Equal(t, "student-new", stored[1].RecipientUserID)
}

func TestExamAssignmentNotificationRepositoryRejectsClosedAssignmentReminders(t *testing.T) {
	tests := []struct {
		name      string
		status    types.ExamAssignmentStatus
		dueOffset time.Duration
	}{
		{name: "withdrawn", status: types.ExamAssignmentStatusWithdrawn, dueOffset: time.Hour},
		{name: "expired", status: types.ExamAssignmentStatusPublished, dueOffset: -time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, repo, assignment, now := setupReminderRepository(t, tt.name, tt.status, tt.dueOffset)
			candidates := testReminderCandidates(assignment, now, "student-1")

			result, err := repo.CreateRemindersIfEligible(context.Background(), 10000, assignment.ID, candidates, now)

			require.Nil(t, result)
			require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
			var count int64
			require.NoError(t, db.Model(&types.ExamAssignmentNotification{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func setupAssignmentNotificationRepository(t *testing.T, databaseName string) (*gorm.DB, *examAssignmentNotificationRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-notification-"+databaseName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamAssignmentNotification{}))
	return db, &examAssignmentNotificationRepository{db: db}
}

func setupReminderRepository(
	t *testing.T,
	databaseName string,
	status types.ExamAssignmentStatus,
	dueOffset time.Duration,
) (*gorm.DB, *examAssignmentNotificationRepository, *types.ExamClassAssignment, time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-reminder-"+databaseName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.ExamClassAssignment{}, &types.ExamPracticeAttempt{}, &types.ExamAssignmentNotification{},
	))
	now := time.Now().UTC()
	assignment := testExamAssignment("assignment-1", "group-1", status, now)
	dueAt := now.Add(dueOffset)
	assignment.DueAt = &dueAt
	require.NoError(t, db.Create(assignment).Error)
	return db, &examAssignmentNotificationRepository{db: db}, assignment, now
}

func testReminderCandidates(
	assignment *types.ExamClassAssignment,
	createdAt time.Time,
	userIDs ...string,
) []*types.ExamAssignmentNotification {
	items := make([]*types.ExamAssignmentNotification, 0, len(userIDs))
	for index, userID := range userIDs {
		items = append(items, &types.ExamAssignmentNotification{
			ID: "candidate-" + userID, TenantID: assignment.TenantID,
			ClassID: assignment.ClassID, AssignmentID: assignment.ID, GroupID: assignment.GroupID,
			RecipientUserID: userID, ActorUserID: "teacher-1",
			Kind:  types.ExamAssignmentNotificationKindReminder,
			Title: "Reminder", Content: "Reminder", CreatedAt: createdAt.Add(time.Duration(index) * time.Millisecond),
		})
	}
	return items
}

func seedAssignmentNotification(
	t *testing.T,
	db *gorm.DB,
	id string,
	tenantID uint64,
	recipientUserID string,
	kind types.ExamAssignmentNotificationKind,
	createdAt time.Time,
	readAt *time.Time,
) {
	t.Helper()
	require.NoError(t, db.Create(&types.ExamAssignmentNotification{
		ID: id, TenantID: tenantID, ClassID: "class-1", AssignmentID: "assignment-1", GroupID: "group-1",
		RecipientUserID: recipientUserID, ActorUserID: "teacher-1", Kind: kind,
		Title: id, Content: id, ReadAt: readAt, CreatedAt: createdAt,
	}).Error)
}

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

func TestExamPracticeRepositoryCreatesAttemptForOpenAssignment(t *testing.T) {
	db, repo, assignment, now := setupAssignmentAttemptRepository(t, "open", types.ExamAssignmentStatusPublished, time.Hour)
	attempt := testAssignmentAttempt(assignment, now)

	err := repo.CreateAssignmentAttemptIfOpen(context.Background(), attempt, now)

	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&types.ExamPracticeAttempt{}).Where("id = ?", attempt.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestExamPracticeRepositoryRejectsAttemptForWithdrawnAssignment(t *testing.T) {
	db, repo, assignment, now := setupAssignmentAttemptRepository(t, "withdrawn", types.ExamAssignmentStatusWithdrawn, time.Hour)
	attempt := testAssignmentAttempt(assignment, now)

	err := repo.CreateAssignmentAttemptIfOpen(context.Background(), attempt, now)

	require.ErrorIs(t, err, ErrExamAssignmentAttemptClosed)
	var count int64
	require.NoError(t, db.Model(&types.ExamPracticeAttempt{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestExamPracticeRepositoryRejectsAttemptForExpiredAssignment(t *testing.T) {
	db, repo, assignment, now := setupAssignmentAttemptRepository(t, "expired", types.ExamAssignmentStatusPublished, -time.Minute)
	attempt := testAssignmentAttempt(assignment, now)

	err := repo.CreateAssignmentAttemptIfOpen(context.Background(), attempt, now)

	require.ErrorIs(t, err, ErrExamAssignmentAttemptClosed)
	var count int64
	require.NoError(t, db.Model(&types.ExamPracticeAttempt{}).Count(&count).Error)
	require.Zero(t, count)
}

func setupAssignmentAttemptRepository(
	t *testing.T,
	databaseName string,
	status types.ExamAssignmentStatus,
	dueOffset time.Duration,
) (*gorm.DB, *examPracticeRepository, *types.ExamClassAssignment, time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:exam-practice-assignment-"+databaseName+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}, &types.ExamPracticeAttempt{}))
	now := time.Now().UTC()
	assignment := testExamAssignment("assignment-1", "group-1", status, now)
	dueAt := now.Add(dueOffset)
	assignment.DueAt = &dueAt
	require.NoError(t, db.Create(assignment).Error)
	return db, &examPracticeRepository{db: db}, assignment, now
}

func testAssignmentAttempt(assignment *types.ExamClassAssignment, now time.Time) *types.ExamPracticeAttempt {
	assignmentID := assignment.ID
	return &types.ExamPracticeAttempt{
		ID: "attempt-1", TenantID: assignment.TenantID, UserID: "student-1",
		SpaceID: assignment.SpaceID, QuestionBankID: assignment.QuestionBankID,
		GroupID: assignment.GroupID, AssignmentID: &assignmentID,
		Status: types.ExamPracticeAttemptStatusInProgress, QuestionCount: 1,
		StartedAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

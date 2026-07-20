package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamAssignmentRepositoryFiltersLifecycleStatuses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-lifecycle?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, db.Create(testExamAssignment("published-1", "group-1", types.ExamAssignmentStatusPublished, now)).Error)
	require.NoError(t, db.Create(testExamAssignment("withdrawn-1", "group-2", types.ExamAssignmentStatusWithdrawn, now.Add(time.Minute))).Error)
	require.NoError(t, db.Create(testExamAssignment("archived-1", "group-3", types.ExamAssignmentStatusArchived, now.Add(2*time.Minute))).Error)

	items, err := repo.ListAssignmentsByClass(ctx, 10000, "class-1", []types.ExamAssignmentStatus{
		types.ExamAssignmentStatusPublished,
		types.ExamAssignmentStatusWithdrawn,
	}, 50)

	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "withdrawn-1", items[0].ID)
	require.Equal(t, "published-1", items[1].ID)
}

func TestExamAssignmentRepositoryTransitionRequiresExpectedStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-transition?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	ctx := context.Background()
	require.NoError(t, db.Create(testExamAssignment(
		"assignment-1", "group-1", types.ExamAssignmentStatusPublished, time.Now().UTC(),
	)).Error)

	err = repo.TransitionAssignmentStatus(
		ctx,
		10000,
		"class-1",
		"assignment-1",
		types.ExamAssignmentStatusWithdrawn,
		types.ExamAssignmentStatusPublished,
		time.Now().UTC(),
	)

	require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
	stored, err := repo.GetAssignmentByIDAndTenant(ctx, 10000, "assignment-1")
	require.NoError(t, err)
	require.Equal(t, types.ExamAssignmentStatusPublished, stored.Status)
}

func TestExamAssignmentRepositoryRepublishRejectsExpiredDueAtAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-expired-republish?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	now := time.Now().UTC()
	assignment := testExamAssignment("assignment-1", "group-1", types.ExamAssignmentStatusWithdrawn, now)
	pastDueAt := now.Add(-time.Minute)
	assignment.DueAt = &pastDueAt
	require.NoError(t, db.Create(assignment).Error)

	err = repo.TransitionAssignmentStatus(
		context.Background(), 10000, "class-1", assignment.ID,
		types.ExamAssignmentStatusWithdrawn, types.ExamAssignmentStatusPublished, now,
	)

	require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
}

func TestExamAssignmentRepositoryUpdatesMetadataOnlyInAllowedState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-metadata?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, db.Create(testExamAssignment(
		"assignment-1", "group-1", types.ExamAssignmentStatusPublished, now,
	)).Error)
	dueAt := now.Add(48 * time.Hour)

	err = repo.UpdateAssignmentMetadata(
		ctx,
		10000,
		"class-1",
		"assignment-1",
		[]types.ExamAssignmentStatus{types.ExamAssignmentStatusPublished},
		"Updated title",
		"Updated instructions",
		&dueAt,
		now.Add(time.Minute),
	)
	require.NoError(t, err)
	stored, err := repo.GetAssignmentByIDAndTenant(ctx, 10000, "assignment-1")
	require.NoError(t, err)
	require.Equal(t, "Updated title", stored.Title)
	require.Equal(t, "Updated instructions", stored.Instructions)
	require.WithinDuration(t, dueAt, *stored.DueAt, time.Second)

	err = repo.UpdateAssignmentMetadata(
		ctx,
		10000,
		"class-1",
		"assignment-1",
		[]types.ExamAssignmentStatus{types.ExamAssignmentStatusWithdrawn},
		"Rejected title",
		"Rejected instructions",
		nil,
		now.Add(2*time.Minute),
	)
	require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
}

func TestExamAssignmentRepositoryUpdateRejectsExpiredPublishedAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment-expired-update?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}))
	repo := &examAssignmentRepository{db: db}
	now := time.Now().UTC()
	assignment := testExamAssignment("assignment-1", "group-1", types.ExamAssignmentStatusPublished, now)
	pastDueAt := now.Add(-time.Minute)
	assignment.DueAt = &pastDueAt
	require.NoError(t, db.Create(assignment).Error)

	err = repo.UpdateAssignmentMetadata(
		context.Background(), 10000, "class-1", assignment.ID,
		[]types.ExamAssignmentStatus{types.ExamAssignmentStatusPublished},
		"Rejected title", "", nil, now,
	)

	require.ErrorIs(t, err, ErrExamClassAssignmentStateConflict)
}

func TestExamAssignmentRepositoryListsPublishedCandidateGroupsWithoutHistoryLimit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:exam-assignment?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.ExamClassAssignment{}, &types.QuestionGroup{}))
	repo := &examAssignmentRepository{db: db}
	now := time.Now().UTC()

	for i := 0; i < 100; i++ {
		groupID := fmt.Sprintf("other-group-%03d", i)
		require.NoError(t, db.Create(testAssignmentQuestionGroup(groupID, now)).Error)
		require.NoError(t, db.Create(testExamAssignment(
			fmt.Sprintf("assignment-%03d", i),
			groupID,
			types.ExamAssignmentStatusPublished,
			now.Add(time.Duration(i)*time.Minute),
		)).Error)
	}
	require.NoError(t, db.Create(testAssignmentQuestionGroup("old-published", now)).Error)
	require.NoError(t, db.Create(testAssignmentQuestionGroup("archived-candidate", now)).Error)
	require.NoError(t, db.Create(testExamAssignment("assignment-old", "old-published", types.ExamAssignmentStatusPublished, now.Add(-time.Hour))).Error)
	require.NoError(t, db.Create(testExamAssignment("assignment-archived", "archived-candidate", types.ExamAssignmentStatusArchived, now)).Error)

	groups, err := repo.ListPublishedGroupsByClass(context.Background(), 10000, "class-1")

	require.NoError(t, err)
	require.Len(t, groups, 101)
	groupIDs := make(map[string]bool, len(groups))
	for _, group := range groups {
		groupIDs[group.ID] = true
	}
	require.True(t, groupIDs["old-published"])
	require.False(t, groupIDs["archived-candidate"])
}

func testAssignmentQuestionGroup(id string, createdAt time.Time) *types.QuestionGroup {
	subjectID := "subject-english"
	return &types.QuestionGroup{
		ID: id, TenantID: 10000, SpaceID: "class-space", QuestionBankID: "bank-1",
		DomainID: "domain-gaokao", SubjectID: &subjectID, GroupType: "reading_passage",
		Title: id, MaterialText: id, MaterialFormat: "plain_text",
		AssetRefs: types.JSON(`[]`), SourceChunkIDs: types.JSON(`[]`),
		ReviewStatus: types.ExamReviewStatusPrivate, Status: "active", CreatedByUserID: "teacher-1",
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func testExamAssignment(
	id string,
	groupID string,
	status types.ExamAssignmentStatus,
	createdAt time.Time,
) *types.ExamClassAssignment {
	return &types.ExamClassAssignment{
		ID: id, TenantID: 10000, ClassID: "class-1", GroupID: groupID,
		Title: id, Status: status, CreatedByUserID: "teacher-1",
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

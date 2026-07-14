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

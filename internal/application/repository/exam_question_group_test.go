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

func TestExamQuestionRepository_ListQuestionGroupDetailsByBankWrapsLegacyQuestions(t *testing.T) {
	db := newExamQuestionGroupTestDB(t)
	repo := NewExamQuestionRepository(db)
	now := time.Now()

	question := &types.Question{
		ID:               "question-1",
		TenantID:         10000,
		QuestionBankID:   "bank-1",
		DomainID:         "gaokao",
		Stem:             "What is the answer?",
		QuestionMetadata: types.JSONMap{},
		Difficulty:       "unknown",
		ReviewStatus:     types.ExamReviewStatusPrivate,
		Status:           "active",
		CreatedByUserID:  "teacher-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, db.Create(question).Error)

	groups, err := repo.ListQuestionGroupDetailsByBank(context.Background(), 10000, "bank-1")

	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "single_question", groups[0].Group.GroupType)
	require.Equal(t, question.Stem, groups[0].Group.MaterialText)
	require.Len(t, groups[0].Questions, 1)
	require.Equal(t, "question-1", groups[0].Questions[0].Question.ID)
}

func newExamQuestionGroupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.QuestionBank{},
		&types.QuestionGroup{},
		&types.QuestionGroupAsset{},
		&types.Question{},
		&types.QuestionOption{},
		&types.QuestionAnswer{},
		&types.QuestionExplanation{},
		&types.QuestionChunkRef{},
	))
	return db
}

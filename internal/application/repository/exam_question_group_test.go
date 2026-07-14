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

func TestExamQuestionRepository_FindQuestionGroupDetailByChunkIDs(t *testing.T) {
	db := newExamQuestionGroupTestDB(t)
	repo := NewExamQuestionRepository(db)
	now := time.Now()
	groupID := "group-reading-a"
	questionID := "question-21"

	require.NoError(t, repo.CreateQuestionGroupDetail(context.Background(), &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:              groupID,
			TenantID:        10000,
			SpaceID:         "space-1",
			QuestionBankID:  "bank-1",
			DomainID:        "gaokao",
			GroupType:       "reading_passage",
			Title:           "Reading A",
			MaterialText:    "SoFi Stadium is the go-to destination.",
			MaterialFormat:  "plain_text",
			AssetRefs:       types.JSON(`[]`),
			SourceChunkIDs:  types.JSON(`["chunk-a"]`),
			ReviewStatus:    types.ExamReviewStatusPrivate,
			Status:          "active",
			CreatedByUserID: "teacher-1",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		Questions: []*types.QuestionDetail{
			{
				Question: &types.Question{
					ID:               questionID,
					TenantID:         10000,
					QuestionBankID:   "bank-1",
					DomainID:         "gaokao",
					GroupID:          &groupID,
					QuestionNo:       "21",
					OrderInGroup:     1,
					Stem:             "Which team will play the most games?",
					QuestionMetadata: types.JSONMap{},
					Difficulty:       "unknown",
					ReviewStatus:     types.ExamReviewStatusPrivate,
					Status:           "active",
					CreatedByUserID:  "teacher-1",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
				Options: []*types.QuestionOption{
					{ID: "option-21-a", QuestionID: questionID, OptionKey: "A", Content: "Dallas Cowboys", SortOrder: 1},
					{ID: "option-21-b", QuestionID: questionID, OptionKey: "B", Content: "Los Angeles Rams", SortOrder: 2},
				},
				Answers: []*types.QuestionAnswer{
					{ID: "answer-21", QuestionID: questionID, AnswerText: "B", IsCorrect: true, CreatedAt: now},
				},
				Explanations: []*types.QuestionExplanation{
					{ID: "explanation-21", QuestionID: questionID, ExplanationText: "The Rams are listed more often.", SourceType: "model", CreatedAt: now},
				},
				ChunkRefs: []*types.QuestionChunkRef{
					{QuestionID: questionID, ChunkID: "chunk-21", RefType: "evidence", Confidence: 0.9, CreatedAt: now},
				},
			},
		},
	}))

	detail, err := repo.FindQuestionGroupDetailByChunkIDs(context.Background(), 10000, []string{"missing", "chunk-21"})

	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Equal(t, groupID, detail.Group.ID)
	require.Len(t, detail.Questions, 1)
	require.Equal(t, questionID, detail.Questions[0].Question.ID)
	require.Len(t, detail.Questions[0].Options, 2)
	require.Equal(t, "B", detail.Questions[0].Answers[0].AnswerText)
	require.Equal(t, "chunk-21", detail.Questions[0].ChunkRefs[0].ChunkID)
}

func TestExamQuestionRepository_FindQuestionGroupDetailByChunkIDsMatchesGroupSourceChunks(t *testing.T) {
	db := newExamQuestionGroupTestDB(t)
	repo := NewExamQuestionRepository(db)
	now := time.Now()
	groupID := "group-reading-source-a"
	questionID := "question-source-21"

	require.NoError(t, repo.CreateQuestionGroupDetail(context.Background(), &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:              groupID,
			TenantID:        10000,
			SpaceID:         "space-1",
			QuestionBankID:  "bank-1",
			DomainID:        "gaokao",
			GroupType:       "reading_passage",
			Title:           "Reading A",
			MaterialText:    "SoFi Stadium is the go-to destination.",
			MaterialFormat:  "plain_text",
			AssetRefs:       types.JSON(`[]`),
			SourceChunkIDs:  types.JSON(`["chunk-a", "chunk-a-body"]`),
			ReviewStatus:    types.ExamReviewStatusPrivate,
			Status:          "active",
			CreatedByUserID: "teacher-1",
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		Questions: []*types.QuestionDetail{
			{
				Question: &types.Question{
					ID:               questionID,
					TenantID:         10000,
					QuestionBankID:   "bank-1",
					DomainID:         "gaokao",
					GroupID:          &groupID,
					QuestionNo:       "21",
					OrderInGroup:     1,
					Stem:             "Which team will play the most games?",
					QuestionMetadata: types.JSONMap{},
					Difficulty:       "unknown",
					ReviewStatus:     types.ExamReviewStatusPrivate,
					Status:           "active",
					CreatedByUserID:  "teacher-1",
					CreatedAt:        now,
					UpdatedAt:        now,
				},
			},
		},
	}))

	detail, err := repo.FindQuestionGroupDetailByChunkIDs(context.Background(), 10000, []string{"missing", "chunk-a-body"})

	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Equal(t, groupID, detail.Group.ID)
	require.Len(t, detail.Questions, 1)
	require.Equal(t, questionID, detail.Questions[0].Question.ID)
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

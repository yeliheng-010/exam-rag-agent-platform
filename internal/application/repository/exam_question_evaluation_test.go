package repository

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExamQuestionRepository_ListEvaluationQuestionGroupCandidatesUsesSourceFileHash(t *testing.T) {
	t.Parallel()

	db := newExamQuestionEvaluationTestDB(t)
	repo := &examQuestionRepository{db: db}
	now := time.Now().UTC()

	require.NoError(t, db.Create([]*types.Knowledge{
		{ID: "source-knowledge", TenantID: 10000, KnowledgeBaseID: "source-kb", FileHash: "same-file", CreatedAt: now, UpdatedAt: now},
		{ID: "eval-knowledge", TenantID: 10000, KnowledgeBaseID: "eval-kb", FileHash: "same-file", CreatedAt: now, UpdatedAt: now},
		{ID: "other-knowledge", TenantID: 10000, KnowledgeBaseID: "other-kb", FileHash: "other-file", CreatedAt: now, UpdatedAt: now},
	}).Error)
	require.NoError(t, db.Create([]*types.Chunk{
		{ID: "source-chunk", TenantID: 10000, KnowledgeID: "source-knowledge", KnowledgeBaseID: "source-kb", Content: "source", CreatedAt: now, UpdatedAt: now},
		{ID: "other-chunk", TenantID: 10000, KnowledgeID: "other-knowledge", KnowledgeBaseID: "other-kb", Content: "other", CreatedAt: now, UpdatedAt: now},
	}).Error)
	createEvaluationQuestionGroup(t, db, "bank-source", "group-source", "question-source", "source-chunk", now)
	createEvaluationQuestionGroup(t, db, "bank-other", "group-other", "question-other", "other-chunk", now)

	candidates, err := repo.ListEvaluationQuestionGroupCandidates(context.Background(), 10000, []string{"eval-kb"})

	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "group-source", candidates[0].Group.ID)
	require.Equal(t, "question-source", candidates[0].Questions[0].Question.ID)
}

func TestExamQuestionRepository_ListEvaluationQuestionGroupCandidatesRejectsCrossTenantHash(t *testing.T) {
	t.Parallel()

	db := newExamQuestionEvaluationTestDB(t)
	repo := &examQuestionRepository{db: db}
	now := time.Now().UTC()
	require.NoError(t, db.Create([]*types.Knowledge{
		{ID: "tenant-source", TenantID: 20000, KnowledgeBaseID: "source-kb", FileHash: "shared-file", CreatedAt: now, UpdatedAt: now},
		{ID: "tenant-eval", TenantID: 10000, KnowledgeBaseID: "eval-kb", FileHash: "shared-file", CreatedAt: now, UpdatedAt: now},
	}).Error)
	require.NoError(t, db.Create(&types.Chunk{
		ID: "tenant-source-chunk", TenantID: 20000, KnowledgeID: "tenant-source",
		KnowledgeBaseID: "source-kb", Content: "source", CreatedAt: now, UpdatedAt: now,
	}).Error)
	createEvaluationQuestionGroupForTenant(t, db, 20000, "bank-tenant", "group-tenant", "question-tenant", "tenant-source-chunk", now)

	candidates, err := repo.ListEvaluationQuestionGroupCandidates(context.Background(), 10000, []string{"eval-kb"})

	require.NoError(t, err)
	require.Empty(t, candidates)
}

func TestExamQuestionRepository_ListEvaluationQuestionGroupCandidatesForSourcePhrasesNarrowsToMatchedSourceFile(t *testing.T) {
	t.Parallel()

	db := newExamQuestionEvaluationTestDB(t)
	repo := &examQuestionRepository{db: db}
	now := time.Now().UTC()
	require.NoError(t, db.Create([]*types.Knowledge{
		{ID: "source-knowledge", TenantID: 10000, KnowledgeBaseID: "source-kb", FileHash: "same-file", CreatedAt: now, UpdatedAt: now},
		{ID: "eval-knowledge", TenantID: 10000, KnowledgeBaseID: "eval-kb", FileHash: "same-file", CreatedAt: now, UpdatedAt: now},
		{ID: "other-source-knowledge", TenantID: 10000, KnowledgeBaseID: "other-source-kb", FileHash: "other-file", CreatedAt: now, UpdatedAt: now},
		{ID: "other-eval-knowledge", TenantID: 10000, KnowledgeBaseID: "eval-kb", FileHash: "other-file", CreatedAt: now, UpdatedAt: now},
	}).Error)
	require.NoError(t, db.Create([]*types.Chunk{
		{ID: "source-prism", TenantID: 10000, KnowledgeID: "source-knowledge", KnowledgeBaseID: "source-kb", Content: "第9题：在 正三棱柱中，D 为 BC 中点。", CreatedAt: now, UpdatedAt: now},
		{ID: "source-sequence", TenantID: 10000, KnowledgeID: "other-source-knowledge", KnowledgeBaseID: "other-source-kb", Content: "第13题：若一个等比数列的各项均为正数。", CreatedAt: now, UpdatedAt: now},
	}).Error)
	createEvaluationQuestionGroup(t, db, "bank-source", "group-8", "question-8", "source-prism", now)
	createEvaluationQuestionGroup(t, db, "bank-source", "group-9", "question-9", "source-prism", now)
	createEvaluationQuestionGroup(t, db, "bank-source", "group-without-ref", "question-without-ref", "source-prism", now)
	require.NoError(t, db.Where("question_id = ?", "question-without-ref").Delete(&types.QuestionChunkRef{}).Error)
	createEvaluationQuestionGroup(t, db, "bank-other", "group-13", "question-13", "source-sequence", now)

	candidates, err := repo.ListEvaluationQuestionGroupCandidatesForSourcePhrases(
		context.Background(), 10000, []string{"eval-kb"}, []string{"在正三棱柱"},
	)

	require.NoError(t, err)
	ids := candidateGroupIDs(candidates)
	sort.Strings(ids)
	require.Equal(t, []string{"group-8", "group-9", "group-without-ref"}, ids)
}

func candidateGroupIDs(candidates []*types.QuestionGroupDetail) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.Group != nil {
			ids = append(ids, candidate.Group.ID)
		}
	}
	return ids
}

func createEvaluationQuestionGroup(
	t *testing.T,
	db *gorm.DB,
	bankID string,
	groupID string,
	questionID string,
	chunkID string,
	now time.Time,
) {
	t.Helper()
	createEvaluationQuestionGroupForTenant(t, db, 10000, bankID, groupID, questionID, chunkID, now)
}

func createEvaluationQuestionGroupForTenant(
	t *testing.T,
	db *gorm.DB,
	tenantID uint64,
	bankID string,
	groupID string,
	questionID string,
	chunkID string,
	now time.Time,
) {
	t.Helper()
	require.NoError(t, db.Create(&types.QuestionGroup{
		ID: groupID, TenantID: tenantID, QuestionBankID: bankID, Title: groupID,
		MaterialText: "unique material for " + groupID, AssetRefs: types.JSON(`[]`),
		SourceChunkIDs: types.JSON(`[]`), Status: "active", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&types.Question{
		ID: questionID, TenantID: tenantID, QuestionBankID: bankID, GroupID: &groupID,
		Stem: "unique stem for " + groupID, Status: "active", CreatedAt: now, UpdatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&types.QuestionChunkRef{
		QuestionID: questionID, ChunkID: chunkID, RefType: "evidence", Confidence: 1, CreatedAt: now,
	}).Error)
}

func newExamQuestionEvaluationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Knowledge{}, &types.Chunk{}, &types.QuestionGroup{}, &types.QuestionGroupAsset{},
		&types.Question{}, &types.QuestionOption{}, &types.QuestionAnswer{},
		&types.QuestionExplanation{}, &types.QuestionChunkRef{},
	))
	return db
}

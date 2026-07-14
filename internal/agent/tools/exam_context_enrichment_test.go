package tools

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubToolExamQuestionRepo struct {
	interfaces.ExamQuestionRepository
	detail *types.QuestionGroupDetail
	gotIDs []string
}

func (r *stubToolExamQuestionRepo) FindQuestionGroupDetailByChunkIDs(_ context.Context, _ uint64, chunkIDs []string) (*types.QuestionGroupDetail, error) {
	r.gotIDs = append([]string{}, chunkIDs...)
	return r.detail, nil
}

func TestKnowledgeSearchToolAppendExamContextBundlesPrefersStructuredGroup(t *testing.T) {
	repo := &stubToolExamQuestionRepo{detail: newToolStructuredReadingGroup()}
	tool := &KnowledgeSearchTool{
		questionRepo: repo,
		searchTargets: types.SearchTargets{
			{KnowledgeBaseID: "kb-1", TenantID: 10000},
		},
	}
	seed := &searchResultWithMeta{
		SearchResult: &types.SearchResult{
			ID:              "chunk-21",
			KnowledgeID:     "knowledge-1",
			KnowledgeBaseID: "kb-1",
			Content:         "raw retrieved content",
			Metadata:        map[string]string{"existing": "true"},
		},
		KnowledgeBaseID: "kb-1",
		SourceQuery:     "第一篇阅读",
		QueryType:       "vector",
	}

	results := tool.appendExamContextBundles(context.Background(), []string{"第一篇阅读的原文和答案是什么"}, []*searchResultWithMeta{seed})

	if len(results) == 0 || results[0] == nil || results[0].SearchResult == nil {
		t.Fatal("expected enriched result")
	}
	got := results[0]
	if !strings.Contains(got.Content, "[Exam Structured Context]") {
		t.Fatalf("expected structured context, got:\n%s", got.Content)
	}
	if got.Metadata["exam_context_mode"] != "structured" {
		t.Fatalf("mode = %q", got.Metadata["exam_context_mode"])
	}
	if got.Metadata["existing"] != "true" {
		t.Fatal("expected metadata to be preserved")
	}
	if got.QueryType != "exam_context" {
		t.Fatalf("query type = %q", got.QueryType)
	}
	if strings.Join(got.SubChunkID, ",") != "chunk-a1" {
		t.Fatalf("sub chunks = %v", got.SubChunkID)
	}
	if strings.Join(repo.gotIDs, ",") != "chunk-21" {
		t.Fatalf("repo chunk IDs = %v", repo.gotIDs)
	}
}

func newToolStructuredReadingGroup() *types.QuestionGroupDetail {
	now := time.Date(2026, 7, 8, 15, 4, 5, 0, time.UTC)
	groupID := "group-reading-a"
	questionID := "question-21"
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:             groupID,
			TenantID:       10000,
			QuestionBankID: "bank-english",
			GroupType:      "reading_passage",
			Title:          "SoFi Stadium Events",
			MaterialText:   "SoFi Stadium is the go-to destination.",
			SourceChunkIDs: types.JSON(`["chunk-a1"]`),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Questions: []*types.QuestionDetail{
			{
				Question: &types.Question{
					ID:           questionID,
					TenantID:     10000,
					GroupID:      &groupID,
					QuestionNo:   "21",
					OrderInGroup: 1,
					Stem:         "Which team will play the most games?",
					Status:       "active",
					CreatedAt:    now,
					UpdatedAt:    now,
				},
				Options: []*types.QuestionOption{
					{ID: "option-21-a", QuestionID: questionID, OptionKey: "A", Content: "Dallas Cowboys", SortOrder: 1},
					{ID: "option-21-b", QuestionID: questionID, OptionKey: "B", Content: "Los Angeles Rams", SortOrder: 2},
				},
				Answers: []*types.QuestionAnswer{
					{ID: "answer-21", QuestionID: questionID, AnswerText: "B", IsCorrect: true, CreatedAt: now},
				},
				ChunkRefs: []*types.QuestionChunkRef{
					{QuestionID: questionID, ChunkID: "chunk-21", RefType: "evidence", Confidence: 0.9, CreatedAt: now},
				},
			},
		},
	}
}

package chatpipeline

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubChatExamQuestionRepo struct {
	interfaces.ExamQuestionRepository
	detail  *types.QuestionGroupDetail
	matchID string
	calls   [][]string
	gotIDs  []string
}

func (r *stubChatExamQuestionRepo) FindQuestionGroupDetailByChunkIDs(_ context.Context, _ uint64, chunkIDs []string) (*types.QuestionGroupDetail, error) {
	ids := append([]string{}, chunkIDs...)
	r.gotIDs = ids
	r.calls = append(r.calls, ids)
	if r.matchID != "" && !containsExamContextString(ids, r.matchID) {
		return nil, nil
	}
	return r.detail, nil
}

func TestEnrichExamQuestionContextPrefersStructuredQuestionGroup(t *testing.T) {
	repo := &stubChatExamQuestionRepo{detail: newChatStructuredReadingGroup()}
	plugin := &PluginSearch{questionRepo: repo}
	chatManage := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			Query:    "第一篇阅读的原文和答案是什么",
			TenantID: 10000,
			SearchTargets: types.SearchTargets{
				{KnowledgeBaseID: "kb-1", TenantID: 10000},
			},
		},
	}
	seed := &types.SearchResult{
		ID:              "chunk-21",
		KnowledgeID:     "knowledge-1",
		KnowledgeBaseID: "kb-1",
		Content:         "raw retrieved content",
		Metadata:        map[string]string{"existing": "true"},
	}

	results := plugin.enrichExamQuestionContext(context.Background(), chatManage, []*types.SearchResult{seed})

	if len(results) == 0 {
		t.Fatal("expected enriched result")
	}
	got := results[0]
	if !strings.Contains(got.Content, "[Exam Structured Context]") {
		t.Fatalf("expected structured context, got:\n%s", got.Content)
	}
	if !strings.Contains(got.Content, "21. Which team will play the most games?") {
		t.Fatalf("missing structured question, got:\n%s", got.Content)
	}
	if got.ID != "chunk-21" {
		t.Fatalf("result ID = %q, want seed chunk", got.ID)
	}
	if got.Metadata["exam_context_mode"] != "structured" {
		t.Fatalf("mode = %q", got.Metadata["exam_context_mode"])
	}
	if got.Metadata["existing"] != "true" {
		t.Fatal("expected metadata to be preserved")
	}
	if strings.Join(got.SubChunkID, ",") != "chunk-a1" {
		t.Fatalf("sub chunks = %v", got.SubChunkID)
	}
	if strings.Join(repo.gotIDs, ",") != "chunk-21" {
		t.Fatalf("repo chunk IDs = %v", repo.gotIDs)
	}
}

func TestEnrichExamQuestionContextUsesFallbackBundleToResolveStructuredGroup(t *testing.T) {
	repo := &stubChatExamQuestionRepo{
		detail:  newChatStructuredReadingGroup(),
		matchID: "chunk-a1",
	}
	plugin := &PluginSearch{
		questionRepo: repo,
		chunkService: &stubChatChunkService{
			repo: &stubChatChunkRepo{chunks: []*types.Chunk{
				{
					ID:         "chunk-listening",
					ChunkIndex: 1,
					ChunkType:  types.ChunkTypeText,
					Content:    "First section listening questions.",
				},
				{
					ID:         "chunk-a1",
					ChunkIndex: 12,
					ChunkType:  types.ChunkTypeText,
					Content:    "**A**\nSoFi Stadium Events This Month\n21. Which team will play the most games?",
				},
				{
					ID:         "chunk-a2",
					ChunkIndex: 13,
					ChunkType:  types.ChunkTypeText,
					Content:    "A. Dallas Cowboys. B. Los Angeles Rams.\nC. Los Angeles Chargers. D. New Orleans Saints.",
				},
				{
					ID:         "chunk-b",
					ChunkIndex: 24,
					ChunkType:  types.ChunkTypeText,
					Content:    "**B**\nNext reading passage.",
				},
			}},
		},
	}
	chatManage := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			Query:    "first reading passage original questions and answer",
			TenantID: 10000,
			SearchTargets: types.SearchTargets{
				{KnowledgeBaseID: "kb-1", TenantID: 10000},
			},
		},
	}
	seed := &types.SearchResult{
		ID:              "chunk-listening",
		KnowledgeID:     "knowledge-1",
		KnowledgeBaseID: "kb-1",
		Content:         "First section listening questions.",
		Metadata:        map[string]string{},
	}

	results := plugin.enrichExamQuestionContext(context.Background(), chatManage, []*types.SearchResult{seed})

	if len(results) == 0 {
		t.Fatal("expected enriched result")
	}
	got := results[0]
	if !strings.Contains(got.Content, "[Exam Structured Context]") {
		t.Fatalf("expected structured context after fallback lookup, got:\n%s", got.Content)
	}
	if got.Metadata["exam_context_mode"] != "structured" {
		t.Fatalf("mode = %q", got.Metadata["exam_context_mode"])
	}
	if len(repo.calls) != 2 {
		t.Fatalf("repo calls = %v, want first seed lookup then fallback bundle lookup", repo.calls)
	}
	if strings.Join(repo.calls[0], ",") != "chunk-listening" {
		t.Fatalf("first repo call = %v", repo.calls[0])
	}
	if !containsExamContextString(repo.calls[1], "chunk-a1") {
		t.Fatalf("fallback repo call = %v, want chunk-a1", repo.calls[1])
	}
}

type stubChatChunkService struct {
	interfaces.ChunkService
	repo interfaces.ChunkRepository
}

func (s *stubChatChunkService) GetRepository() interfaces.ChunkRepository {
	return s.repo
}

type stubChatChunkRepo struct {
	interfaces.ChunkRepository
	chunks []*types.Chunk
}

func (r *stubChatChunkRepo) ListChunksByKnowledgeID(_ context.Context, _ uint64, _ string) ([]*types.Chunk, error) {
	return r.chunks, nil
}

func containsExamContextString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func newChatStructuredReadingGroup() *types.QuestionGroupDetail {
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

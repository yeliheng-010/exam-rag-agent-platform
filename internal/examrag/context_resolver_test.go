package examrag

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubContextResolverQuestionRepo struct {
	interfaces.ExamQuestionRepository
	detail               *types.QuestionGroupDetail
	evaluationCandidates []*types.QuestionGroupDetail
	gotTenantID          uint64
	gotChunkIDs          []string
	gotGroupID           string
	gotEvaluationKBIDs   []string
	gotEvaluationAnchors []string
	groupLookups         int
	evaluationLookups    int
}

func (r *stubContextResolverQuestionRepo) ListEvaluationQuestionGroupCandidates(
	_ context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
) ([]*types.QuestionGroupDetail, error) {
	r.gotTenantID = tenantID
	r.gotEvaluationKBIDs = append([]string{}, knowledgeBaseIDs...)
	r.evaluationLookups++
	return r.evaluationCandidates, nil
}

func (r *stubContextResolverQuestionRepo) ListEvaluationQuestionGroupCandidatesForSourcePhrases(
	_ context.Context,
	tenantID uint64,
	knowledgeBaseIDs []string,
	sourcePhrases []string,
) ([]*types.QuestionGroupDetail, error) {
	r.gotTenantID = tenantID
	r.gotEvaluationKBIDs = append([]string{}, knowledgeBaseIDs...)
	r.gotEvaluationAnchors = append([]string{}, sourcePhrases...)
	r.evaluationLookups++
	return r.evaluationCandidates, nil
}

func (r *stubContextResolverQuestionRepo) FindQuestionGroupDetailByChunkIDs(
	_ context.Context,
	tenantID uint64,
	chunkIDs []string,
) (*types.QuestionGroupDetail, error) {
	r.gotTenantID = tenantID
	r.gotChunkIDs = append([]string{}, chunkIDs...)
	return r.detail, nil
}

func (r *stubContextResolverQuestionRepo) GetQuestionGroupDetailByIDAndTenant(
	_ context.Context,
	tenantID uint64,
	groupID string,
) (*types.QuestionGroupDetail, error) {
	r.gotTenantID = tenantID
	r.gotGroupID = groupID
	r.groupLookups++
	return r.detail, nil
}

type stubContextResolverKBService struct {
	interfaces.KnowledgeBaseService
	results   []*types.SearchResult
	trace     *types.SearchTrace
	gotKBID   string
	gotParams types.SearchParams
	calls     int
}

func (s *stubContextResolverKBService) HybridSearchWithTrace(
	ctx context.Context,
	kbID string,
	params types.SearchParams,
) ([]*types.SearchResult, *types.SearchTrace, error) {
	results, err := s.HybridSearch(ctx, kbID, params)
	return results, s.trace, err
}

func (s *stubContextResolverKBService) HybridSearch(
	_ context.Context,
	kbID string,
	params types.SearchParams,
) ([]*types.SearchResult, error) {
	s.calls++
	s.gotKBID = kbID
	s.gotParams = params
	return s.results, nil
}

func TestResolverResolveQuerySearchesChunksAndBuildsStructuredContext(t *testing.T) {
	t.Parallel()

	repo := &stubContextResolverQuestionRepo{detail: newResolverStructuredReadingGroup()}
	kbService := &stubContextResolverKBService{
		trace: &types.SearchTrace{
			KnowledgeBaseID: "kb-1",
			FusionMethod:    types.SearchTraceFusionRRF,
		},
		results: []*types.SearchResult{
			{ID: "chunk-21", Content: "first retrieved content", SubChunkID: []string{"sub-21", "chunk-21"}},
			{ID: "chunk-22", Content: "second retrieved content", SubChunkID: []string{"sub-22"}},
		},
	}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo:         repo,
		KnowledgeBaseService: kbService,
		SearchTargets: types.SearchTargets{
			{
				Type:            types.SearchTargetTypeKnowledge,
				KnowledgeBaseID: "kb-1",
				TenantID:        10000,
				KnowledgeIDs:    []string{"knowledge-1"},
				TagIDs:          []string{"tag-a"},
			},
		},
	})

	result, err := resolver.Resolve(context.Background(), ExamQuestionContextResolveRequest{
		Query:            "first reading question 21 answer",
		KnowledgeBaseIDs: []string{"kb-1"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result == nil || result.Bundle == nil {
		t.Fatalf("expected context bundle, got %#v", result)
	}
	if kbService.calls != 1 {
		t.Fatalf("HybridSearch calls = %d", kbService.calls)
	}
	if kbService.gotKBID != "kb-1" {
		t.Fatalf("HybridSearch kb = %q", kbService.gotKBID)
	}
	if kbService.gotParams.QueryText != "first reading question 21 answer" {
		t.Fatalf("query text = %q", kbService.gotParams.QueryText)
	}
	if kbService.gotParams.MatchCount == 0 || kbService.gotParams.VectorThreshold == 0 || kbService.gotParams.KeywordThreshold == 0 {
		t.Fatalf("search params should use resolver defaults: %#v", kbService.gotParams)
	}
	if got := strings.Join(kbService.gotParams.KnowledgeIDs, ","); got != "knowledge-1" {
		t.Fatalf("knowledge filters = %q", got)
	}
	if got := strings.Join(kbService.gotParams.TagIDs, ","); got != "tag-a" {
		t.Fatalf("tag filters = %q", got)
	}
	if repo.gotTenantID != 10000 {
		t.Fatalf("repo tenant = %d", repo.gotTenantID)
	}
	if got := strings.Join(repo.gotChunkIDs, ","); got != "chunk-21,sub-21,chunk-22,sub-22" {
		t.Fatalf("repo chunk IDs = %q", got)
	}
	if got := strings.Join(result.RetrievedChunkIDs, ","); got != "chunk-21,sub-21,chunk-22,sub-22" {
		t.Fatalf("retrieved chunk IDs = %q", got)
	}
	if got := strings.Join(result.RetrievedContents, "|"); got != "first retrieved content|second retrieved content" {
		t.Fatalf("retrieved contents = %q", got)
	}
	if got := rankedItemPrimaryIDs(result.RetrievedItems); strings.Join(got, ",") != "chunk-21,chunk-22" {
		t.Fatalf("retrieved ranked items = %v", got)
	}
	if got := rankedItemPrimaryIDs(result.CandidateItems); strings.Join(got, ",") != "chunk-21,chunk-22" {
		t.Fatalf("candidate ranked items = %v", got)
	}
	if got := strings.Join(result.SourceChunkIDs, ","); got != "chunk-a1,chunk-21" {
		t.Fatalf("source chunk IDs = %q", got)
	}
	if result.ContextSource != types.ExamRAGContextSourceStructuredQuestionGroup {
		t.Fatalf("context source = %q", result.ContextSource)
	}
	if result.GroupID != "group-reading-a" {
		t.Fatalf("group ID = %q", result.GroupID)
	}
	if len(result.SearchTraces) != 1 || result.SearchTraces[0].FusionMethod != types.SearchTraceFusionRRF {
		t.Fatalf("search traces = %#v", result.SearchTraces)
	}
	if result.DurationMS < 0 {
		t.Fatalf("duration = %d", result.DurationMS)
	}
	if !strings.Contains(result.Bundle.Content, "B. Los Angeles Rams") {
		t.Fatalf("missing structured answer context:\n%s", result.Bundle.Content)
	}
}

func TestResolverResolveByGroupIDUsesTenantScopeWithoutSearch(t *testing.T) {
	t.Parallel()

	repo := &stubContextResolverQuestionRepo{detail: newResolverStructuredReadingGroup()}
	kbService := &stubContextResolverKBService{}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo:         repo,
		KnowledgeBaseService: kbService,
		SearchTargets: types.SearchTargets{
			{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 10000},
		},
	})

	result, err := resolver.Resolve(context.Background(), ExamQuestionContextResolveRequest{
		Query:            "group lookup",
		KnowledgeBaseIDs: []string{"kb-1"},
		GroupID:          "group-reading-a",
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result == nil || result.Bundle == nil {
		t.Fatalf("expected bundle, got %#v", result)
	}
	if kbService.calls != 0 {
		t.Fatalf("HybridSearch should not be called, got %d calls", kbService.calls)
	}
	if repo.gotGroupID != "group-reading-a" || repo.groupLookups != 1 {
		t.Fatalf("group lookup = %q calls=%d", repo.gotGroupID, repo.groupLookups)
	}
	if repo.gotTenantID != 10000 {
		t.Fatalf("repo tenant = %d", repo.gotTenantID)
	}
}

func newResolverStructuredReadingGroup() *types.QuestionGroupDetail {
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

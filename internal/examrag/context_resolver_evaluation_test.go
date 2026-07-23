package examrag

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestResolverEvalResolverUsesProductionRetrievalPath(t *testing.T) {
	t.Parallel()
	repo := &stubContextResolverQuestionRepo{detail: newResolverStructuredReadingGroup()}
	kbService := &stubContextResolverKBService{results: []*types.SearchResult{{
		ID: "chunk-21", Content: "stable phrase content", SubChunkID: []string{"sub-21"},
	}}}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo: repo, KnowledgeBaseService: kbService,
		SearchTargets: types.SearchTargets{{
			Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 10000,
		}},
	})
	resolution, err := resolver.EvalResolver(10000, []string{"kb-1"})(
		context.Background(), "first reading question 21 answer", nil,
	)
	if err != nil {
		t.Fatalf("EvalResolver returned error: %v", err)
	}
	if resolution.Bundle == nil || !strings.Contains(resolution.Bundle.Content, "SoFi Stadium is the go-to destination.") {
		t.Fatalf("expected structured bundle, got %#v", resolution.Bundle)
	}
	if got := strings.Join(resolution.RetrievedChunkIDs, ","); got != "chunk-21,sub-21" {
		t.Fatalf("retrieved chunk IDs = %q", got)
	}
	if got := strings.Join(resolution.RetrievedContents, "|"); got != "stable phrase content" {
		t.Fatalf("retrieved contents = %q", got)
	}
	if got := strings.Join(repo.gotChunkIDs, ","); got != "chunk-21,sub-21" {
		t.Fatalf("repo chunk IDs = %q", got)
	}
}

func TestResolverEvalResolverFallsBackToEvaluationAnchorMatch(t *testing.T) {
	t.Parallel()
	expected := newResolverStructuredReadingGroup()
	other := evaluationCandidate(
		"group-dictionary", "Dictionary Curiosity",
		"Kevin used a dictionary to look up an unfamiliar word.",
		"How did Kevin feel while looking up the word?",
	)
	repo := &stubContextResolverQuestionRepo{evaluationCandidates: []*types.QuestionGroupDetail{other, expected}}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo: repo,
		KnowledgeBaseService: &stubContextResolverKBService{results: []*types.SearchResult{{
			ID: "clone-chunk-21", Content: "SoFi Stadium is the go-to destination. Which team will play the most games?",
		}}},
		SearchTargets: types.SearchTargets{{
			Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "eval-kb", TenantID: 10000,
		}},
	})
	resolution, err := resolver.EvalResolver(10000, []string{"eval-kb"})(
		context.Background(), "Which team will play the most games?", []string{"SoFi Stadium is the go-to destination"},
	)
	if err != nil {
		t.Fatalf("EvalResolver returned error: %v", err)
	}
	if resolution == nil || resolution.Bundle == nil || resolution.GroupID != "group-reading-a" {
		t.Fatalf("resolution = %#v", resolution)
	}
	if resolution.ContextSource != types.ExamRAGContextSourceEvaluationAnchorMatch ||
		resolution.AssociationConfidence <= 0 {
		t.Fatalf("association = %q %.3f", resolution.ContextSource, resolution.AssociationConfidence)
	}
	if repo.evaluationLookups != 1 || strings.Join(repo.gotEvaluationKBIDs, ",") != "eval-kb" {
		t.Fatalf("evaluation lookup calls=%d kb=%v", repo.evaluationLookups, repo.gotEvaluationKBIDs)
	}
	if strings.Join(repo.gotEvaluationAnchors, ",") != "SoFi Stadium is the go-to destination" {
		t.Fatalf("evaluation anchors=%v", repo.gotEvaluationAnchors)
	}
}

func TestResolverResolveDoesNotUseEvaluationAnchorFallback(t *testing.T) {
	t.Parallel()
	repo := &stubContextResolverQuestionRepo{evaluationCandidates: []*types.QuestionGroupDetail{newResolverStructuredReadingGroup()}}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo: repo,
		KnowledgeBaseService: &stubContextResolverKBService{results: []*types.SearchResult{{
			ID: "clone-chunk-21", Content: "SoFi Stadium is the go-to destination.",
		}}},
		SearchTargets: types.SearchTargets{{
			Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "eval-kb", TenantID: 10000,
		}},
	})
	result, err := resolver.Resolve(context.Background(), ExamQuestionContextResolveRequest{
		Query: "Which team will play the most games?", KnowledgeBaseIDs: []string{"eval-kb"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result.Bundle != nil || result.ContextSource != types.ExamRAGContextSourceNone {
		t.Fatalf("unexpected fallback result = %#v", result)
	}
	if repo.evaluationLookups != 0 {
		t.Fatalf("ordinary Resolve used evaluation fallback %d times", repo.evaluationLookups)
	}
}

func TestResolverResolveReportsNoStructuredContext(t *testing.T) {
	t.Parallel()
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo:         &stubContextResolverQuestionRepo{},
		KnowledgeBaseService: &stubContextResolverKBService{results: []*types.SearchResult{{ID: "chunk-404"}}},
		SearchTargets: types.SearchTargets{{
			Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 10000,
		}},
	})
	result, err := resolver.Resolve(context.Background(), ExamQuestionContextResolveRequest{
		Query: "missing question group", KnowledgeBaseIDs: []string{"kb-1"},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if result.ContextSource != types.ExamRAGContextSourceNone || result.Bundle != nil || result.GroupID != "" {
		t.Fatalf("unexpected structured result = %#v", result)
	}
}

func TestResolverEvaluateRetrievalUsesSharedEvalSummary(t *testing.T) {
	t.Parallel()
	repo := &stubContextResolverQuestionRepo{detail: newResolverStructuredReadingGroup()}
	kbService := &stubContextResolverKBService{results: []*types.SearchResult{{
		ID: "chunk-21", SubChunkID: []string{"sub-21"},
	}}}
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		QuestionRepo: repo, KnowledgeBaseService: kbService,
		SearchTargets: types.SearchTargets{{
			Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb-1", TenantID: 10000,
		}},
	})
	summary := resolver.EvaluateRetrieval(context.Background(), ExamQuestionContextEvalRequest{
		TenantID: 10000, KnowledgeBaseIDs: []string{"kb-1"},
		Cases: []ExamContextRetrievalEvalCase{{
			Name: "q21_answer", Query: "first reading question 21 answer",
			RequiredPhrases: []string{"B. Los Angeles Rams"}, ExpectedChunkIDs: []string{"chunk-21"},
		}},
	})
	if summary.Total != 1 || summary.Passed != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.RetrievalHitRate != 1 || summary.AnswerHitRate != 1 {
		t.Fatalf("rates = %.2f %.2f", summary.RetrievalHitRate, summary.AnswerHitRate)
	}
}

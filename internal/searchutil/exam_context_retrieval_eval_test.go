package searchutil

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestEvaluateExamContextRetrievalReportsRetrievalAndAnswerHits(t *testing.T) {
	t.Parallel()

	detail := newGaokaoEnglishReadingEvalGroup()
	cases := []ExamContextRetrievalEvalCase{
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name:            "q21_answer",
				Query:           "first reading question 21 answer",
				RequiredPhrases: []string{"B. Los Angeles Rams", "The schedule names the Rams"},
			},
			ExpectedChunkIDs: []string{"chunk-21"},
		},
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name:            "q22_answer",
				Query:           "first reading question 22 answer",
				RequiredPhrases: []string{"A. Check event dates online", "chunk-22"},
			},
			ExpectedChunkIDs: []string{"chunk-22"},
		},
	}

	summary := EvaluateExamContextRetrieval(context.Background(), cases, func(_ context.Context, query string, _ []string) (*ExamContextResolution, error) {
		bundle := BuildStructuredExamQuestionContextBundle(query, detail)
		if strings.Contains(query, "22") {
			return &ExamContextResolution{
				Bundle:            bundle,
				RetrievedChunkIDs: []string{"chunk-22", "chunk-a1"},
				ContextSource:     types.ExamRAGContextSourceStructuredQuestionGroup,
				DurationMS:        20,
			}, nil
		}
		return &ExamContextResolution{
			Bundle:            bundle,
			RetrievedChunkIDs: []string{"chunk-21", "chunk-a1"},
			ContextSource:     types.ExamRAGContextSourceStructuredQuestionGroup,
			DurationMS:        10,
		}, nil
	})

	if summary.Total != 2 {
		t.Fatalf("total = %d", summary.Total)
	}
	if summary.Passed != 2 || summary.RetrievalPassed != 2 || summary.AnswerPassed != 2 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.HitRate != 1 || summary.RetrievalHitRate != 1 || summary.AnswerHitRate != 1 {
		t.Fatalf("rates = %.2f %.2f %.2f", summary.HitRate, summary.RetrievalHitRate, summary.AnswerHitRate)
	}
	if got := strings.Join(summary.Results[0].RetrievedChunkIDs, ","); got != "chunk-21,chunk-a1" {
		t.Fatalf("retrieved chunks = %q", got)
	}
	if summary.RecallAtK != 1 || summary.MeanReciprocalRank != 1 {
		t.Fatalf("rank metrics = %.2f %.2f", summary.RecallAtK, summary.MeanReciprocalRank)
	}
	if summary.RankedCaseCount != 2 || summary.StructuredResolutionRate != 1 {
		t.Fatalf("structured metrics = %#v", summary)
	}
	if summary.AverageDurationMS != 15 {
		t.Fatalf("average duration = %.2f", summary.AverageDurationMS)
	}
}

func TestEvaluateExamContextRetrievalReportsMissesSeparately(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name:            "miss",
				Query:           "first reading question 21 answer",
				RequiredPhrases: []string{"B. Los Angeles Rams", "not in context"},
			},
			ExpectedChunkIDs: []string{"chunk-21", "chunk-999"},
		},
	}, func(ctx context.Context, query string, _ []string) (*ExamContextResolution, error) {
		return &ExamContextResolution{
			Bundle:            BuildStructuredExamQuestionContextBundle(query, newGaokaoEnglishReadingEvalGroup()),
			RetrievedChunkIDs: []string{"chunk-21"},
			ContextSource:     types.ExamRAGContextSourceStructuredQuestionGroup,
		}, nil
	})

	if summary.Total != 1 || summary.Passed != 0 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.RetrievalPassed != 0 || summary.AnswerPassed != 0 {
		t.Fatalf("summary = %#v", summary)
	}
	result := summary.Results[0]
	if result.RetrievalScore != 0.5 {
		t.Fatalf("retrieval score = %.2f", result.RetrievalScore)
	}
	if result.AnswerScore != 0.5 {
		t.Fatalf("answer score = %.2f", result.AnswerScore)
	}
	if got := strings.Join(result.MissingChunkIDs, ","); got != "chunk-999" {
		t.Fatalf("missing chunks = %q", got)
	}
	if got := strings.Join(result.MissingPhrases, ","); got != "not in context" {
		t.Fatalf("missing phrases = %q", got)
	}
}

func TestEvaluateExamContextRetrievalExcludesUnrankedCasesFromRankMetrics(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name:            "answer_only",
				Query:           "answer only",
				RequiredPhrases: []string{"Answer"},
			},
		},
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name:            "ranked",
				Query:           "ranked",
				RequiredPhrases: []string{"Answer"},
			},
			ExpectedChunkIDs: []string{"chunk-2"},
		},
	}, func(_ context.Context, query string, _ []string) (*ExamContextResolution, error) {
		return &ExamContextResolution{
			Bundle: &ExamQuestionContextBundle{
				Content: "Answer",
			},
			RetrievedChunkIDs: []string{"chunk-1", "chunk-2"},
			ContextSource:     types.ExamRAGContextSourceStructuredQuestionGroup,
		}, nil
	})

	if summary.RankedCaseCount != 1 {
		t.Fatalf("ranked cases = %d", summary.RankedCaseCount)
	}
	if summary.RetrievalPassed != 1 || summary.RetrievalHitRate != 1 {
		t.Fatalf("retrieval summary = %#v", summary)
	}
	if summary.Results[0].RetrievalPassed || !summary.Results[0].Passed {
		t.Fatalf("answer-only case = %#v", summary.Results[0])
	}
	if summary.RecallAtK != 1 || summary.MeanReciprocalRank != 0.5 {
		t.Fatalf("rank metrics = %.2f %.2f", summary.RecallAtK, summary.MeanReciprocalRank)
	}
}

func TestEvaluateExamContextRetrievalCountsDynamicAssociationAsStructured(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{{
		ExamContextEvalCase: ExamContextEvalCase{
			Name: "dynamic", Query: "question", RequiredPhrases: []string{"Answer: B"},
		},
	}}, func(context.Context, string, []string) (*ExamContextResolution, error) {
		return &ExamContextResolution{
			Bundle:                &ExamQuestionContextBundle{Content: "Answer: B"},
			ContextSource:         types.ExamRAGContextSourceEvaluationAnchorMatch,
			AssociationConfidence: 0.875,
		}, nil
	})

	if summary.StructuredResolved != 1 || summary.StructuredResolutionRate != 1 {
		t.Fatalf("structured summary = %#v", summary)
	}
	if summary.Results[0].AssociationConfidence != 0.875 {
		t.Fatalf("association confidence = %.3f", summary.Results[0].AssociationConfidence)
	}
}

func TestEvaluateExamContextRetrievalUsesContentPhraseGold(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{
		{
			ExamContextEvalCase: ExamContextEvalCase{
				Name: "phrase_gold", Query: "find stable evidence",
				RequiredPhrases: []string{"Answer"},
			},
			RequiredRetrievalPhrases: []string{"first target", "second target"},
		},
	}, func(_ context.Context, _ string, anchors []string) (*ExamContextResolution, error) {
		if got := strings.Join(anchors, ","); got != "first target,second target" {
			t.Fatalf("resolver anchors = %q", got)
		}
		return &ExamContextResolution{
			Bundle:            &ExamQuestionContextBundle{Content: "Answer"},
			RetrievedChunkIDs: []string{"chunk-1", "chunk-2", "chunk-3"},
			RetrievedContents: []string{"irrelevant", "contains first target", "contains second target"},
		}, nil
	})

	if summary.RankedCaseCount != 1 || summary.RetrievalPassed != 1 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.RecallAtK != 1 || summary.MeanReciprocalRank != 0.5 {
		t.Fatalf("rank metrics = %.2f %.2f", summary.RecallAtK, summary.MeanReciprocalRank)
	}
	result := summary.Results[0]
	if got := strings.Join(result.MatchedRetrievalPhrases, ","); got != "first target,second target" {
		t.Fatalf("matched retrieval phrases = %q", got)
	}
	if len(result.MissingRetrievalPhrases) != 0 || result.FirstRelevantRank != 2 {
		t.Fatalf("phrase result = %#v", result)
	}
}

func TestEvaluateExamContextRetrievalSeparatesGlobalTopKFromCandidateRecall(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{{
		ExamContextEvalCase: ExamContextEvalCase{Name: "outside_top_k", Query: "find gold"},
		ExpectedChunkIDs:    []string{"gold"},
	}}, func(context.Context, string, []string) (*ExamContextResolution, error) {
		return &ExamContextResolution{
			RetrievedItems: []ExamContextRankedItem{
				{Rank: 1, ChunkIDs: []string{"noise-1"}, Contents: []string{"noise"}},
				{Rank: 2, ChunkIDs: []string{"noise-2"}, Contents: []string{"noise"}},
			},
			CandidateItems: []ExamContextRankedItem{
				{Rank: 1, ChunkIDs: []string{"noise-1"}, Contents: []string{"noise"}},
				{Rank: 2, ChunkIDs: []string{"noise-2"}, Contents: []string{"noise"}},
				{Rank: 3, ChunkIDs: []string{"gold"}, Contents: []string{"target"}},
			},
		}, nil
	})

	require.Equal(t, 0.0, summary.RecallAtK)
	require.Equal(t, 0.0, summary.RetrievalHitRate)
	require.Equal(t, 1.0, summary.CandidateRecall)
	require.Equal(t, 1.0, summary.CandidateHitRate)
	result := summary.Results[0]
	require.False(t, result.RetrievalPassed)
	require.True(t, result.CandidateRetrievalPassed)
	require.Equal(t, 0.0, result.RetrievalScore)
	require.Equal(t, 1.0, result.CandidateRetrievalScore)
}

func TestEvaluateExamContextRetrievalRanksParentContentWithItsMainChunk(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{{
		ExamContextEvalCase:      ExamContextEvalCase{Name: "parent_phrase", Query: "find table"},
		RequiredRetrievalPhrases: []string{"Upcoming Football Events"},
	}}, func(context.Context, string, []string) (*ExamContextResolution, error) {
		items := []ExamContextRankedItem{
			{
				Rank: 1, ChunkIDs: []string{"child-1", "parent-1"},
				Contents: []string{"question text", "Upcoming Football Events table"},
			},
			{Rank: 2, ChunkIDs: []string{"child-2"}, Contents: []string{"other"}},
		}
		return &ExamContextResolution{RetrievedItems: items, CandidateItems: items}, nil
	})

	require.Equal(t, 1, summary.Results[0].FirstRelevantRank)
	require.Equal(t, 1.0, summary.MeanReciprocalRank)
	require.True(t, summary.Results[0].RetrievalPassed)
}

func TestEvaluateExamContextRetrievalMatchesMixedGoldAtMainResultRanks(t *testing.T) {
	t.Parallel()

	summary := EvaluateExamContextRetrieval(context.Background(), []ExamContextRetrievalEvalCase{{
		ExamContextEvalCase:      ExamContextEvalCase{Name: "mixed_gold", Query: "find both"},
		ExpectedChunkIDs:         []string{"chunk-2"},
		RequiredRetrievalPhrases: []string{"first phrase"},
	}}, func(context.Context, string, []string) (*ExamContextResolution, error) {
		items := []ExamContextRankedItem{
			{Rank: 1, ChunkIDs: []string{"chunk-1"}, Contents: []string{"first phrase"}},
			{Rank: 2, ChunkIDs: []string{"chunk-2", "sub-2"}, Contents: []string{"second"}},
		}
		return &ExamContextResolution{RetrievedItems: items, CandidateItems: items}, nil
	})

	result := summary.Results[0]
	require.True(t, result.RetrievalPassed)
	require.Equal(t, 1.0, result.RetrievalScore)
	require.Equal(t, 1, result.FirstRelevantRank)
	require.Equal(t, 1.0, result.ReciprocalRank)
}

package searchutil

import (
	"context"
	"strings"
	"testing"
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

	summary := EvaluateExamContextRetrieval(context.Background(), cases, func(_ context.Context, query string) (*ExamQuestionContextBundle, []string, error) {
		bundle := BuildStructuredExamQuestionContextBundle(query, detail)
		if strings.Contains(query, "22") {
			return bundle, []string{"chunk-22", "chunk-a1"}, nil
		}
		return bundle, []string{"chunk-21", "chunk-a1"}, nil
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
	}, func(ctx context.Context, query string) (*ExamQuestionContextBundle, []string, error) {
		return BuildStructuredExamQuestionContextBundle(query, newGaokaoEnglishReadingEvalGroup()), []string{"chunk-21"}, nil
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

package searchutil

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// ExamContextEvalCase describes one deterministic answerability check for an
// exam question context bundle.
type ExamContextEvalCase struct {
	Name            string
	Query           string
	RequiredPhrases []string
}

type ExamContextEvalResult struct {
	Name           string
	Query          string
	Passed         bool
	Score          float64
	MatchedPhrases []string
	MissingPhrases []string
	ContextLabel   string
	SourceChunkIDs []string
}

type ExamContextEvalSummary struct {
	Total   int
	Passed  int
	HitRate float64
	Results []ExamContextEvalResult
}

type ExamContextRetrievalEvalCase struct {
	ExamContextEvalCase
	ExpectedChunkIDs         []string
	RequiredRetrievalPhrases []string
}

type ExamContextResolution struct {
	Bundle                *ExamQuestionContextBundle
	RetrievedChunkIDs     []string
	RetrievedContents     []string
	RetrievedItems        []ExamContextRankedItem
	CandidateChunkIDs     []string
	CandidateItems        []ExamContextRankedItem
	SourceChunkIDs        []string
	ContextSource         types.ExamRAGContextSource
	GroupID               string
	AssociationConfidence float64
	SearchTraces          []*types.SearchTrace
	DurationMS            int64
}

type ExamContextBundleResolver func(
	ctx context.Context,
	query string,
	evaluationAnchors []string,
) (*ExamContextResolution, error)

type ExamContextRetrievalEvalResult struct {
	Name                     string
	Query                    string
	Passed                   bool
	RetrievalPassed          bool
	AnswerPassed             bool
	RetrievalScore           float64
	CandidateRetrievalScore  float64
	AnswerScore              float64
	CandidateRetrievalPassed bool
	MatchedChunkIDs          []string
	MissingChunkIDs          []string
	RetrievedChunkIDs        []string
	RetrievedItems           []ExamContextRankedItem
	MatchedRetrievalPhrases  []string
	MissingRetrievalPhrases  []string
	MatchedPhrases           []string
	MissingPhrases           []string
	ContextLabel             string
	SourceChunkIDs           []string
	CandidateChunkIDs        []string
	CandidateItems           []ExamContextRankedItem
	ContextSource            types.ExamRAGContextSource
	GroupID                  string
	AssociationConfidence    float64
	SearchTraces             []*types.SearchTrace
	DurationMS               int64
	ReciprocalRank           float64
	FirstRelevantRank        int
	Error                    string
}

type ExamContextRetrievalEvalSummary struct {
	Total                    int
	Passed                   int
	RetrievalPassed          int
	AnswerPassed             int
	HitRate                  float64
	RetrievalHitRate         float64
	AnswerHitRate            float64
	RecallAtK                float64
	CandidateRecall          float64
	CandidateHitRate         float64
	MeanReciprocalRank       float64
	RankedCaseCount          int
	StructuredResolutionRate float64
	StructuredResolved       int
	AverageDurationMS        float64
	FailedCaseCount          int
	Results                  []ExamContextRetrievalEvalResult
}

func EvaluateStructuredExamQuestionContext(
	detail *types.QuestionGroupDetail,
	cases []ExamContextEvalCase,
) ExamContextEvalSummary {
	results := make([]ExamContextEvalResult, 0, len(cases))
	passed := 0
	for _, evalCase := range cases {
		result := evaluateStructuredExamQuestionContextCase(detail, evalCase)
		if result.Passed {
			passed++
		}
		results = append(results, result)
	}
	return ExamContextEvalSummary{
		Total:   len(cases),
		Passed:  passed,
		HitRate: examEvalHitRate(passed, len(cases)),
		Results: results,
	}
}

func evaluateStructuredExamQuestionContextCase(
	detail *types.QuestionGroupDetail,
	evalCase ExamContextEvalCase,
) ExamContextEvalResult {
	result := ExamContextEvalResult{Name: evalCase.Name, Query: evalCase.Query}
	bundle := BuildStructuredExamQuestionContextBundle(evalCase.Query, detail)
	if bundle == nil {
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return result
	}
	result.ContextLabel = bundle.Label
	result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	result.MatchedPhrases, result.MissingPhrases = matchExamEvalPhrases(
		bundle.Content,
		evalCase.RequiredPhrases,
	)
	result.Score = examEvalScore(len(result.MatchedPhrases), len(evalCase.RequiredPhrases))
	result.Passed = len(result.MissingPhrases) == 0
	return result
}

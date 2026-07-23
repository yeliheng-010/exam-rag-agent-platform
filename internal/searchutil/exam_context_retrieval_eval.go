package searchutil

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type retrievalEvalCounters struct {
	passed, retrievalPassed, answerPassed int
	candidatePassed, rankedCaseCount      int
	structuredResolved, failedCaseCount   int
	recallSum, candidateRecallSum         float64
	reciprocalRankSum                     float64
	totalDurationMS                       int64
}

func EvaluateExamContextRetrieval(
	ctx context.Context,
	cases []ExamContextRetrievalEvalCase,
	resolver ExamContextBundleResolver,
) ExamContextRetrievalEvalSummary {
	results := make([]ExamContextRetrievalEvalResult, 0, len(cases))
	counters := retrievalEvalCounters{}
	for _, evalCase := range cases {
		result := evaluateExamContextRetrievalCase(ctx, evalCase, resolver)
		counters.add(evalCase, result)
		results = append(results, result)
	}
	return counters.summary(len(cases), results)
}

func (c *retrievalEvalCounters) add(
	evalCase ExamContextRetrievalEvalCase,
	result ExamContextRetrievalEvalResult,
) {
	if result.Passed {
		c.passed++
	}
	if result.AnswerPassed {
		c.answerPassed++
	}
	if hasRetrievalGold(evalCase) {
		c.addRankedResult(result)
	}
	if isStructuredExamContextSource(result.ContextSource) {
		c.structuredResolved++
	}
	if result.Error != "" {
		c.failedCaseCount++
	}
	c.totalDurationMS += result.DurationMS
}

func (c *retrievalEvalCounters) addRankedResult(result ExamContextRetrievalEvalResult) {
	c.rankedCaseCount++
	if result.RetrievalPassed {
		c.retrievalPassed++
	}
	if result.CandidateRetrievalPassed {
		c.candidatePassed++
	}
	c.recallSum += result.RetrievalScore
	c.candidateRecallSum += result.CandidateRetrievalScore
	c.reciprocalRankSum += result.ReciprocalRank
}

func (c retrievalEvalCounters) summary(
	total int,
	results []ExamContextRetrievalEvalResult,
) ExamContextRetrievalEvalSummary {
	return ExamContextRetrievalEvalSummary{
		Total: total, Passed: c.passed, RetrievalPassed: c.retrievalPassed,
		AnswerPassed: c.answerPassed, HitRate: examEvalHitRate(c.passed, total),
		RetrievalHitRate:         examEvalHitRate(c.retrievalPassed, c.rankedCaseCount),
		AnswerHitRate:            examEvalHitRate(c.answerPassed, total),
		RecallAtK:                examEvalAverage(c.recallSum, c.rankedCaseCount),
		CandidateRecall:          examEvalAverage(c.candidateRecallSum, c.rankedCaseCount),
		CandidateHitRate:         examEvalHitRate(c.candidatePassed, c.rankedCaseCount),
		MeanReciprocalRank:       examEvalAverage(c.reciprocalRankSum, c.rankedCaseCount),
		RankedCaseCount:          c.rankedCaseCount,
		StructuredResolutionRate: examEvalHitRate(c.structuredResolved, total),
		StructuredResolved:       c.structuredResolved,
		AverageDurationMS:        examEvalAverage(float64(c.totalDurationMS), total),
		FailedCaseCount:          c.failedCaseCount, Results: results,
	}
}

func evaluateExamContextRetrievalCase(
	ctx context.Context,
	evalCase ExamContextRetrievalEvalCase,
	resolver ExamContextBundleResolver,
) ExamContextRetrievalEvalResult {
	result := ExamContextRetrievalEvalResult{Name: evalCase.Name, Query: evalCase.Query}
	if resolver == nil {
		result.Error = "exam context resolver is nil"
		setMissingEvalGold(&result, evalCase)
		return result
	}
	resolution, err := resolver(ctx, evalCase.Query, evalCase.RequiredRetrievalPhrases)
	if err != nil {
		result.Error = err.Error()
		setMissingEvalGold(&result, evalCase)
		return result
	}
	if resolution == nil {
		setMissingEvalGold(&result, evalCase)
		return result
	}
	copyExamContextResolution(&result, resolution)
	applyRetrievalGold(&result, evalCase, resolution)
	applyAnswerGold(&result, evalCase, resolution)
	return result
}

func setMissingEvalGold(result *ExamContextRetrievalEvalResult, evalCase ExamContextRetrievalEvalCase) {
	result.MissingChunkIDs = cleanEvalPhrases(evalCase.ExpectedChunkIDs)
	result.MissingRetrievalPhrases = cleanEvalPhrases(evalCase.RequiredRetrievalPhrases)
	result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
}

func copyExamContextResolution(result *ExamContextRetrievalEvalResult, resolution *ExamContextResolution) {
	result.RetrievedChunkIDs = cleanEvalPhrases(resolution.RetrievedChunkIDs)
	result.CandidateChunkIDs = cleanEvalPhrases(resolution.CandidateChunkIDs)
	result.RetrievedItems = append([]ExamContextRankedItem{}, resolution.RetrievedItems...)
	result.CandidateItems = append([]ExamContextRankedItem{}, resolution.CandidateItems...)
	if len(result.RetrievedChunkIDs) == 0 {
		result.RetrievedChunkIDs = flattenRankedItemIDs(resolution.RetrievedItems)
	}
	if len(result.CandidateChunkIDs) == 0 {
		result.CandidateChunkIDs = flattenRankedItemIDs(resolution.CandidateItems)
	}
	result.ContextSource = resolution.ContextSource
	result.GroupID = resolution.GroupID
	result.AssociationConfidence = resolution.AssociationConfidence
	result.SearchTraces = append([]*types.SearchTrace{}, resolution.SearchTraces...)
	result.DurationMS = resolution.DurationMS
}

func applyAnswerGold(
	result *ExamContextRetrievalEvalResult,
	evalCase ExamContextRetrievalEvalCase,
	resolution *ExamContextResolution,
) {
	bundle := resolution.Bundle
	if bundle == nil {
		result.MissingPhrases = cleanEvalPhrases(evalCase.RequiredPhrases)
		return
	}
	result.ContextLabel = bundle.Label
	result.SourceChunkIDs = cleanEvalPhrases(resolution.SourceChunkIDs)
	if len(result.SourceChunkIDs) == 0 {
		result.SourceChunkIDs = append([]string{}, bundle.SourceChunkIDs...)
	}
	result.MatchedPhrases, result.MissingPhrases = matchExamEvalPhrases(bundle.Content, evalCase.RequiredPhrases)
	result.AnswerScore = examEvalScore(len(result.MatchedPhrases), len(cleanEvalPhrases(evalCase.RequiredPhrases)))
	result.AnswerPassed = len(result.MissingPhrases) == 0
	result.Passed = result.AnswerPassed && (!hasRetrievalGold(evalCase) || result.RetrievalPassed)
}

func isStructuredExamContextSource(source types.ExamRAGContextSource) bool {
	return source == types.ExamRAGContextSourceStructuredQuestionGroup ||
		source == types.ExamRAGContextSourceEvaluationAnchorMatch
}

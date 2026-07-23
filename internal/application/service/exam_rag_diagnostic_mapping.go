package service

import (
	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

func toExamRAGDiagnosticSummary(summary examrag.ExamContextRetrievalEvalSummary) types.ExamRAGDiagnosticSummary {
	results := make([]types.ExamRAGDiagnosticResultItem, 0, len(summary.Results))
	for _, item := range summary.Results {
		results = append(results, toExamRAGDiagnosticResultItem(item))
	}
	return types.ExamRAGDiagnosticSummary{
		Total: summary.Total, Passed: summary.Passed, RetrievalPassed: summary.RetrievalPassed,
		AnswerPassed: summary.AnswerPassed, HitRate: summary.HitRate,
		RetrievalHitRate: summary.RetrievalHitRate, AnswerHitRate: summary.AnswerHitRate,
		RecallAtK: summary.RecallAtK, CandidateRecall: diagnosticFloat64Ptr(summary.CandidateRecall),
		CandidateHitRate:   diagnosticFloat64Ptr(summary.CandidateHitRate),
		MeanReciprocalRank: summary.MeanReciprocalRank, RankedCaseCount: summary.RankedCaseCount,
		StructuredResolutionRate: summary.StructuredResolutionRate,
		StructuredResolved:       summary.StructuredResolved, AverageDurationMS: summary.AverageDurationMS,
		FailedCaseCount: summary.FailedCaseCount, Results: results,
	}
}

func toExamRAGDiagnosticResultItem(item searchutil.ExamContextRetrievalEvalResult) types.ExamRAGDiagnosticResultItem {
	return types.ExamRAGDiagnosticResultItem{
		Name: item.Name, Query: item.Query, Passed: item.Passed,
		RetrievalPassed: item.RetrievalPassed, AnswerPassed: item.AnswerPassed,
		RetrievalScore: item.RetrievalScore, AnswerScore: item.AnswerScore,
		CandidateRetrievalScore:  diagnosticFloat64Ptr(item.CandidateRetrievalScore),
		CandidateRetrievalPassed: diagnosticBoolPtr(item.CandidateRetrievalPassed),
		MatchedChunkIDs:          copyDiagnosticStrings(item.MatchedChunkIDs),
		MissingChunkIDs:          copyDiagnosticStrings(item.MissingChunkIDs),
		RetrievedChunkIDs:        copyDiagnosticStrings(item.RetrievedChunkIDs),
		RetrievedItems:           toDiagnosticRankedItems(item.RetrievedItems),
		MatchedRetrievalPhrases:  copyDiagnosticStrings(item.MatchedRetrievalPhrases),
		MissingRetrievalPhrases:  copyDiagnosticStrings(item.MissingRetrievalPhrases),
		MatchedPhrases:           copyDiagnosticStrings(item.MatchedPhrases),
		MissingPhrases:           copyDiagnosticStrings(item.MissingPhrases), ContextLabel: item.ContextLabel,
		SourceChunkIDs:    copyDiagnosticStrings(item.SourceChunkIDs),
		CandidateChunkIDs: copyDiagnosticStrings(item.CandidateChunkIDs),
		CandidateItems:    toDiagnosticRankedItems(item.CandidateItems), ContextSource: item.ContextSource,
		GroupID: item.GroupID, AssociationConfidence: item.AssociationConfidence,
		SearchTraces: append([]*types.SearchTrace{}, item.SearchTraces...), DurationMS: item.DurationMS,
		ReciprocalRank: item.ReciprocalRank, FirstRelevantRank: item.FirstRelevantRank, Error: item.Error,
	}
}

func toDiagnosticRankedItems(items []searchutil.ExamContextRankedItem) []types.ExamRAGDiagnosticRankedItem {
	out := make([]types.ExamRAGDiagnosticRankedItem, 0, len(items))
	for _, item := range items {
		out = append(out, types.ExamRAGDiagnosticRankedItem{
			Rank: item.Rank, KnowledgeBaseID: item.KnowledgeBaseID, LocalRank: item.LocalRank,
			ChunkIDs: copyDiagnosticStrings(item.ChunkIDs), Contents: copyDiagnosticStrings(item.Contents),
		})
	}
	return out
}

func diagnosticFloat64Ptr(value float64) *float64 { return &value }

func diagnosticBoolPtr(value bool) *bool { return &value }

func copyDiagnosticStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string{}, values...)
}

package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/searchutil"
)

func TestToExamRAGDiagnosticSummaryPersistsStrictAndCandidateRankings(t *testing.T) {
	summary := toExamRAGDiagnosticSummary(examrag.ExamContextRetrievalEvalSummary{
		RecallAtK: 0.25, CandidateRecall: 1, CandidateHitRate: 0.8,
		Results: []searchutil.ExamContextRetrievalEvalResult{{
			RetrievalScore: 0.25, CandidateRetrievalScore: 1,
			CandidateRetrievalPassed: true,
			RetrievedItems: []searchutil.ExamContextRankedItem{{
				Rank: 1, KnowledgeBaseID: "kb-1", LocalRank: 1,
				ChunkIDs: []string{"chunk-1", "parent-1"}, Contents: []string{"child", "parent"},
			}},
			CandidateItems: []searchutil.ExamContextRankedItem{{
				Rank: 2, KnowledgeBaseID: "kb-2", LocalRank: 1,
				ChunkIDs: []string{"gold"}, Contents: []string{"target"},
			}},
		}},
	})

	if summary.CandidateRecall == nil || *summary.CandidateRecall != 1 {
		t.Fatalf("candidate recall = %#v", summary.CandidateRecall)
	}
	if summary.CandidateHitRate == nil || *summary.CandidateHitRate != 0.8 {
		t.Fatalf("candidate hit rate = %#v", summary.CandidateHitRate)
	}
	item := summary.Results[0]
	if item.CandidateRetrievalScore == nil || *item.CandidateRetrievalScore != 1 {
		t.Fatalf("candidate score = %#v", item.CandidateRetrievalScore)
	}
	if item.CandidateRetrievalPassed == nil || !*item.CandidateRetrievalPassed {
		t.Fatalf("candidate passed = %#v", item.CandidateRetrievalPassed)
	}
	if len(item.RetrievedItems) != 1 || item.RetrievedItems[0].ChunkIDs[1] != "parent-1" {
		t.Fatalf("retrieved items = %#v", item.RetrievedItems)
	}
	if len(item.CandidateItems) != 1 || item.CandidateItems[0].ChunkIDs[0] != "gold" {
		t.Fatalf("candidate items = %#v", item.CandidateItems)
	}
}

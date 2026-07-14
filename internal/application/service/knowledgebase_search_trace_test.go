package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestSearchTraceRecorderCapturesRetrievalStages(t *testing.T) {
	t.Parallel()

	ctx, recorder := withSearchTrace(context.Background(), types.SearchTrace{
		Query:           "SoFi Stadium",
		KnowledgeBaseID: "kb-1",
		Parameters: types.SearchTraceParameters{
			MatchCount:       2,
			VectorThreshold:  0.5,
			KeywordThreshold: 0.3,
		},
	})
	vector := []*types.IndexWithScore{
		{ChunkID: "chunk-vector", Score: 0.91},
		{ChunkID: "chunk-both", Score: 0.82},
	}
	keyword := []*types.IndexWithScore{
		{ChunkID: "chunk-both", Score: 14.2},
		{ChunkID: "chunk-keyword", Score: 8.1},
	}

	recordSearchTraceCandidates(ctx, vector, keyword)
	fused := fuseOrDeduplicate(ctx, vector, keyword, nil)
	recordFinalSearchTraceResults(ctx, []*types.SearchResult{
		{ID: fused[0].ChunkID, Score: fused[0].Score},
		{ID: fused[1].ChunkID, Score: fused[1].Score},
	})
	trace := recorder.finish()

	if trace.FusionMethod != types.SearchTraceFusionRRF {
		t.Fatalf("fusion method = %q", trace.FusionMethod)
	}
	if got := trace.VectorCandidates[0]; got.ChunkID != "chunk-vector" || got.Rank != 1 {
		t.Fatalf("vector candidate = %#v", got)
	}
	if got := trace.KeywordCandidates[0]; got.ChunkID != "chunk-both" || got.Rank != 1 {
		t.Fatalf("keyword candidate = %#v", got)
	}
	fusedBoth := findSearchTraceCandidate(t, trace.FusionCandidates, "chunk-both")
	if fusedBoth.VectorRank != 2 || fusedBoth.KeywordRank != 1 || fusedBoth.Rank != 1 {
		t.Fatalf("fused candidate = %#v", fusedBoth)
	}
	if len(trace.FinalChunks) != 2 || trace.FinalChunks[0].ChunkID != fused[0].ChunkID {
		t.Fatalf("final chunks = %#v", trace.FinalChunks)
	}
	if trace.DurationMS < 0 {
		t.Fatalf("duration = %d", trace.DurationMS)
	}
}

func TestSearchTraceRecordersAreNoopWithoutTraceContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	recordSearchTraceCandidates(ctx, []*types.IndexWithScore{{ChunkID: "chunk-1"}}, nil)
	recordFinalSearchTraceResults(ctx, []*types.SearchResult{{ID: "chunk-1"}})
}

func findSearchTraceCandidate(
	t *testing.T,
	candidates []types.SearchTraceCandidate,
	chunkID string,
) types.SearchTraceCandidate {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.ChunkID == chunkID {
			return candidate
		}
	}
	t.Fatalf("candidate %s not found in %#v", chunkID, candidates)
	return types.SearchTraceCandidate{}
}

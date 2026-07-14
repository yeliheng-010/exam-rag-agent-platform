package service

import (
	"context"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type searchTraceContextKey struct{}

type searchTraceRecorder struct {
	mu      sync.Mutex
	trace   types.SearchTrace
	started time.Time
}

func withSearchTrace(ctx context.Context, initial types.SearchTrace) (context.Context, *searchTraceRecorder) {
	recorder := &searchTraceRecorder{trace: initial, started: time.Now()}
	return context.WithValue(ctx, searchTraceContextKey{}, recorder), recorder
}

func searchTraceRecorderFromContext(ctx context.Context) *searchTraceRecorder {
	if ctx == nil {
		return nil
	}
	recorder, _ := ctx.Value(searchTraceContextKey{}).(*searchTraceRecorder)
	return recorder
}

func (r *searchTraceRecorder) finish() *types.SearchTrace {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.trace.DurationMS = time.Since(r.started).Milliseconds()
	cloned := r.trace
	cloned.KnowledgeBaseIDs = append([]string{}, r.trace.KnowledgeBaseIDs...)
	cloned.VectorCandidates = append([]types.SearchTraceCandidate{}, r.trace.VectorCandidates...)
	cloned.KeywordCandidates = append([]types.SearchTraceCandidate{}, r.trace.KeywordCandidates...)
	cloned.FusionCandidates = append([]types.SearchTraceCandidate{}, r.trace.FusionCandidates...)
	cloned.FinalChunks = append([]types.SearchTraceCandidate{}, r.trace.FinalChunks...)
	return &cloned
}

func recordSearchTraceConfiguration(
	ctx context.Context,
	kb *types.KnowledgeBase,
	kbIDs []string,
	params types.SearchParams,
) {
	recorder := searchTraceRecorderFromContext(ctx)
	if recorder == nil {
		return
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.trace.Query = params.QueryText
	recorder.trace.KnowledgeBaseIDs = append([]string{}, kbIDs...)
	recorder.trace.Parameters.MatchCount = params.MatchCount
	recorder.trace.Parameters.VectorThreshold = params.VectorThreshold
	recorder.trace.Parameters.KeywordThreshold = params.KeywordThreshold
	recorder.trace.Parameters.VectorEnabled = !params.DisableVectorMatch
	recorder.trace.Parameters.KeywordEnabled = !params.DisableKeywordsMatch
	recorder.trace.EmbeddingDimensions = len(params.QueryEmbedding)
	if kb != nil {
		recorder.trace.KnowledgeBaseID = kb.ID
		recorder.trace.EmbeddingModelID = kb.EmbeddingModelID
	}
}

func recordSearchTraceCandidates(
	ctx context.Context,
	vectorResults []*types.IndexWithScore,
	keywordResults []*types.IndexWithScore,
) {
	recorder := searchTraceRecorderFromContext(ctx)
	if recorder == nil {
		return
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.trace.VectorCandidates = traceCandidatesFromIndexes(vectorResults)
	recorder.trace.KeywordCandidates = traceCandidatesFromIndexes(keywordResults)
}

func recordSearchTraceFusion(
	ctx context.Context,
	method types.SearchTraceFusionMethod,
	results []*types.IndexWithScore,
	vectorResults []*types.IndexWithScore,
	keywordResults []*types.IndexWithScore,
	retrievalCfg *types.RetrievalConfig,
) {
	recorder := searchTraceRecorderFromContext(ctx)
	if recorder == nil {
		return
	}
	vectorRanks := firstSearchTraceRanks(vectorResults)
	keywordRanks := firstSearchTraceRanks(keywordResults)
	candidates := make([]types.SearchTraceCandidate, 0, len(results))
	for index, result := range results {
		if result == nil {
			continue
		}
		candidates = append(candidates, types.SearchTraceCandidate{
			ChunkID:     result.ChunkID,
			Score:       result.Score,
			Rank:        index + 1,
			VectorRank:  vectorRanks[result.ChunkID],
			KeywordRank: keywordRanks[result.ChunkID],
		})
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.trace.FusionMethod = method
	recorder.trace.FusionCandidates = candidates
	if method == types.SearchTraceFusionRRF {
		recorder.trace.Parameters.RRFK = retrievalCfg.GetEffectiveRRFK()
		vectorWeight, keywordWeight := retrievalCfg.GetEffectiveRRFWeights()
		recorder.trace.Parameters.RRFVectorWeight = vectorWeight
		recorder.trace.Parameters.RRFKeywordWeight = keywordWeight
	}
}

func recordFinalSearchTraceResults(ctx context.Context, results []*types.SearchResult) {
	recorder := searchTraceRecorderFromContext(ctx)
	if recorder == nil {
		return
	}
	candidates := make([]types.SearchTraceCandidate, 0, len(results))
	for index, result := range results {
		if result == nil {
			continue
		}
		candidates = append(candidates, types.SearchTraceCandidate{
			ChunkID: result.ID,
			Score:   result.Score,
			Rank:    index + 1,
		})
	}
	recorder.mu.Lock()
	recorder.trace.FinalChunks = candidates
	recorder.mu.Unlock()
}

func traceCandidatesFromIndexes(results []*types.IndexWithScore) []types.SearchTraceCandidate {
	candidates := make([]types.SearchTraceCandidate, 0, len(results))
	for index, result := range results {
		if result == nil {
			continue
		}
		candidates = append(candidates, types.SearchTraceCandidate{
			ChunkID: result.ChunkID,
			Score:   result.Score,
			Rank:    index + 1,
		})
	}
	return candidates
}

func firstSearchTraceRanks(results []*types.IndexWithScore) map[string]int {
	ranks := make(map[string]int, len(results))
	for index, result := range results {
		if result == nil || result.ChunkID == "" || ranks[result.ChunkID] != 0 {
			continue
		}
		ranks[result.ChunkID] = index + 1
	}
	return ranks
}

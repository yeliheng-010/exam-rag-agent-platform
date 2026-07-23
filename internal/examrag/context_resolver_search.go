package examrag

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func orderExamSearchResults(results []*types.SearchResult) []*types.SearchResult {
	parents := make(map[string]*types.SearchResult)
	for _, result := range results {
		if result != nil && result.MatchType == types.MatchTypeParentChunk {
			parents[result.ID] = result
		}
	}
	ordered := make([]*types.SearchResult, 0, len(results))
	added := make(map[string]bool, len(results))
	appendResult := func(result *types.SearchResult) {
		if result == nil || added[result.ID] {
			return
		}
		added[result.ID] = true
		ordered = append(ordered, result)
	}
	for _, result := range results {
		if result == nil || result.MatchType == types.MatchTypeParentChunk {
			continue
		}
		appendResult(result)
		appendResult(parents[result.ParentChunkID])
	}
	for _, result := range results {
		appendResult(result)
	}
	return ordered
}

func (r *ExamQuestionContextResolver) searchTarget(
	ctx context.Context,
	query string,
	target *types.SearchTarget,
) ([]*types.SearchResult, *types.SearchTrace, error) {
	if target == nil || strings.TrimSpace(target.KnowledgeBaseID) == "" {
		return nil, nil, nil
	}
	params := types.SearchParams{
		QueryText: query, MatchCount: r.matchCount,
		VectorThreshold: r.vectorThreshold, KeywordThreshold: r.keywordThreshold,
		KnowledgeIDs: CleanIDs(target.KnowledgeIDs), TagIDs: CleanIDs(target.TagIDs),
	}
	traceService, traceEnabled := r.knowledgeBaseService.(interfaces.KnowledgeBaseSearchTraceService)
	if traceEnabled {
		results, trace, err := traceService.HybridSearchWithTrace(ctx, target.KnowledgeBaseID, params)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to search knowledge base %s: %w", target.KnowledgeBaseID, err)
		}
		return results, trace, nil
	}
	results, err := r.knowledgeBaseService.HybridSearch(ctx, target.KnowledgeBaseID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search knowledge base %s: %w", target.KnowledgeBaseID, err)
	}
	return results, nil, nil
}

func (r *ExamQuestionContextResolver) selectSearchTargets(
	kbIDs []string,
	tenantID uint64,
) (types.SearchTargets, error) {
	requested := CleanIDs(kbIDs)
	requestedSet := make(map[string]bool, len(requested))
	for _, kbID := range requested {
		requestedSet[kbID] = true
	}
	targets := r.filterSearchTargets(requestedSet, tenantID)
	for kbID := range requestedSet {
		return nil, fmt.Errorf("knowledge base %s is not accessible", kbID)
	}
	return targets, nil
}

func (r *ExamQuestionContextResolver) filterSearchTargets(
	requestedSet map[string]bool,
	tenantID uint64,
) types.SearchTargets {
	targets := make(types.SearchTargets, 0, len(r.searchTargets))
	for _, target := range r.searchTargets {
		if target == nil || strings.TrimSpace(target.KnowledgeBaseID) == "" {
			continue
		}
		if len(requestedSet) > 0 {
			if requestedSet[target.KnowledgeBaseID] {
				targets = append(targets, target)
				delete(requestedSet, target.KnowledgeBaseID)
			}
			continue
		}
		if tenantID == 0 || target.TenantID == 0 || target.TenantID == tenantID {
			targets = append(targets, target)
		}
	}
	return targets
}

func appendChunkIDs(ids []string, seen map[string]bool, values ...string) []string {
	for _, id := range values {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

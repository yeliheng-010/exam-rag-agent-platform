package examrag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

type ExamContextRankedItem = searchutil.ExamContextRankedItem

type rankedExamSearchCandidate struct {
	item        ExamContextRankedItem
	identityIDs []string
	targetOrder int
}

func (r *ExamQuestionContextResolver) searchRankedItemsForKBs(
	ctx context.Context,
	query string,
	kbIDs []string,
	tenantID uint64,
) ([]ExamContextRankedItem, []ExamContextRankedItem, []*types.SearchTrace, error) {
	if r.knowledgeBaseService == nil {
		return nil, nil, nil, fmt.Errorf("knowledge base service is not configured")
	}
	targets, err := r.selectSearchTargets(kbIDs, tenantID)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(targets) == 0 {
		return nil, nil, nil, fmt.Errorf("no accessible knowledge bases available for exam question context search")
	}
	return r.searchRankedItemsForQuery(ctx, query, targets)
}

func (r *ExamQuestionContextResolver) searchRankedItemsForQuery(
	ctx context.Context,
	query string,
	targets types.SearchTargets,
) ([]ExamContextRankedItem, []ExamContextRankedItem, []*types.SearchTrace, error) {
	candidates := make([]rankedExamSearchCandidate, 0, len(targets)*r.matchCount)
	traces := make([]*types.SearchTrace, 0, len(targets))
	for targetOrder, target := range targets {
		results, trace, err := r.searchTarget(ctx, query, target)
		if err != nil {
			return nil, nil, nil, err
		}
		if trace != nil {
			traces = append(traces, trace)
		}
		candidates = append(candidates, rankTargetSearchResults(results, target, targetOrder)...)
	}
	items := fuseRankedSearchCandidates(candidates)
	limit := min(r.matchCount, len(items))
	return cloneRankedItems(items[:limit]), items, traces, nil
}

func rankTargetSearchResults(
	results []*types.SearchResult,
	target *types.SearchTarget,
	targetOrder int,
) []rankedExamSearchCandidate {
	parents := parentSearchResultsByID(results)
	items := make([]rankedExamSearchCandidate, 0, len(results))
	localRank := 0
	for _, result := range results {
		if result == nil || result.MatchType == types.MatchTypeParentChunk {
			continue
		}
		localRank++
		item := rankedItemFromSearchResult(result, parents[result.ParentChunkID], localRank)
		if target != nil {
			item.KnowledgeBaseID = target.KnowledgeBaseID
		}
		items = append(items, rankedExamSearchCandidate{
			item: item, identityIDs: rankedIdentityIDs(result), targetOrder: targetOrder,
		})
	}
	return items
}

func parentSearchResultsByID(results []*types.SearchResult) map[string]*types.SearchResult {
	parents := make(map[string]*types.SearchResult)
	for _, result := range results {
		if result != nil && result.MatchType == types.MatchTypeParentChunk {
			parents[result.ID] = result
		}
	}
	return parents
}

func rankedItemFromSearchResult(
	result *types.SearchResult,
	parent *types.SearchResult,
	localRank int,
) ExamContextRankedItem {
	seen := make(map[string]bool)
	item := ExamContextRankedItem{LocalRank: localRank}
	item.ChunkIDs = appendChunkIDs(item.ChunkIDs, seen, result.ID)
	item.ChunkIDs = appendChunkIDs(item.ChunkIDs, seen, result.SubChunkID...)
	item.Contents = append(item.Contents, result.Content)
	if parent != nil {
		item.ChunkIDs = appendChunkIDs(item.ChunkIDs, seen, parent.ID)
		item.Contents = append(item.Contents, parent.Content)
	}
	return item
}

func fuseRankedSearchCandidates(candidates []rankedExamSearchCandidate) []ExamContextRankedItem {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.item.LocalRank != right.item.LocalRank {
			return left.item.LocalRank < right.item.LocalRank
		}
		if left.targetOrder != right.targetOrder {
			return left.targetOrder < right.targetOrder
		}
		return primaryRankedChunkID(left.item) < primaryRankedChunkID(right.item)
	})
	seen := make(map[string]bool)
	items := make([]ExamContextRankedItem, 0, len(candidates))
	for _, candidate := range candidates {
		if rankedItemOverlaps(candidate.identityIDs, seen) {
			continue
		}
		candidate.item.Rank = len(items) + 1
		items = append(items, candidate.item)
		for _, id := range candidate.identityIDs {
			seen[id] = true
		}
	}
	return items
}

func rankedItemOverlaps(ids []string, seen map[string]bool) bool {
	for _, id := range ids {
		if seen[id] {
			return true
		}
	}
	return false
}

func rankedIdentityIDs(result *types.SearchResult) []string {
	seen := make(map[string]bool)
	ids := appendChunkIDs(nil, seen, result.ID)
	return appendChunkIDs(ids, seen, result.SubChunkID...)
}

func primaryRankedChunkID(item ExamContextRankedItem) string {
	if len(item.ChunkIDs) == 0 {
		return ""
	}
	return item.ChunkIDs[0]
}

func cloneRankedItems(items []ExamContextRankedItem) []ExamContextRankedItem {
	out := make([]ExamContextRankedItem, len(items))
	for index, item := range items {
		out[index] = item
		out[index].ChunkIDs = append([]string{}, item.ChunkIDs...)
		out[index].Contents = append([]string{}, item.Contents...)
	}
	return out
}

func rankedItemChunkIDs(items []ExamContextRankedItem) []string {
	seen := make(map[string]bool)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = appendChunkIDs(ids, seen, item.ChunkIDs...)
	}
	return ids
}

func rankedItemContents(items []ExamContextRankedItem) []string {
	contents := make([]string, 0, len(items))
	for _, item := range items {
		contents = append(contents, strings.Join(item.Contents, "\n\n"))
	}
	return contents
}

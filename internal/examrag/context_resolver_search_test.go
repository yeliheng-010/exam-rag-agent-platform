package examrag

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type rankedSearchKBService struct {
	interfaces.KnowledgeBaseService
	results map[string][]*types.SearchResult
}

func (s *rankedSearchKBService) HybridSearch(
	_ context.Context,
	kbID string,
	_ types.SearchParams,
) ([]*types.SearchResult, error) {
	return s.results[kbID], nil
}

func TestSearchRankedItemsForQueryAppliesGlobalTopKAcrossKnowledgeBases(t *testing.T) {
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		KnowledgeBaseService: &rankedSearchKBService{results: map[string][]*types.SearchResult{
			"kb-a": {
				{ID: "a-1", Content: "a first", Score: 0.01},
				{ID: "a-2", Content: "a second", Score: 0.99},
				{ID: "a-3", Content: "a third", Score: 0.98},
			},
			"kb-b": {
				{ID: "b-1", Content: "b first", Score: 100},
				{ID: "b-2", Content: "b second", Score: 90},
				{ID: "b-3", Content: "b third", Score: 80},
			},
		}},
		MatchCount: 3,
	})
	targets := types.SearchTargets{
		{KnowledgeBaseID: "kb-a", TenantID: 10000},
		{KnowledgeBaseID: "kb-b", TenantID: 10000},
	}

	finalItems, candidateItems, _, err := resolver.searchRankedItemsForQuery(
		context.Background(), "query", targets,
	)

	require.NoError(t, err)
	require.Equal(t, []string{"a-1", "b-1", "a-2"}, rankedItemPrimaryIDs(finalItems))
	require.Equal(t, []string{"a-1", "b-1", "a-2", "b-2", "a-3", "b-3"}, rankedItemPrimaryIDs(candidateItems))
	require.Equal(t, []int{1, 2, 3}, rankedItemRanks(finalItems))
}

func TestSearchRankedItemsForQueryKeepsParentWithChildWithoutConsumingRank(t *testing.T) {
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		KnowledgeBaseService: &rankedSearchKBService{results: map[string][]*types.SearchResult{
			"kb-a": {
				{ID: "child-a", Content: "child context", ParentChunkID: "parent-a", SubChunkID: []string{"sub-a"}},
				{ID: "child-b", Content: "second child"},
				{ID: "parent-a", Content: "parent context", MatchType: types.MatchTypeParentChunk},
				{ID: "orphan-parent", Content: "orphan", MatchType: types.MatchTypeParentChunk},
			},
		}},
		MatchCount: 2,
	})

	finalItems, candidateItems, _, err := resolver.searchRankedItemsForQuery(
		context.Background(), "query", types.SearchTargets{{KnowledgeBaseID: "kb-a", TenantID: 10000}},
	)

	require.NoError(t, err)
	require.Len(t, finalItems, 2)
	require.Len(t, candidateItems, 2)
	require.Equal(t, []string{"child-a", "sub-a", "parent-a"}, finalItems[0].ChunkIDs)
	require.Equal(t, []string{"child context", "parent context"}, finalItems[0].Contents)
	require.Equal(t, "child-b", finalItems[1].ChunkIDs[0])
	require.Equal(t, 2, finalItems[1].Rank)
}

func TestSearchRankedItemsForQueryKeepsSiblingChildrenThatShareAParent(t *testing.T) {
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		KnowledgeBaseService: &rankedSearchKBService{results: map[string][]*types.SearchResult{
			"kb-a": {
				{ID: "child-a", Content: "first child", ParentChunkID: "parent-a"},
				{ID: "child-b", Content: "second child", ParentChunkID: "parent-a"},
				{ID: "parent-a", Content: "shared parent", MatchType: types.MatchTypeParentChunk},
			},
		}},
		MatchCount: 2,
	})

	finalItems, candidateItems, _, err := resolver.searchRankedItemsForQuery(
		context.Background(), "query", types.SearchTargets{{KnowledgeBaseID: "kb-a", TenantID: 10000}},
	)

	require.NoError(t, err)
	require.Equal(t, []string{"child-a", "child-b"}, rankedItemPrimaryIDs(finalItems))
	require.Equal(t, []string{"child-a", "child-b"}, rankedItemPrimaryIDs(candidateItems))
	require.Equal(t, []int{1, 2}, rankedItemRanks(finalItems))
	require.Equal(t, []string{"second child", "shared parent"}, finalItems[1].Contents)
}

func TestSearchRankedItemsForQueryUsesStableTargetOrderAndDeduplicatesAliases(t *testing.T) {
	resolver := NewExamQuestionContextResolver(ExamQuestionContextResolverConfig{
		KnowledgeBaseService: &rankedSearchKBService{results: map[string][]*types.SearchResult{
			"kb-b": {{ID: "b-1", SubChunkID: []string{"shared"}}, {ID: "b-2"}},
			"kb-a": {{ID: "a-1", SubChunkID: []string{"shared"}}, {ID: "a-2"}},
		}},
		MatchCount: 3,
	})

	finalItems, candidateItems, _, err := resolver.searchRankedItemsForQuery(
		context.Background(), "query", types.SearchTargets{
			{KnowledgeBaseID: "kb-b", TenantID: 10000},
			{KnowledgeBaseID: "kb-a", TenantID: 10000},
		},
	)

	require.NoError(t, err)
	require.Equal(t, []string{"b-1", "b-2", "a-2"}, rankedItemPrimaryIDs(candidateItems))
	require.Equal(t, rankedItemPrimaryIDs(candidateItems), rankedItemPrimaryIDs(finalItems))
}

func TestOrderExamSearchResultsPlacesParentAfterMatchedChild(t *testing.T) {
	results := []*types.SearchResult{
		{ID: "child-a", ParentChunkID: "parent-a"},
		{ID: "child-b", ParentChunkID: "parent-b"},
		{ID: "parent-b", MatchType: types.MatchTypeParentChunk},
		{ID: "parent-a", MatchType: types.MatchTypeParentChunk},
	}

	ordered := orderExamSearchResults(results)

	require.Equal(t, []string{"child-a", "parent-a", "child-b", "parent-b"}, searchResultIDs(ordered))
}

func searchResultIDs(results []*types.SearchResult) []string {
	ids := make([]string, 0, len(results))
	for _, result := range results {
		if result != nil {
			ids = append(ids, result.ID)
		}
	}
	return ids
}

func rankedItemPrimaryIDs(items []ExamContextRankedItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if len(item.ChunkIDs) > 0 {
			ids = append(ids, item.ChunkIDs[0])
		}
	}
	return ids
}

func rankedItemRanks(items []ExamContextRankedItem) []int {
	ranks := make([]int, 0, len(items))
	for _, item := range items {
		ranks = append(ranks, item.Rank)
	}
	return ranks
}

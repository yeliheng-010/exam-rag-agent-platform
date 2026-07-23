package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestAssembleSearchResultsIncludesParentContext(t *testing.T) {
	service := &knowledgeBaseService{}
	child := &types.Chunk{
		ID: "child", KnowledgeID: "knowledge", ChunkType: types.ChunkTypeText,
		Content: "Which team will play the most games?", ParentChunkID: "parent",
	}
	parent := &types.Chunk{
		ID: "parent", KnowledgeID: "knowledge", ChunkType: types.ChunkTypeParentText,
		Content: "Los Angeles Chargers v Los Angeles Rams",
	}
	input := []*types.IndexWithScore{{
		ChunkID: "child", KnowledgeID: "knowledge", Score: 0.9,
	}}
	index := service.buildChunkIndex(input)
	require.Equal(t, []string{"parent"}, service.collectEnrichmentChunkIDs(
		context.Background(), []*types.Chunk{child}, index,
	))

	results := service.assembleSearchResults(
		context.Background(), input,
		map[string]*types.Chunk{"child": child, "parent": parent},
		map[string]*types.Knowledge{"knowledge": {ID: "knowledge"}},
		index, false,
	)

	require.Len(t, results, 2)
	require.Equal(t, "parent", results[1].ID)
	require.Equal(t, types.MatchTypeParentChunk, results[1].MatchType)
	require.Contains(t, results[1].Content, "Los Angeles Chargers v Los Angeles Rams")
}

func TestAssembleSearchResultsRejectsNonParentEnrichmentParentText(t *testing.T) {
	service := &knowledgeBaseService{}
	parent := &types.Chunk{
		ID: "parent", KnowledgeID: "knowledge", ChunkType: types.ChunkTypeParentText,
	}
	knowledgeMap := map[string]*types.Knowledge{"knowledge": {ID: "knowledge"}}

	tests := []struct {
		name      string
		input     []*types.IndexWithScore
		matchType types.MatchType
	}{
		{
			name: "direct retrieval",
			input: []*types.IndexWithScore{{
				ChunkID: "parent", KnowledgeID: "knowledge", Score: 0.9,
			}},
			matchType: types.MatchTypeEmbedding,
		},
		{
			name:      "relation enrichment",
			matchType: types.MatchTypeRelationChunk,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := service.buildChunkIndex(tt.input)
			index.matchTypes[parent.ID] = tt.matchType
			results := service.assembleSearchResults(
				context.Background(), tt.input, map[string]*types.Chunk{"parent": parent},
				knowledgeMap, index, false,
			)

			require.Empty(t, results)
		})
	}
}

package service

import (
	"encoding/json"
	"slices"

	"github.com/Tencent/WeKnora/internal/types"
)

// collectRelatedChunkIDs extracts related chunk IDs from a chunk.
func (s *knowledgeBaseService) collectRelatedChunkIDs(chunk *types.Chunk, processedIDs map[string]bool) []string {
	var relatedIDs []string
	if len(chunk.RelationChunks) > 0 {
		var relations []string
		if err := json.Unmarshal(chunk.RelationChunks, &relations); err == nil {
			for _, id := range relations {
				if !processedIDs[id] {
					relatedIDs = append(relatedIDs, id)
					processedIDs[id] = true
				}
			}
		}
	}
	return relatedIDs
}

// buildSearchResult creates a search result from chunk and knowledge.
func (s *knowledgeBaseService) buildSearchResult(chunk *types.Chunk,
	knowledge *types.Knowledge,
	score float64,
	matchType types.MatchType,
	matchedContent string,
) *types.SearchResult {
	return &types.SearchResult{
		ID:                   chunk.ID,
		Content:              chunk.Content,
		KnowledgeID:          chunk.KnowledgeID,
		ChunkIndex:           chunk.ChunkIndex,
		KnowledgeTitle:       knowledge.Title,
		StartAt:              chunk.StartAt,
		EndAt:                chunk.EndAt,
		Seq:                  chunk.ChunkIndex,
		Score:                score,
		MatchType:            matchType,
		Metadata:             knowledge.GetMetadata(),
		ChunkType:            string(chunk.ChunkType),
		ParentChunkID:        chunk.ParentChunkID,
		ImageInfo:            chunk.ImageInfo,
		KnowledgeFilename:    knowledge.FileName,
		KnowledgeSource:      knowledge.Source,
		KnowledgeChannel:     knowledge.Channel,
		KnowledgeDescription: knowledge.Description,
		ChunkMetadata:        chunk.Metadata,
		MatchedContent:       matchedContent,
		KnowledgeBaseID:      knowledge.KnowledgeBaseID,
	}
}

func (s *knowledgeBaseService) isSearchableChunk(chunk *types.Chunk) bool {
	return slices.Contains([]types.ChunkType{
		types.ChunkTypeText, types.ChunkTypeSummary,
		types.ChunkTypeTableColumn, types.ChunkTypeTableSummary,
		types.ChunkTypeFAQ,
		types.ChunkTypeImageOCR, types.ChunkTypeImageCaption,
	}, chunk.ChunkType)
}

func (s *knowledgeBaseService) isEnrichmentSearchableChunk(
	chunk *types.Chunk,
	matchType types.MatchType,
) bool {
	return s.isSearchableChunk(chunk) ||
		(matchType == types.MatchTypeParentChunk && chunk.ChunkType == types.ChunkTypeParentText)
}

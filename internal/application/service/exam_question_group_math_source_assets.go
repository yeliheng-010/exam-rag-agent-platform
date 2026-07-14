package service

import (
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var mathLocalAssetPattern = regexp.MustCompile(`local://[^)\s]+`)

func extractMathSourceAssets(content string, sourceIDs []string) []types.ExamQuestionGroupDraftAssetCandidate {
	matches := mathLocalAssetPattern.FindAllString(content, -1)
	assets := make([]types.ExamQuestionGroupDraftAssetCandidate, 0, len(matches))
	seen := map[string]bool{}
	for _, uri := range matches {
		uri = strings.TrimRight(uri, ".,;:，。；：")
		if seen[uri] {
			continue
		}
		seen[uri] = true
		sourceID := ""
		if len(sourceIDs) > 0 {
			sourceID = sourceIDs[0]
		}
		assets = append(assets, types.ExamQuestionGroupDraftAssetCandidate{
			AssetType: "image", StorageURI: uri, AltText: "题目公式或图形", SourceChunkID: sourceID,
			BBox: types.JSONMap{}, Metadata: types.JSONMap{}, SortOrder: len(assets) + 1,
		})
	}
	return assets
}

func uniqueSortedMathSourceChunks(groups ...[]*types.Chunk) []*types.Chunk {
	seen := map[string]bool{}
	chunks := make([]*types.Chunk, 0)
	for _, group := range groups {
		for _, chunk := range group {
			if chunk == nil || seen[questionPromptChunkKey(chunk)] {
				continue
			}
			seen[questionPromptChunkKey(chunk)] = true
			chunks = append(chunks, chunk)
		}
	}
	return sortedQuestionGroupChunks(chunks)
}

func joinMathSourceChunkContent(chunks []*types.Chunk) string {
	parts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil {
			parts = append(parts, chunk.Content)
		}
	}
	return strings.Join(parts, "\n")
}

func mathSourceChunkIDs(chunks []*types.Chunk) []string {
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && chunk.ID != "" {
			ids = append(ids, chunk.ID)
		}
	}
	return ids
}

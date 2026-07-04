package chatpipeline

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

func (p *PluginSearch) enrichExamQuestionContext(
	ctx context.Context,
	chatManage *types.ChatManage,
	results []*types.SearchResult,
) []*types.SearchResult {
	if p == nil || p.chunkService == nil || chatManage == nil || len(results) == 0 {
		return results
	}

	query := chatManage.RewriteQuery
	if query == "" {
		query = chatManage.Query
	}
	if !searchutil.ShouldEnrichExamQuestionContext(query) {
		return results
	}

	seenKnowledge := make(map[string]bool)
	enriched := results
	for _, seed := range results {
		if seed == nil || seed.KnowledgeID == "" || seed.KnowledgeBaseID == "" || seenKnowledge[seed.KnowledgeID] {
			continue
		}
		seenKnowledge[seed.KnowledgeID] = true

		tenantID := tenantIDForExamContext(chatManage, seed.KnowledgeBaseID)
		chunks, err := p.chunkService.GetRepository().ListChunksByKnowledgeID(ctx, tenantID, seed.KnowledgeID)
		if err != nil {
			pipelineWarn(ctx, "Search", "exam_context_list_chunks_failed", map[string]interface{}{
				"knowledge_id": seed.KnowledgeID,
				"error":        err.Error(),
			})
			continue
		}
		bundle := searchutil.BuildExamQuestionContextBundle(query, chunks)
		if bundle == nil {
			continue
		}

		enrichedResult := buildExamContextSearchResult(seed, bundle)
		enriched = upsertExamContextResult(enriched, enrichedResult)
		pipelineInfo(ctx, "Search", "exam_context_enriched", map[string]interface{}{
			"knowledge_id":   seed.KnowledgeID,
			"label":          bundle.Label,
			"body_chunks":    len(bundle.BodyChunks),
			"answer_chunks":  len(bundle.AnswerChunks),
			"primary_chunk":  enrichedResult.ID,
			"source_chunks":  len(enrichedResult.SubChunkID) + 1,
			"content_length": len([]rune(enrichedResult.Content)),
		})
		break
	}
	return enriched
}

func tenantIDForExamContext(chatManage *types.ChatManage, kbID string) uint64 {
	if chatManage != nil {
		if tenantID := chatManage.SearchTargets.GetTenantIDForKB(kbID); tenantID != 0 {
			return tenantID
		}
		return chatManage.TenantID
	}
	return 0
}

func buildExamContextSearchResult(
	seed *types.SearchResult,
	bundle *searchutil.ExamQuestionContextBundle,
) *types.SearchResult {
	if seed == nil || bundle == nil || len(bundle.BodyChunks) == 0 {
		return seed
	}

	primary := bundle.BodyChunks[0]
	out := *seed
	out.ID = primary.ID
	out.Content = bundle.Content
	out.ChunkIndex = primary.ChunkIndex
	out.StartAt = primary.StartAt
	out.EndAt = examBundleEndAt(bundle)
	out.Seq = primary.ChunkIndex
	out.Score = 1.0
	out.MatchType = types.MatchTypeDirectLoad
	out.ChunkType = string(types.ChunkTypeText)
	out.ParentChunkID = primary.ParentChunkID
	out.ImageInfo = primary.ImageInfo
	out.SubChunkID = examBundleSourceIDs(bundle, primary.ID)
	out.Metadata = cloneSearchMetadata(seed.Metadata)
	out.Metadata["exam_context_enriched"] = "true"
	out.Metadata["exam_passage_label"] = bundle.Label
	out.Metadata["source_chunk_count"] = fmt.Sprintf("%d", len(out.SubChunkID)+1)
	return &out
}

func examBundleEndAt(bundle *searchutil.ExamQuestionContextBundle) int {
	end := 0
	for _, chunk := range bundle.BodyChunks {
		if chunk != nil && chunk.EndAt > end {
			end = chunk.EndAt
		}
	}
	return end
}

func examBundleSourceIDs(bundle *searchutil.ExamQuestionContextBundle, primaryID string) []string {
	seen := map[string]bool{primaryID: true}
	out := make([]string, 0, len(bundle.BodyChunks)+len(bundle.AnswerChunks))
	add := func(chunks []*types.Chunk) {
		for _, chunk := range chunks {
			if chunk == nil || chunk.ID == "" || seen[chunk.ID] {
				continue
			}
			seen[chunk.ID] = true
			out = append(out, chunk.ID)
		}
	}
	add(bundle.BodyChunks)
	add(bundle.AnswerChunks)
	return out
}

func upsertExamContextResult(results []*types.SearchResult, enriched *types.SearchResult) []*types.SearchResult {
	if enriched == nil || enriched.ID == "" {
		return results
	}
	out := make([]*types.SearchResult, 0, len(results)+1)
	out = append(out, enriched)
	for _, result := range results {
		if result == nil || result.ID == enriched.ID {
			continue
		}
		out = append(out, result)
	}
	return out
}

func cloneSearchMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata)+3)
	for k, v := range metadata {
		out[k] = v
	}
	return out
}

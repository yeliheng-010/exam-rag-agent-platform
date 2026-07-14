package chatpipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

func (p *PluginSearch) enrichExamQuestionContext(
	ctx context.Context,
	chatManage *types.ChatManage,
	results []*types.SearchResult,
) []*types.SearchResult {
	if p == nil || chatManage == nil || len(results) == 0 {
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
		bundle, err := p.structuredExamQuestionContextBundle(ctx, tenantID, query, seed)
		if err != nil {
			pipelineWarn(ctx, "Search", "exam_structured_context_failed", map[string]interface{}{
				"knowledge_id": seed.KnowledgeID,
				"chunk_id":     seed.ID,
				"error":        err.Error(),
			})
		}
		if bundle != nil {
			enrichedResult := buildExamContextSearchResult(seed, bundle)
			enriched = upsertExamContextResult(enriched, enrichedResult)
			pipelineInfo(ctx, "Search", "exam_structured_context_enriched", map[string]interface{}{
				"knowledge_id":   seed.KnowledgeID,
				"label":          bundle.Label,
				"primary_chunk":  enrichedResult.ID,
				"source_chunks":  len(enrichedResult.SubChunkID) + 1,
				"content_length": len([]rune(enrichedResult.Content)),
			})
			break
		}

		if p.chunkService == nil {
			continue
		}
		chunks, err := p.chunkService.GetRepository().ListChunksByKnowledgeID(ctx, tenantID, seed.KnowledgeID)
		if err != nil {
			pipelineWarn(ctx, "Search", "exam_context_list_chunks_failed", map[string]interface{}{
				"knowledge_id": seed.KnowledgeID,
				"error":        err.Error(),
			})
			continue
		}
		fallbackBundle := searchutil.BuildExamQuestionContextBundle(query, chunks)
		if fallbackBundle == nil {
			continue
		}
		bundle, err = p.structuredExamQuestionContextBundleByChunkIDs(ctx, tenantID, query, examBundleChunkIDs(fallbackBundle))
		if err != nil {
			pipelineWarn(ctx, "Search", "exam_structured_context_from_bundle_failed", map[string]interface{}{
				"knowledge_id": seed.KnowledgeID,
				"error":        err.Error(),
			})
		}
		if bundle != nil {
			enrichedResult := buildExamContextSearchResult(seed, bundle)
			enriched = upsertExamContextResult(enriched, enrichedResult)
			pipelineInfo(ctx, "Search", "exam_structured_context_enriched", map[string]interface{}{
				"knowledge_id":   seed.KnowledgeID,
				"label":          bundle.Label,
				"primary_chunk":  enrichedResult.ID,
				"source_chunks":  len(enrichedResult.SubChunkID) + 1,
				"content_length": len([]rune(enrichedResult.Content)),
			})
			break
		}

		enrichedResult := buildExamContextSearchResult(seed, fallbackBundle)
		enriched = upsertExamContextResult(enriched, enrichedResult)
		pipelineInfo(ctx, "Search", "exam_context_enriched", map[string]interface{}{
			"knowledge_id":   seed.KnowledgeID,
			"label":          fallbackBundle.Label,
			"body_chunks":    len(fallbackBundle.BodyChunks),
			"answer_chunks":  len(fallbackBundle.AnswerChunks),
			"primary_chunk":  enrichedResult.ID,
			"source_chunks":  len(enrichedResult.SubChunkID) + 1,
			"content_length": len([]rune(enrichedResult.Content)),
		})
		break
	}
	return enriched
}

func (p *PluginSearch) structuredExamQuestionContextBundle(
	ctx context.Context,
	tenantID uint64,
	query string,
	seed *types.SearchResult,
) (*searchutil.ExamQuestionContextBundle, error) {
	return p.structuredExamQuestionContextBundleByChunkIDs(ctx, tenantID, query, examContextCandidateChunkIDs(seed))
}

func (p *PluginSearch) structuredExamQuestionContextBundleByChunkIDs(
	ctx context.Context,
	tenantID uint64,
	query string,
	chunkIDs []string,
) (*searchutil.ExamQuestionContextBundle, error) {
	if p == nil || p.questionRepo == nil || len(chunkIDs) == 0 || tenantID == 0 {
		return nil, nil
	}
	resolved, err := examrag.NewExamQuestionContextResolver(examrag.ExamQuestionContextResolverConfig{
		QuestionRepo:         p.questionRepo,
		KnowledgeBaseService: p.knowledgeBaseService,
	}).Resolve(ctx, examrag.ExamQuestionContextResolveRequest{
		Query:    query,
		TenantID: tenantID,
		ChunkIDs: chunkIDs,
	})
	if err != nil || resolved == nil {
		return nil, err
	}
	return resolved.Bundle, nil
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
	if seed == nil || bundle == nil {
		return seed
	}

	out := *seed
	out.Content = bundle.Content
	out.Score = 1.0
	out.MatchType = types.MatchTypeDirectLoad
	out.ChunkType = string(types.ChunkTypeText)

	if len(bundle.BodyChunks) > 0 && bundle.BodyChunks[0] != nil {
		primary := bundle.BodyChunks[0]
		out.ID = primary.ID
		out.ChunkIndex = primary.ChunkIndex
		out.StartAt = primary.StartAt
		out.EndAt = examBundleEndAt(bundle)
		out.Seq = primary.ChunkIndex
		out.ParentChunkID = primary.ParentChunkID
		out.ImageInfo = primary.ImageInfo
	}

	out.SubChunkID = examBundleSourceIDs(bundle, out.ID)
	out.Metadata = cloneSearchMetadata(seed.Metadata)
	out.Metadata["exam_context_enriched"] = "true"
	out.Metadata["exam_passage_label"] = bundle.Label
	out.Metadata["exam_context_mode"] = examContextBundleMode(bundle)
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
	for _, chunkID := range bundle.SourceChunkIDs {
		if chunkID == "" || seen[chunkID] {
			continue
		}
		seen[chunkID] = true
		out = append(out, chunkID)
	}
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

func examBundleChunkIDs(bundle *searchutil.ExamQuestionContextBundle) []string {
	if bundle == nil {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(bundle.SourceChunkIDs)+len(bundle.BodyChunks)+len(bundle.AnswerChunks))
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range bundle.SourceChunkIDs {
		add(id)
	}
	for _, chunk := range bundle.BodyChunks {
		if chunk != nil {
			add(chunk.ID)
		}
	}
	for _, chunk := range bundle.AnswerChunks {
		if chunk != nil {
			add(chunk.ID)
		}
	}
	return out
}

func examContextCandidateChunkIDs(seed *types.SearchResult) []string {
	if seed == nil {
		return nil
	}
	out := make([]string, 0, 1+len(seed.SubChunkID))
	if seed.ID != "" {
		out = append(out, seed.ID)
	}
	out = append(out, seed.SubChunkID...)
	return out
}

func examContextBundleMode(bundle *searchutil.ExamQuestionContextBundle) string {
	if bundle != nil && len(bundle.BodyChunks) == 0 && len(bundle.SourceChunkIDs) > 0 {
		return "structured"
	}
	return "chunk"
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

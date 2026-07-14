package tools

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/examrag"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/searchutil"
	"github.com/Tencent/WeKnora/internal/types"
)

func (t *KnowledgeSearchTool) appendExamContextBundles(
	ctx context.Context,
	queries []string,
	results []*searchResultWithMeta,
) []*searchResultWithMeta {
	if t == nil || len(results) == 0 {
		return results
	}

	query := strings.Join(queries, " ")
	if !searchutil.ShouldEnrichExamQuestionContext(query) {
		return results
	}

	seenKnowledge := make(map[string]bool)
	out := results
	for _, seed := range results {
		if seed == nil || seed.SearchResult == nil || seed.KnowledgeID == "" || seed.KnowledgeBaseID == "" || seenKnowledge[seed.KnowledgeID] {
			continue
		}
		seenKnowledge[seed.KnowledgeID] = true

		tenantID := t.searchTargets.GetTenantIDForKB(seed.KnowledgeBaseID)
		if tenantID == 0 {
			logger.Warnf(ctx, "[Tool][KnowledgeSearch] Skip exam context for KB %s: tenant not found", seed.KnowledgeBaseID)
			continue
		}

		bundle, err := t.structuredExamQuestionContextBundle(ctx, tenantID, query, seed)
		if err != nil {
			logger.Warnf(ctx, "[Tool][KnowledgeSearch] Failed to build structured exam context, knowledge=%s chunk=%s: %v",
				seed.KnowledgeID, seed.ID, err)
		}
		if bundle != nil {
			enriched := buildToolExamContextResult(seed, bundle, query)
			out = upsertToolExamContextResult(out, enriched)
			logger.Infof(ctx, "[Tool][KnowledgeSearch] Structured exam context enriched: knowledge=%s label=%s sources=%d",
				seed.KnowledgeID, bundle.Label, len(bundle.SourceChunkIDs))
			break
		}

		if t.chunkService == nil {
			continue
		}
		chunks, err := t.chunkService.GetRepository().ListChunksByKnowledgeID(ctx, tenantID, seed.KnowledgeID)
		if err != nil {
			logger.Warnf(ctx, "[Tool][KnowledgeSearch] Failed to list chunks for exam context, knowledge=%s: %v", seed.KnowledgeID, err)
			continue
		}
		fallbackBundle := searchutil.BuildExamQuestionContextBundle(query, chunks)
		if fallbackBundle == nil {
			continue
		}

		enriched := buildToolExamContextResult(seed, fallbackBundle, query)
		out = upsertToolExamContextResult(out, enriched)
		logger.Infof(ctx, "[Tool][KnowledgeSearch] Exam context enriched: knowledge=%s label=%s body=%d answer=%d",
			seed.KnowledgeID, fallbackBundle.Label, len(fallbackBundle.BodyChunks), len(fallbackBundle.AnswerChunks))
		break
	}
	return out
}

func (t *KnowledgeSearchTool) structuredExamQuestionContextBundle(
	ctx context.Context,
	tenantID uint64,
	query string,
	seed *searchResultWithMeta,
) (*searchutil.ExamQuestionContextBundle, error) {
	if t == nil || t.questionRepo == nil || seed == nil || seed.SearchResult == nil || tenantID == 0 {
		return nil, nil
	}
	resolved, err := examrag.NewExamQuestionContextResolver(examrag.ExamQuestionContextResolverConfig{
		QuestionRepo:         t.questionRepo,
		KnowledgeBaseService: t.knowledgeBaseService,
		SearchTargets:        t.searchTargets,
	}).Resolve(ctx, examrag.ExamQuestionContextResolveRequest{
		Query:    query,
		TenantID: tenantID,
		ChunkIDs: toolExamContextCandidateChunkIDs(seed),
	})
	if err != nil || resolved == nil {
		return nil, err
	}
	return resolved.Bundle, nil
}

func buildToolExamContextResult(
	seed *searchResultWithMeta,
	bundle *searchutil.ExamQuestionContextBundle,
	query string,
) *searchResultWithMeta {
	if seed == nil || seed.SearchResult == nil || bundle == nil {
		return seed
	}

	base := *seed.SearchResult
	base.Content = bundle.Content
	base.Score = 1.0
	base.MatchType = types.MatchTypeDirectLoad
	base.ChunkType = string(types.ChunkTypeText)
	if len(bundle.BodyChunks) > 0 && bundle.BodyChunks[0] != nil {
		primary := bundle.BodyChunks[0]
		base.ID = primary.ID
		base.ChunkIndex = primary.ChunkIndex
		base.StartAt = primary.StartAt
		base.EndAt = examToolBundleEndAt(bundle)
		base.Seq = primary.ChunkIndex
		base.ParentChunkID = primary.ParentChunkID
		base.ImageInfo = primary.ImageInfo
	}
	base.SubChunkID = examToolBundleSourceIDs(bundle, base.ID)
	base.Metadata = cloneToolSearchMetadata(seed.Metadata)
	base.Metadata["exam_context_enriched"] = "true"
	base.Metadata["exam_passage_label"] = bundle.Label
	base.Metadata["exam_context_mode"] = examToolContextBundleMode(bundle)

	return &searchResultWithMeta{
		SearchResult:      &base,
		SourceQuery:       query,
		QueryType:         "exam_context",
		KnowledgeBaseID:   seed.KnowledgeBaseID,
		KnowledgeBaseType: seed.KnowledgeBaseType,
	}
}

func examToolBundleEndAt(bundle *searchutil.ExamQuestionContextBundle) int {
	end := 0
	for _, chunk := range bundle.BodyChunks {
		if chunk != nil && chunk.EndAt > end {
			end = chunk.EndAt
		}
	}
	return end
}

func examToolBundleSourceIDs(bundle *searchutil.ExamQuestionContextBundle, primaryID string) []string {
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

func toolExamContextCandidateChunkIDs(seed *searchResultWithMeta) []string {
	if seed == nil || seed.SearchResult == nil {
		return nil
	}
	out := make([]string, 0, 1+len(seed.SubChunkID))
	if seed.ID != "" {
		out = append(out, seed.ID)
	}
	out = append(out, seed.SubChunkID...)
	return out
}

func examToolContextBundleMode(bundle *searchutil.ExamQuestionContextBundle) string {
	if bundle != nil && len(bundle.BodyChunks) == 0 && len(bundle.SourceChunkIDs) > 0 {
		return "structured"
	}
	return "chunk"
}

func upsertToolExamContextResult(results []*searchResultWithMeta, enriched *searchResultWithMeta) []*searchResultWithMeta {
	if enriched == nil || enriched.ID == "" {
		return results
	}
	out := make([]*searchResultWithMeta, 0, len(results)+1)
	out = append(out, enriched)
	for _, result := range results {
		if result == nil || result.ID == enriched.ID {
			continue
		}
		out = append(out, result)
	}
	return out
}

func cloneToolSearchMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata)+2)
	for k, v := range metadata {
		out[k] = v
	}
	return out
}

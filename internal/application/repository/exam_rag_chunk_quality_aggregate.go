package repository

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/infrastructure/chunker"
	"github.com/Tencent/WeKnora/internal/types"
)

func newChunkQualityAccumulators(
	kbs []chunkQualityKBRow,
	knowledge []chunkQualityKnowledgeRow,
) map[string]*chunkQualityAccumulator {
	result := make(map[string]*chunkQualityAccumulator, len(kbs))
	for _, kb := range kbs {
		config := decodeChunkingConfig(kb.ChunkingConfig)
		result[kb.ID] = &chunkQualityAccumulator{snapshot: types.ExamRAGChunkingSnapshot{
			KnowledgeBaseID:  kb.ID,
			Config:           snapshotChunkingConfig(config),
			ActualTierCounts: map[string]int{},
		}}
	}
	for _, item := range knowledge {
		if acc := result[item.KnowledgeBaseID]; acc != nil {
			acc.snapshot.KnowledgeCount++
		}
	}
	return result
}

func finalizeChunkQualitySnapshots(
	requested []string,
	accumulators map[string]*chunkQualityAccumulator,
) []types.ExamRAGChunkingSnapshot {
	result := make([]types.ExamRAGChunkingSnapshot, 0, len(accumulators))
	for _, id := range requested {
		acc := accumulators[id]
		if acc == nil {
			continue
		}
		finalizeChunkQualityStats(acc)
		result = append(result, acc.snapshot)
	}
	return result
}

func finalizeChunkQualityStats(acc *chunkQualityAccumulator) {
	if len(acc.lengths) == 0 {
		return
	}
	sort.Ints(acc.lengths)
	target := acc.snapshot.Config.ChunkSize
	if acc.snapshot.Config.EnableParentChild {
		target = acc.snapshot.Config.ChildChunkSize
	}
	var tiny, oversize int
	for _, length := range acc.lengths {
		if target > 0 && length < target/4 {
			tiny++
		}
		if target > 0 && length > target {
			oversize++
		}
	}
	count := len(acc.lengths)
	acc.snapshot.TextChunkCount = count
	acc.snapshot.MinChars = acc.lengths[0]
	acc.snapshot.P50Chars = chunkQualityNearestRank(acc.lengths, 0.50)
	acc.snapshot.P90Chars = chunkQualityNearestRank(acc.lengths, 0.90)
	acc.snapshot.MaxChars = acc.lengths[count-1]
	acc.snapshot.TinyChunkRate = float64(tiny) / float64(count)
	acc.snapshot.OversizeRate = float64(oversize) / float64(count)
	acc.snapshot.ParentCoverage = float64(acc.parentLinked) / float64(count)
}

func decodeChunkQualityDiagnostics(output types.JSONMap) (chunkQualitySpanDiagnostics, bool) {
	var diagnostics chunkQualitySpanDiagnostics
	if len(output) == 0 {
		return diagnostics, false
	}
	raw, err := json.Marshal(output)
	if err != nil || json.Unmarshal(raw, &diagnostics) != nil || strings.TrimSpace(diagnostics.ParentTier) == "" {
		return chunkQualitySpanDiagnostics{}, false
	}
	return diagnostics, true
}

func decodeChunkingConfig(raw types.JSON) types.ChunkingConfig {
	var config types.ChunkingConfig
	_ = json.Unmarshal(raw, &config)
	return config
}

func snapshotChunkingConfig(config types.ChunkingConfig) types.ChunkingConfigSnapshot {
	if config.ChunkSize <= 0 {
		config.ChunkSize = chunker.DefaultChunkSize
	}
	if config.ChunkOverlap <= 0 {
		config.ChunkOverlap = chunker.DefaultChunkOverlap
	}
	if config.ParentChunkSize <= 0 {
		config.ParentChunkSize = 4096
	}
	if config.ChildChunkSize <= 0 {
		config.ChildChunkSize = 384
	}
	strategy := strings.TrimSpace(config.Strategy)
	if strategy == "" {
		strategy = chunker.StrategyLegacy
	}
	return types.ChunkingConfigSnapshot{
		Strategy: strategy, ChunkSize: config.ChunkSize, ChunkOverlap: config.ChunkOverlap,
		EnableParentChild: config.EnableParentChild, ParentChunkSize: config.ParentChunkSize,
		ChildChunkSize: config.ChildChunkSize, TokenLimit: config.TokenLimit,
		Languages: append([]string{}, config.Languages...),
	}
}

func chunkQualityNearestRank(sorted []int, percentile float64) int {
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func uniqueChunkQualityIDs(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func kbIDs(rows []chunkQualityKBRow) []string {
	result := make([]string, len(rows))
	for index, row := range rows {
		result[index] = row.ID
	}
	return result
}

func knowledgeIDs(rows []chunkQualityKnowledgeRow) []string {
	result := make([]string, len(rows))
	for index, row := range rows {
		result[index] = row.ID
	}
	return result
}

func nilIfError(err error) ([]types.ExamRAGChunkingSnapshot, error) {
	if err != nil {
		return nil, err
	}
	return []types.ExamRAGChunkingSnapshot{}, nil
}

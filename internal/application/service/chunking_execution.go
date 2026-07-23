package service

import (
	"math"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/infrastructure/chunker"
	"github.com/Tencent/WeKnora/internal/types"
)

type ChunkingExecutionResult struct {
	Chunks       []types.ParsedChunk
	ParentChunks []types.ParsedParentChunk
	Diagnostics  ChunkingExecutionDiagnostics
}

type ChunkingExecutionDiagnostics struct {
	RequestedStrategy string                  `json:"requested_strategy"`
	ParentTier        string                  `json:"parent_tier,omitempty"`
	ChildTierCounts   map[string]int          `json:"child_tier_counts,omitempty"`
	Rejections        []chunker.TierRejection `json:"rejections,omitempty"`
	TextChunkCount    int                     `json:"text_chunk_count"`
	ParentChunkCount  int                     `json:"parent_chunk_count"`
	MinChars          int                     `json:"min_chars"`
	P50Chars          int                     `json:"p50_chars"`
	P90Chars          int                     `json:"p90_chars"`
	MaxChars          int                     `json:"max_chars"`
	TinyChunkCount    int                     `json:"tiny_chunk_count"`
	OversizeCount     int                     `json:"oversize_count"`
}

type chunkSizeSummary struct {
	Count         int
	Min           int
	P50           int
	P90           int
	Max           int
	TinyCount     int
	OversizeCount int
}

func executeChunking(markdown string, cc types.ChunkingConfig) ChunkingExecutionResult {
	base := buildSplitterConfigFromChunking(cc)
	if cc.EnableParentChild {
		return executeParentChildChunking(markdown, cc, base)
	}
	split, diag := chunker.SplitWithDiagnostics(markdown, base)
	parsed := parsedChunks(split, false)
	return ChunkingExecutionResult{
		Chunks:      parsed,
		Diagnostics: buildChunkingDiagnostics(cc, diag.SelectedTier, nil, diag.Rejected, parsed, 0, base.ChunkSize),
	}
}

func executeParentChildChunking(
	markdown string,
	cc types.ChunkingConfig,
	base chunker.SplitterConfig,
) ChunkingExecutionResult {
	parentCfg, childCfg := buildParentChildConfigs(cc, base)
	split, diag := chunker.SplitParentChildWithDiagnostics(markdown, parentCfg, childCfg)
	parsed := parsedChildChunks(split.Children)
	parents := parsedParentChunks(split.Parents)
	return ChunkingExecutionResult{
		Chunks:       parsed,
		ParentChunks: parents,
		Diagnostics: buildChunkingDiagnostics(
			cc, diag.Parent.SelectedTier, diag.ChildTierCounts,
			diag.Rejected, parsed, len(parents), childCfg.ChunkSize,
		),
	}
}

func buildChunkingDiagnostics(
	cc types.ChunkingConfig,
	parentTier chunker.StrategyTier,
	childTiers map[chunker.StrategyTier]int,
	rejections []chunker.TierRejection,
	chunks []types.ParsedChunk,
	parentCount, targetSize int,
) ChunkingExecutionDiagnostics {
	stats := summarizeChunkSizes(chunks, targetSize)
	return ChunkingExecutionDiagnostics{
		RequestedStrategy: requestedChunkingStrategy(cc.Strategy),
		ParentTier:        string(parentTier),
		ChildTierCounts:   stringifyTierCounts(childTiers),
		Rejections:        append([]chunker.TierRejection{}, rejections...),
		TextChunkCount:    stats.Count,
		ParentChunkCount:  parentCount,
		MinChars:          stats.Min,
		P50Chars:          stats.P50,
		P90Chars:          stats.P90,
		MaxChars:          stats.Max,
		TinyChunkCount:    stats.TinyCount,
		OversizeCount:     stats.OversizeCount,
	}
}

func summarizeChunkSizes(chunks []types.ParsedChunk, targetSize int) chunkSizeSummary {
	lengths := make([]int, 0, len(chunks))
	for _, parsed := range chunks {
		if strings.TrimSpace(parsed.Content) == "" {
			continue
		}
		lengths = append(lengths, len([]rune(parsed.Content)))
	}
	if len(lengths) == 0 {
		return chunkSizeSummary{}
	}
	sort.Ints(lengths)
	summary := chunkSizeSummary{
		Count: len(lengths), Min: lengths[0], Max: lengths[len(lengths)-1],
		P50: nearestRank(lengths, 0.50), P90: nearestRank(lengths, 0.90),
	}
	for _, length := range lengths {
		if targetSize > 0 && length < targetSize/4 {
			summary.TinyCount++
		}
		if targetSize > 0 && length > targetSize {
			summary.OversizeCount++
		}
	}
	return summary
}

func nearestRank(sorted []int, percentile float64) int {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func parsedChunks(chunks []chunker.Chunk, child bool) []types.ParsedChunk {
	parsed := make([]types.ParsedChunk, len(chunks))
	for index, item := range chunks {
		parentIndex := -1
		if child {
			parentIndex = 0
		}
		parsed[index] = types.ParsedChunk{
			Content: item.Content, ContextHeader: item.ContextHeader,
			Seq: item.Seq, Start: item.Start, End: item.End, ParentIndex: parentIndex,
		}
	}
	return parsed
}

func parsedChildChunks(chunks []chunker.ChildChunk) []types.ParsedChunk {
	parsed := make([]types.ParsedChunk, len(chunks))
	for index, item := range chunks {
		parsed[index] = types.ParsedChunk{
			Content: item.Content, ContextHeader: item.ContextHeader,
			Seq: item.Seq, Start: item.Start, End: item.End, ParentIndex: item.ParentIndex,
		}
	}
	return parsed
}

func parsedParentChunks(chunks []chunker.Chunk) []types.ParsedParentChunk {
	parsed := make([]types.ParsedParentChunk, len(chunks))
	for index, item := range chunks {
		parsed[index] = types.ParsedParentChunk{
			Content: item.Content, Seq: item.Seq, Start: item.Start, End: item.End,
		}
	}
	return parsed
}

func stringifyTierCounts(counts map[chunker.StrategyTier]int) map[string]int {
	result := make(map[string]int, len(counts))
	for tier, count := range counts {
		result[string(tier)] = count
	}
	return result
}

func requestedChunkingStrategy(strategy string) string {
	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		return chunker.StrategyLegacy
	}
	return strategy
}

func chunkingSpanInput(cc types.ChunkingConfig) types.JSONMap {
	base := buildSplitterConfigFromChunking(cc)
	parentSize, childSize := cc.ParentChunkSize, cc.ChildChunkSize
	if parentSize <= 0 {
		parentSize = 4096
	}
	if childSize <= 0 {
		childSize = 384
	}
	return types.JSONMap{
		"strategy":            requestedChunkingStrategy(base.Strategy),
		"chunk_size":          base.ChunkSize,
		"chunk_overlap":       base.ChunkOverlap,
		"separators":          append([]string{}, base.Separators...),
		"enable_parent_child": cc.EnableParentChild,
		"parent_chunk_size":   parentSize,
		"child_chunk_size":    childSize,
		"token_limit":         base.TokenLimit,
		"languages":           append([]string{}, base.Languages...),
	}
}

func chunkingSpanOutput(diag ChunkingExecutionDiagnostics) types.JSONMap {
	return types.JSONMap{
		"requested_strategy": diag.RequestedStrategy,
		"parent_tier":        diag.ParentTier,
		"child_tier_counts":  diag.ChildTierCounts,
		"rejections":         diag.Rejections,
		"text_chunk_count":   diag.TextChunkCount,
		"parent_chunk_count": diag.ParentChunkCount,
		"min_chars":          diag.MinChars,
		"p50_chars":          diag.P50Chars,
		"p90_chars":          diag.P90Chars,
		"max_chars":          diag.MaxChars,
		"tiny_chunk_count":   diag.TinyChunkCount,
		"oversize_count":     diag.OversizeCount,
	}
}

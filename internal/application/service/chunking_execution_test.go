package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/infrastructure/chunker"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExecuteChunking_FlatReportsSelectedTierAndSizes(t *testing.T) {
	t.Parallel()

	result := executeChunking(strings.Repeat("alpha beta gamma delta.\n\n", 20), types.ChunkingConfig{
		ChunkSize: 120, ChunkOverlap: 20,
		Separators: []string{"\n\n", " "}, Strategy: chunker.StrategyLegacy,
	})

	require.NotEmpty(t, result.Chunks)
	require.Empty(t, result.ParentChunks)
	require.Equal(t, chunker.StrategyLegacy, result.Diagnostics.RequestedStrategy)
	require.Equal(t, string(chunker.TierLegacy), result.Diagnostics.ParentTier)
	require.Equal(t, len(result.Chunks), result.Diagnostics.TextChunkCount)
	require.Zero(t, result.Diagnostics.ParentChunkCount)
	require.Positive(t, result.Diagnostics.MaxChars)
	require.GreaterOrEqual(t, result.Diagnostics.P90Chars, result.Diagnostics.P50Chars)
	for index, parsed := range result.Chunks {
		require.Equal(t, index, parsed.Seq)
		require.Equal(t, -1, parsed.ParentIndex)
	}
}

func TestExecuteChunking_ParentChildReportsTierDistribution(t *testing.T) {
	t.Parallel()

	result := executeChunking(strings.Repeat("A long sentence provides material for splitting. Another sentence follows.\n\n", 40), types.ChunkingConfig{
		ChunkSize: 120, ChunkOverlap: 20,
		Separators: []string{"\n\n", ". ", " "}, Strategy: chunker.StrategyLegacy,
		EnableParentChild: true, ParentChunkSize: 320, ChildChunkSize: 90,
	})

	require.NotEmpty(t, result.Chunks)
	require.NotEmpty(t, result.ParentChunks)
	require.Equal(t, string(chunker.TierLegacy), result.Diagnostics.ParentTier)
	require.Positive(t, result.Diagnostics.ChildTierCounts[string(chunker.TierLegacy)])
	require.Equal(t, len(result.Chunks), result.Diagnostics.TextChunkCount)
	require.Equal(t, len(result.ParentChunks), result.Diagnostics.ParentChunkCount)
	for _, parsed := range result.Chunks {
		if parsed.ParentIndex >= 0 {
			require.Less(t, parsed.ParentIndex, len(result.ParentChunks))
		}
	}
}

func TestSummarizeChunkSizes_UsesRuneNearestRankAndThresholds(t *testing.T) {
	t.Parallel()

	chunks := []types.ParsedChunk{
		{Content: strings.Repeat("甲", 10)},
		{Content: strings.Repeat("a", 25)},
		{Content: strings.Repeat("b", 50)},
		{Content: strings.Repeat("c", 75)},
		{Content: strings.Repeat("d", 120)},
		{Content: "   "},
	}

	stats := summarizeChunkSizes(chunks, 100)

	require.Equal(t, 5, stats.Count)
	require.Equal(t, 10, stats.Min)
	require.Equal(t, 50, stats.P50)
	require.Equal(t, 120, stats.P90)
	require.Equal(t, 120, stats.Max)
	require.Equal(t, 1, stats.TinyCount)
	require.Equal(t, 1, stats.OversizeCount)
}

func TestChunkingSpanPayloadsContainConfigAndDiagnosticsOnly(t *testing.T) {
	t.Parallel()

	input := chunkingSpanInput(types.ChunkingConfig{
		ChunkSize: 512, ChunkOverlap: 80, Strategy: chunker.StrategyAuto,
		EnableParentChild: true, ParentChunkSize: 4096, ChildChunkSize: 384,
		TokenLimit: 256, Languages: []string{chunker.LangEnglish},
	})
	output := chunkingSpanOutput(ChunkingExecutionDiagnostics{
		RequestedStrategy: chunker.StrategyAuto,
		ParentTier:        string(chunker.TierHeading),
		ChildTierCounts:   map[string]int{string(chunker.TierHeuristic): 2},
		TextChunkCount:    12, ParentChunkCount: 2,
		MinChars: 40, P50Chars: 280, P90Chars: 390, MaxChars: 480,
		TinyChunkCount: 1, OversizeCount: 2,
	})

	require.Equal(t, chunker.StrategyAuto, input["strategy"])
	require.Equal(t, 4096, input["parent_chunk_size"])
	require.Equal(t, 384, input["child_chunk_size"])
	require.Equal(t, 256, input["token_limit"])
	require.Equal(t, string(chunker.TierHeading), output["parent_tier"])
	require.Equal(t, 12, output["text_chunk_count"])
	require.NotContains(t, input, "content")
	require.NotContains(t, output, "content")
}

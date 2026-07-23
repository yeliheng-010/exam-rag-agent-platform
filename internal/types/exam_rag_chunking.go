package types

type ChunkingConfigSnapshot struct {
	Strategy          string   `json:"strategy"`
	ChunkSize         int      `json:"chunk_size"`
	ChunkOverlap      int      `json:"chunk_overlap"`
	EnableParentChild bool     `json:"enable_parent_child"`
	ParentChunkSize   int      `json:"parent_chunk_size"`
	ChildChunkSize    int      `json:"child_chunk_size"`
	TokenLimit        int      `json:"token_limit"`
	Languages         []string `json:"languages"`
}

type ExamRAGChunkingSnapshot struct {
	KnowledgeBaseID  string                 `json:"knowledge_base_id"`
	Config           ChunkingConfigSnapshot `json:"config"`
	KnowledgeCount   int                    `json:"knowledge_count"`
	TextChunkCount   int                    `json:"text_chunk_count"`
	ParentChunkCount int                    `json:"parent_chunk_count"`
	MinChars         int                    `json:"min_chars"`
	P50Chars         int                    `json:"p50_chars"`
	P90Chars         int                    `json:"p90_chars"`
	MaxChars         int                    `json:"max_chars"`
	TinyChunkRate    float64                `json:"tiny_chunk_rate"`
	OversizeRate     float64                `json:"oversize_rate"`
	ParentCoverage   float64                `json:"parent_coverage"`
	ActualTierCounts map[string]int         `json:"actual_tier_counts"`
	UnknownTierCount int                    `json:"unknown_tier_count"`
}

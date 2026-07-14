package types

type SearchTraceFusionMethod string

const (
	SearchTraceFusionNone        SearchTraceFusionMethod = "none"
	SearchTraceFusionRRF         SearchTraceFusionMethod = "rrf"
	SearchTraceFusionVectorOnly  SearchTraceFusionMethod = "vector_only"
	SearchTraceFusionKeywordOnly SearchTraceFusionMethod = "keyword_only"
)

type SearchTraceParameters struct {
	MatchCount       int     `json:"match_count"`
	VectorThreshold  float64 `json:"vector_threshold"`
	KeywordThreshold float64 `json:"keyword_threshold"`
	VectorEnabled    bool    `json:"vector_enabled"`
	KeywordEnabled   bool    `json:"keyword_enabled"`
	RRFK             int     `json:"rrf_k"`
	RRFVectorWeight  float64 `json:"rrf_vector_weight"`
	RRFKeywordWeight float64 `json:"rrf_keyword_weight"`
}

type SearchTraceCandidate struct {
	ChunkID     string  `json:"chunk_id"`
	Score       float64 `json:"score"`
	Rank        int     `json:"rank"`
	VectorRank  int     `json:"vector_rank,omitempty"`
	KeywordRank int     `json:"keyword_rank,omitempty"`
}

type SearchTrace struct {
	Query               string                  `json:"query"`
	KnowledgeBaseID     string                  `json:"knowledge_base_id"`
	KnowledgeBaseIDs    []string                `json:"knowledge_base_ids"`
	Parameters          SearchTraceParameters   `json:"parameters"`
	EmbeddingModelID    string                  `json:"embedding_model_id"`
	EmbeddingDimensions int                     `json:"embedding_dimensions"`
	VectorCandidates    []SearchTraceCandidate  `json:"vector_candidates"`
	KeywordCandidates   []SearchTraceCandidate  `json:"keyword_candidates"`
	FusionMethod        SearchTraceFusionMethod `json:"fusion_method"`
	FusionCandidates    []SearchTraceCandidate  `json:"fusion_candidates"`
	FinalChunks         []SearchTraceCandidate  `json:"final_chunks"`
	DurationMS          int64                   `json:"duration_ms"`
}

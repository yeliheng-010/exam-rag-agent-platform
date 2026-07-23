package searchutil

// ExamContextRankedItem keeps all aliases and attached context at one main-result rank.
type ExamContextRankedItem struct {
	Rank            int      `json:"rank"`
	KnowledgeBaseID string   `json:"knowledge_base_id"`
	LocalRank       int      `json:"local_rank"`
	ChunkIDs        []string `json:"chunk_ids"`
	Contents        []string `json:"contents"`
}

package types

type ExamRAGDiagnosticCase struct {
	Name             string   `json:"name"`
	Query            string   `json:"query"`
	RequiredPhrases  []string `json:"required_phrases"`
	ExpectedChunkIDs []string `json:"expected_chunk_ids"`
}

type RunExamRAGDiagnosticRequest struct {
	KnowledgeBaseIDs []string                `json:"knowledge_base_ids"`
	Cases            []ExamRAGDiagnosticCase `json:"cases"`
}

type ExamRAGDiagnosticResultItem struct {
	Name              string   `json:"name"`
	Query             string   `json:"query"`
	Passed            bool     `json:"passed"`
	RetrievalPassed   bool     `json:"retrieval_passed"`
	AnswerPassed      bool     `json:"answer_passed"`
	RetrievalScore    float64  `json:"retrieval_score"`
	AnswerScore       float64  `json:"answer_score"`
	MatchedChunkIDs   []string `json:"matched_chunk_ids"`
	MissingChunkIDs   []string `json:"missing_chunk_ids"`
	RetrievedChunkIDs []string `json:"retrieved_chunk_ids"`
	MatchedPhrases    []string `json:"matched_phrases"`
	MissingPhrases    []string `json:"missing_phrases"`
	ContextLabel      string   `json:"context_label"`
	SourceChunkIDs    []string `json:"source_chunk_ids"`
	Error             string   `json:"error"`
}

type ExamRAGDiagnosticSummary struct {
	Total            int                           `json:"total"`
	Passed           int                           `json:"passed"`
	RetrievalPassed  int                           `json:"retrieval_passed"`
	AnswerPassed     int                           `json:"answer_passed"`
	HitRate          float64                       `json:"hit_rate"`
	RetrievalHitRate float64                       `json:"retrieval_hit_rate"`
	AnswerHitRate    float64                       `json:"answer_hit_rate"`
	Results          []ExamRAGDiagnosticResultItem `json:"results"`
}

type ExamRAGDiagnosticResult struct {
	QuestionBank     *QuestionBank            `json:"question_bank"`
	KnowledgeBaseIDs []string                 `json:"knowledge_base_ids"`
	UsedDefaultCases bool                     `json:"used_default_cases"`
	Summary          ExamRAGDiagnosticSummary `json:"summary"`
	Cases            []ExamRAGDiagnosticCase  `json:"cases"`
}

package types

type ExamRAGContextSource string

const (
	ExamRAGContextSourceNone                    ExamRAGContextSource = "none"
	ExamRAGContextSourceStructuredQuestionGroup ExamRAGContextSource = "structured_question_group"
)

type ExamRAGDiagnosticCase struct {
	Name             string   `json:"name"`
	Query            string   `json:"query"`
	RequiredPhrases  []string `json:"required_phrases"`
	ExpectedChunkIDs []string `json:"expected_chunk_ids"`
}

type RunExamRAGDiagnosticRequest struct {
	KnowledgeBaseIDs []string                `json:"knowledge_base_ids"`
	Cases            []ExamRAGDiagnosticCase `json:"cases"`
	MatchCount       int                     `json:"match_count"`
	VectorThreshold  *float64                `json:"vector_threshold"`
	KeywordThreshold *float64                `json:"keyword_threshold"`
}

type ExamRAGDiagnosticPreparation struct {
	QuestionBank     *QuestionBank
	Request          RunExamRAGDiagnosticRequest
	UsedDefaultCases bool
}

type ExamRAGDiagnosticResultItem struct {
	Name              string               `json:"name"`
	Query             string               `json:"query"`
	Passed            bool                 `json:"passed"`
	RetrievalPassed   bool                 `json:"retrieval_passed"`
	AnswerPassed      bool                 `json:"answer_passed"`
	RetrievalScore    float64              `json:"retrieval_score"`
	AnswerScore       float64              `json:"answer_score"`
	MatchedChunkIDs   []string             `json:"matched_chunk_ids"`
	MissingChunkIDs   []string             `json:"missing_chunk_ids"`
	RetrievedChunkIDs []string             `json:"retrieved_chunk_ids"`
	MatchedPhrases    []string             `json:"matched_phrases"`
	MissingPhrases    []string             `json:"missing_phrases"`
	ContextLabel      string               `json:"context_label"`
	SourceChunkIDs    []string             `json:"source_chunk_ids"`
	CandidateChunkIDs []string             `json:"candidate_chunk_ids"`
	ContextSource     ExamRAGContextSource `json:"context_source"`
	GroupID           string               `json:"group_id"`
	SearchTraces      []*SearchTrace       `json:"search_traces"`
	DurationMS        int64                `json:"duration_ms"`
	ReciprocalRank    float64              `json:"reciprocal_rank"`
	Error             string               `json:"error"`
}

type ExamRAGDiagnosticSummary struct {
	Total                    int                           `json:"total"`
	Passed                   int                           `json:"passed"`
	RetrievalPassed          int                           `json:"retrieval_passed"`
	AnswerPassed             int                           `json:"answer_passed"`
	HitRate                  float64                       `json:"hit_rate"`
	RetrievalHitRate         float64                       `json:"retrieval_hit_rate"`
	AnswerHitRate            float64                       `json:"answer_hit_rate"`
	RecallAtK                float64                       `json:"recall_at_k"`
	MeanReciprocalRank       float64                       `json:"mean_reciprocal_rank"`
	RankedCaseCount          int                           `json:"ranked_case_count"`
	StructuredResolutionRate float64                       `json:"structured_resolution_rate"`
	StructuredResolved       int                           `json:"structured_resolved"`
	AverageDurationMS        float64                       `json:"average_duration_ms"`
	FailedCaseCount          int                           `json:"failed_case_count"`
	Results                  []ExamRAGDiagnosticResultItem `json:"results"`
}

type ExamRAGDiagnosticResult struct {
	QuestionBank     *QuestionBank            `json:"question_bank"`
	KnowledgeBaseIDs []string                 `json:"knowledge_base_ids"`
	UsedDefaultCases bool                     `json:"used_default_cases"`
	Summary          ExamRAGDiagnosticSummary `json:"summary"`
	Cases            []ExamRAGDiagnosticCase  `json:"cases"`
}

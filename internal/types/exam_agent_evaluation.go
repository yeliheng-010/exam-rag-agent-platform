package types

type ExamAgentExpectedToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ExamAgentEvaluationCase struct {
	Name                    string                      `json:"name"`
	Input                   string                      `json:"input"`
	ExpectedToolCalls       []ExamAgentExpectedToolCall `json:"expected_tool_calls,omitempty"`
	ExpectedEvidencePhrases []string                    `json:"expected_evidence_phrases,omitempty"`
	ExpectedCitations       []string                    `json:"expected_citations,omitempty"`
	ExpectedAnswerPhrases   []string                    `json:"expected_answer_phrases,omitempty"`
	GroundedPhrases         []string                    `json:"grounded_phrases,omitempty"`
}

type RunExamAgentEvaluationRequest struct {
	AgentID string                    `json:"agent_id"`
	Cases   []ExamAgentEvaluationCase `json:"cases"`
}

type ExamAgentSnapshot struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	ModelID        string            `json:"model_id"`
	AllowedTools   []string          `json:"allowed_tools"`
	KnowledgeBases []string          `json:"knowledge_bases"`
	Config         CustomAgentConfig `json:"config"`
}

type ExamAgentEvaluationRequestSnapshot struct {
	Agent ExamAgentSnapshot         `json:"agent"`
	Cases []ExamAgentEvaluationCase `json:"cases"`
}

type ExamAgentActualToolCall struct {
	Name          string                 `json:"name"`
	Arguments     map[string]interface{} `json:"arguments,omitempty"`
	Success       bool                   `json:"success"`
	DurationMS    int64                  `json:"duration_ms"`
	OutputExcerpt string                 `json:"output_excerpt,omitempty"`
	Error         string                 `json:"error,omitempty"`
}

type ExamAgentEvaluationCaseResult struct {
	Name               string                    `json:"name"`
	Input              string                    `json:"input"`
	Passed             bool                      `json:"passed"`
	ToolSequenceScore  float64                   `json:"tool_sequence_score"`
	ToolArgumentsScore float64                   `json:"tool_arguments_score"`
	EvidenceScore      float64                   `json:"evidence_score"`
	CitationScore      float64                   `json:"citation_score"`
	AnswerScore        float64                   `json:"answer_score"`
	GroundednessScore  float64                   `json:"groundedness_score"`
	DurationMS         int64                     `json:"duration_ms"`
	ActualToolCalls    []ExamAgentActualToolCall `json:"actual_tool_calls"`
	FinalAnswer        string                    `json:"final_answer"`
	AssertionFailures  []string                  `json:"assertion_failures,omitempty"`
	Error              string                    `json:"error,omitempty"`
}

type ExamAgentEvaluationSummary struct {
	Total             int     `json:"total"`
	Passed            int     `json:"passed"`
	Failed            int     `json:"failed"`
	PassRate          float64 `json:"pass_rate"`
	ToolSequenceRate  float64 `json:"tool_sequence_rate"`
	ToolArgumentsRate float64 `json:"tool_arguments_rate"`
	EvidenceRate      float64 `json:"evidence_rate"`
	CitationRate      float64 `json:"citation_rate"`
	AnswerRate        float64 `json:"answer_rate"`
	GroundednessRate  float64 `json:"groundedness_rate"`
	AverageDurationMS float64 `json:"average_duration_ms"`
}

type ExamAgentEvaluationResult struct {
	QuestionBankID string                          `json:"question_bank_id"`
	Agent          ExamAgentSnapshot               `json:"agent"`
	Summary        ExamAgentEvaluationSummary      `json:"summary"`
	Results        []ExamAgentEvaluationCaseResult `json:"results"`
}

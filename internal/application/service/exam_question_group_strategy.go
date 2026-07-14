package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

type QuestionGroupStrategyInput struct {
	Material              *types.ExamMaterial
	Task                  *types.ExamStructuringTask
	Chunks                []*types.Chunk
	ContextChunks         []*types.Chunk
	AnswerChunks          []*types.Chunk
	TargetQuestionNumbers []int
}

type QuestionGroupStrategyOptions struct {
	MaxGroups       int
	RequireMaterial bool
}

type ExamQuestionGroupExtractionStrategy interface {
	Code() string
	Match(material *types.ExamMaterial, task *types.ExamStructuringTask) bool
	BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error)
	Validate(candidate *types.ExamQuestionGroupDraftCandidate) error
}

type QuestionGroupStrategyRegistry struct {
	strategies []ExamQuestionGroupExtractionStrategy
	fallback   ExamQuestionGroupExtractionStrategy
}

func NewQuestionGroupStrategyRegistry() *QuestionGroupStrategyRegistry {
	return &QuestionGroupStrategyRegistry{
		strategies: []ExamQuestionGroupExtractionStrategy{
			newGaokaoEnglishReadingStrategy(),
			newGaokaoMathBasicStrategy(),
		},
		fallback: newGenericQuestionGroupStrategy(),
	}
}

func (r *QuestionGroupStrategyRegistry) Match(material *types.ExamMaterial, task *types.ExamStructuringTask) ExamQuestionGroupExtractionStrategy {
	if r == nil {
		return newGenericQuestionGroupStrategy()
	}
	for _, strategy := range r.strategies {
		if strategy != nil && strategy.Match(material, task) {
			return strategy
		}
	}
	if r.fallback != nil {
		return r.fallback
	}
	return newGenericQuestionGroupStrategy()
}

type questionGroupStrategy struct {
	code            string
	domainAliases   []string
	subjectAliases  []string
	groupType       string
	requireMaterial bool
	maxGroups       int
	instructions    []string
}

func newGaokaoEnglishReadingStrategy() ExamQuestionGroupExtractionStrategy {
	return &questionGroupStrategy{
		code:            "gaokao_english_reading_v1",
		domainAliases:   []string{"gaokao", "00000000-0000-0000-0000-000000000101"},
		subjectAliases:  []string{"english", "00000000-0000-0000-0000-000000001103"},
		groupType:       "reading_passage",
		requireMaterial: true,
		maxGroups:       2,
		instructions: []string{
			"Focus on the first complete English reading passage group.",
			"Use group_type=reading_passage and put the full passage in material_text.",
			"Preserve standalone passage headings, section headings, and table titles in material_text exactly as they appear.",
			"Do not omit introductory sentences before a table or timetable.",
			"Each question must keep its own stem, all options, answer, explanation, and source_chunk_ids.",
			"Do not leave explanation empty for objective questions; infer it from the passage and answer key when needed.",
			"Each explanation must include a short evidence quote or precise passage reference that supports the answer.",
			"Do not turn options into separate questions.",
		},
	}
}

func newGaokaoMathBasicStrategy() ExamQuestionGroupExtractionStrategy {
	return &questionGroupStrategy{
		code:            "gaokao_math_basic_v1",
		domainAliases:   []string{"gaokao", "00000000-0000-0000-0000-000000000101"},
		subjectAliases:  []string{"math", "00000000-0000-0000-0000-000000001102"},
		groupType:       "math_problem",
		requireMaterial: false,
		maxGroups:       6,
		instructions: []string{
			"Use group_type=math_problem for each independent math problem.",
			"Keep formulas in LaTeX when possible.",
			"Describe diagrams, tables, formulas, and image references in the assets array.",
			"Do not drop geometry figures or chart references even when storage_uri is unknown.",
			"Answer-key chunks are reference-only. Never create questions from answer-only text.",
			"Extract only question numbers whose stems appear in non-answer chunks.",
			"For choice questions, preserve every option across adjacent chunks; omit incomplete questions instead of inventing options.",
		},
	}
}

func newGenericQuestionGroupStrategy() ExamQuestionGroupExtractionStrategy {
	return &questionGroupStrategy{
		code:      "generic_question_group_v1",
		groupType: "single_question",
		maxGroups: 8,
		instructions: []string{
			"Use group_type=single_question when no shared passage or stem exists.",
			"Group sub-questions only when they share the same material_text.",
		},
	}
}

func (s *questionGroupStrategy) Code() string {
	if s == nil || s.code == "" {
		return "generic_question_group_v1"
	}
	return s.code
}

func (s *questionGroupStrategy) Match(material *types.ExamMaterial, _ *types.ExamStructuringTask) bool {
	if s == nil || len(s.domainAliases) == 0 {
		return false
	}
	if material == nil || !matchesAnyAlias(material.DomainID, s.domainAliases) {
		return false
	}
	return matchesAnyAlias(promptSubjectID(material), s.subjectAliases)
}

func (s *questionGroupStrategy) BuildPrompt(input QuestionGroupStrategyInput) (string, QuestionGroupStrategyOptions, error) {
	if s == nil {
		return "", QuestionGroupStrategyOptions{}, errors.New("question group strategy is nil")
	}
	options := defaultQuestionGroupStrategyOptions()
	options.MaxGroups = s.maxGroups
	options.RequireMaterial = s.requireMaterial
	prompt := buildQuestionGroupPrompt(input, s.groupType, s.Code(), s.instructions, options)
	return prompt, options, nil
}

func (s *questionGroupStrategy) Validate(candidate *types.ExamQuestionGroupDraftCandidate) error {
	requireMaterial := false
	if s != nil {
		requireMaterial = s.requireMaterial
	}
	return validateQuestionGroupCandidate(candidate, requireMaterial)
}

func defaultQuestionGroupStrategyOptions() QuestionGroupStrategyOptions {
	return QuestionGroupStrategyOptions{
		MaxGroups:       8,
		RequireMaterial: false,
	}
}

func buildQuestionGroupPrompt(input QuestionGroupStrategyInput, groupType string, strategyCode string, instructions []string, options QuestionGroupStrategyOptions) string {
	var builder strings.Builder
	builder.WriteString("Extract structured exam question groups from the paper chunks.\n")
	builder.WriteString("Return JSON only. Do not return Markdown or explanations outside JSON.\n")
	builder.WriteString(fmt.Sprintf("strategy_code: %s\npreferred_group_type: %s\nmax_groups: %d\n", strategyCode, groupType, options.MaxGroups))
	writeQuestionGroupPromptSchema(&builder)
	writeQuestionGroupPromptMaterial(&builder, input.Material, input.Task)
	writeQuestionGroupPromptInstructions(&builder, instructions)
	writeQuestionGroupPromptChunks(&builder, input)
	return builder.String()
}

func writeQuestionGroupPromptSchema(builder *strings.Builder) {
	builder.WriteString(`JSON schema:
{
  "question_groups": [{
    "group_no": "group number or label",
    "group_type": "reading_passage|math_problem|single_question|listening|writing|unknown",
    "title": "group title",
    "material_text": "shared passage/stem/context, empty only when none exists",
    "material_format": "plain_text|markdown|latex",
    "source_chunk_ids": ["chunk id"],
    "assets": [{"asset_type":"image|table|formula|audio","storage_uri":"","alt_text":"","source_chunk_id":"","bbox":{},"metadata":{},"sort_order":1}],
    "questions": [{
      "question_no": "question number",
      "question_type_code": "single_choice|multiple_choice|fill_blank|short_answer|math_problem|writing|unknown",
      "stem": "question stem",
      "options": [{"key":"A","content":"option text"}],
      "answer": {"value":"A"},
      "explanation": "answer explanation",
      "evidence": [{"chunk_id":"chunk id","quote":"short evidence"}],
      "metadata": {},
      "difficulty": "easy|medium|hard|unknown",
      "confidence": 0.8,
      "order_in_group": 1,
      "source_chunk_ids": ["chunk id"]
    }],
    "confidence": 0.8
  }]
}
`)
}

func writeQuestionGroupPromptMaterial(builder *strings.Builder, material *types.ExamMaterial, task *types.ExamStructuringTask) {
	if material != nil {
		builder.WriteString(fmt.Sprintf("\nMaterial title: %s\nExam domain: %s\nSubject: %s\n", material.Title, material.DomainID, promptSubjectID(material)))
	}
	if task != nil {
		builder.WriteString(fmt.Sprintf("Target question bank: %s\n", task.QuestionBankID))
	}
}

func writeQuestionGroupPromptInstructions(builder *strings.Builder, instructions []string) {
	if len(instructions) == 0 {
		return
	}
	builder.WriteString("\nSubject-specific rules:\n")
	for _, instruction := range instructions {
		builder.WriteString("- ")
		builder.WriteString(instruction)
		builder.WriteByte('\n')
	}
}

func writeQuestionGroupPromptChunks(builder *strings.Builder, input QuestionGroupStrategyInput) {
	if len(input.TargetQuestionNumbers) > 0 {
		writeRoleAwareQuestionGroupPromptChunks(builder, input)
		return
	}
	chunks := input.Chunks
	builder.WriteString("\nPaper chunks:\n")
	bodyChunks := selectQuestionExtractionBodyChunks(chunks)
	answerChunks := selectLikelyAnswerChunks(chunks)
	included := make(map[string]bool, len(chunks))
	for _, chunk := range bodyChunks {
		if chunk == nil {
			continue
		}
		writeQuestionPromptChunk(builder, chunk)
		included[questionPromptChunkKey(chunk)] = true
		if builder.Len() > questionExtractionBodyPromptLimit {
			builder.WriteString("\n[body chunks truncated by prompt length; answer chunks may follow]\n")
			break
		}
	}
	appendAnswerPromptChunks(builder, answerChunks, included)
}

func writeRoleAwareQuestionGroupPromptChunks(builder *strings.Builder, input QuestionGroupStrategyInput) {
	targets := make([]string, 0, len(input.TargetQuestionNumbers))
	for _, number := range input.TargetQuestionNumbers {
		targets = append(targets, fmt.Sprintf("%d", number))
	}
	builder.WriteString("\nTarget question numbers: " + strings.Join(targets, ", ") + "\n")
	builder.WriteString("Extract only these top-level question numbers. Use N(1), N(2), ... for subquestions.\n")
	writeQuestionGroupChunkRole(builder, "Core question chunks", input.Chunks)
	writeQuestionGroupChunkRole(builder, "Context-only chunks (do not create additional questions)", input.ContextChunks)
	writeQuestionGroupChunkRole(builder, "Answer-reference chunks (answers only; never create questions from them)", input.AnswerChunks)
}

func writeQuestionGroupChunkRole(builder *strings.Builder, label string, chunks []*types.Chunk) {
	if len(chunks) == 0 {
		return
	}
	builder.WriteString("\n" + label + ":\n")
	for _, chunk := range chunks {
		if chunk != nil {
			writeQuestionPromptChunk(builder, chunk)
		}
	}
}

func matchesAnyAlias(value string, aliases []string) bool {
	value = strings.TrimSpace(value)
	for _, alias := range aliases {
		if strings.EqualFold(value, strings.TrimSpace(alias)) {
			return true
		}
	}
	return false
}

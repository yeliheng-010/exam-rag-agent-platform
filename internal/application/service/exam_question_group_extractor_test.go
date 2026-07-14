package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

type captureQuestionGroupChat struct {
	opts *chat.ChatOptions
}

func (c *captureQuestionGroupChat) Chat(_ context.Context, _ []chat.Message, opts *chat.ChatOptions) (*types.ChatResponse, error) {
	c.opts = opts
	return &types.ChatResponse{Content: `{}`}, nil
}

func (c *captureQuestionGroupChat) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	return nil, nil
}

func (c *captureQuestionGroupChat) GetModelName() string { return "capture" }

func (c *captureQuestionGroupChat) GetModelID() string { return "capture" }

func TestQuestionGroupStrategyRegistrySelectsGaokaoEnglishReading(t *testing.T) {
	subjectID := "english"
	registry := NewQuestionGroupStrategyRegistry()
	strategy := registry.Match(
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
	)

	if strategy == nil {
		t.Fatal("expected a strategy")
	}
	if strategy.Code() != "gaokao_english_reading_v1" {
		t.Fatalf("expected gaokao English reading strategy, got %q", strategy.Code())
	}
}

func TestQuestionGroupStrategyRegistrySelectsGaokaoEnglishReadingBySeedID(t *testing.T) {
	subjectID := "00000000-0000-0000-0000-000000001103"
	registry := NewQuestionGroupStrategyRegistry()
	strategy := registry.Match(
		&types.ExamMaterial{DomainID: "00000000-0000-0000-0000-000000000101", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
	)

	if strategy == nil {
		t.Fatal("expected a strategy")
	}
	if strategy.Code() != "gaokao_english_reading_v1" {
		t.Fatalf("expected gaokao English reading strategy, got %q", strategy.Code())
	}
}

func TestParseQuestionGroupCandidatesKeepsReadingMaterialAndQuestions(t *testing.T) {
	raw := `{
	  "question_groups": [{
	    "group_no": "Reading A",
	    "group_type": "reading_passage",
	    "title": "Reading A",
	    "material_text": "Passage text",
	    "source_chunk_ids": ["chunk-1"],
	    "questions": [{
	      "question_no": "21",
	      "question_type_code": "single_choice",
	      "stem": "What is true?",
	      "options": [{"key":"A","content":"One"},{"key":"B","content":"Two"}],
	      "answer": {"value":"B"},
	      "explanation": "Because of sentence one.",
	      "confidence": 0.8,
	      "order_in_group": 1
	    }],
	    "confidence": 0.9
	  }]
	}`

	groups, err := parseQuestionGroupDraftCandidates(raw)

	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected one group, got %d", len(groups))
	}
	if groups[0].GroupType != "reading_passage" {
		t.Fatalf("expected reading_passage, got %q", groups[0].GroupType)
	}
	if groups[0].MaterialText != "Passage text" {
		t.Fatalf("expected material text to be kept, got %q", groups[0].MaterialText)
	}
	if len(groups[0].Questions) != 1 {
		t.Fatalf("expected one question, got %d", len(groups[0].Questions))
	}
	if groups[0].Questions[0].QuestionNo != "21" {
		t.Fatalf("expected question no 21, got %q", groups[0].Questions[0].QuestionNo)
	}
}

func TestParseQuestionGroupCandidatesKeepsIncompleteChoiceForQualityReview(t *testing.T) {
	raw := `{
	  "question_groups": [{
	    "group_no": "5",
	    "group_type": "math_problem",
	    "questions": [{
	      "question_no": "5",
	      "question_type_code": "single_choice",
	      "stem": "Question stem with an image reference",
	      "options": [{"key":"A","content":"only recovered option"}],
	      "answer": {"value":"A"}
	    }]
	  }]
	}`

	groups, err := parseQuestionGroupDraftCandidates(raw)
	if err != nil {
		t.Fatalf("expected incomplete choice to reach quality review, got %v", err)
	}
	if len(groups) != 1 || len(groups[0].Questions[0].Options) != 1 {
		t.Fatalf("unexpected groups: %#v", groups)
	}
}

func TestParseQuestionGroupCandidatesReportsRejectedCandidateReasons(t *testing.T) {
	raw := `{"question_groups":[{"group_no":"9","group_type":"math_problem","questions":[{"question_no":"9","stem":""}]}]}`

	_, err := parseQuestionGroupDraftCandidates(raw)

	if !errors.Is(err, errNoValidQuestionGroupDrafts) {
		t.Fatalf("expected no-valid-candidate sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "candidate 1: question stem is required") {
		t.Fatalf("expected rejected candidate reason, got %v", err)
	}
}

func TestBuildGaokaoMathPromptMentionsFormulaAndAssets(t *testing.T) {
	subjectID := "math"
	registry := NewQuestionGroupStrategyRegistry()
	strategy := registry.Match(
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
	)

	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Material: &types.ExamMaterial{Title: "2026 Gaokao Math", DomainID: "gaokao", SubjectID: &subjectID},
		Task:     &types.ExamStructuringTask{QuestionBankID: "bank-1"},
		Chunks:   []*types.Chunk{{ID: "chunk-1", ChunkIndex: 1, Content: "17. f(x)=x^2"}},
	})

	if err != nil {
		t.Fatalf("expected prompt build success, got %v", err)
	}
	if !strings.Contains(prompt, "LaTeX") {
		t.Fatalf("expected prompt to mention LaTeX, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "assets") {
		t.Fatalf("expected prompt to mention assets, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "math_problem") {
		t.Fatalf("expected prompt to mention math_problem, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Answer-key chunks are reference-only") {
		t.Fatalf("expected prompt to prevent answer-only fake questions, got:\n%s", prompt)
	}
}

func TestBuildGaokaoEnglishReadingPromptRequiresEvidenceBasedExplanation(t *testing.T) {
	subjectID := "english"
	strategy := NewQuestionGroupStrategyRegistry().Match(
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
	)

	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Material: &types.ExamMaterial{Title: "2026 Gaokao English", DomainID: "gaokao", SubjectID: &subjectID},
		Task:     &types.ExamStructuringTask{QuestionBankID: "bank-1"},
		Chunks: []*types.Chunk{
			{ID: "chunk-a", ChunkIndex: 1, Content: "A. Reading passage\n21. Which team? A. Cowboys B. Rams\nAnswers: 21 B"},
		},
	})

	if err != nil {
		t.Fatalf("expected prompt build success, got %v", err)
	}
	if !strings.Contains(prompt, "Do not leave explanation empty") {
		t.Fatalf("expected prompt to require non-empty explanations, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "quote") {
		t.Fatalf("expected prompt to require evidence quotes, got:\n%s", prompt)
	}
}

func TestQuestionGroupExtractionChatOptionsEnableJSONMode(t *testing.T) {
	model := &captureQuestionGroupChat{}
	if _, err := callQuestionGroupExtractionModel(context.Background(), model, "prompt", 4096); err != nil {
		t.Fatalf("expected model call to succeed, got %v", err)
	}

	if model.opts == nil {
		t.Fatal("expected chat options to be captured")
	}
	if string(model.opts.Format) != `"json"` {
		t.Fatalf("expected Ollama/OpenAI JSON mode, got %q", string(model.opts.Format))
	}
	if model.opts.MaxTokens != 4096 {
		t.Fatalf("expected max tokens 4096, got %d", model.opts.MaxTokens)
	}
}

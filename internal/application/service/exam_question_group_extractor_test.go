package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

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
}

package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestEnrichQuestionGroupDraftCandidateMaterialRestoresMissingReadingHeadings(t *testing.T) {
	candidate := &types.ExamQuestionGroupDraftCandidate{
		GroupNo:        "A",
		GroupType:      "reading_passage",
		Title:          "A. SoFi Stadium Events This Month",
		MaterialText:   "SoFi Stadium Events This Month\n\nLos Angeles Rams v Dallas Cowboys | Saturday, August 9 4:00 PM\n\nNearby Hotels\nNearby hotel paragraph.",
		SourceChunkIDs: []string{"chunk-title", "chunk-table"},
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{
				QuestionNo:       "21",
				SourceChunkIDs:   []string{"chunk-q21"},
				QuestionTypeCode: "single_choice",
			},
		},
	}
	chunks := []*types.Chunk{
		{
			ID:         "chunk-title",
			ChunkIndex: 10,
			Content: strings.Join([]string{
				"**A** **SoFi Stadium Events This Month**",
			}, "\n"),
		},
		{
			ID:         "chunk-missing-heading",
			ChunkIndex: 11,
			Content: strings.Join([]string{
				"**A**",
				"**SoFi Stadium Events This Month**",
				"SoFi Stadium is the go-to destination in the heart of Los Angeles for sports fans.",
				"Upcoming Football Events",
				"Los Angeles Rams v Dallas Cowboys | Saturday, August 9 4:00 PM",
			}, "\n"),
		},
		{
			ID:         "chunk-table",
			ChunkIndex: 12,
			Content: strings.Join([]string{
				"Los Angeles Rams v Dallas Cowboys | Saturday, August 9 4:00 PM",
				"Nearby Hotels",
				"Nearby hotel paragraph.",
			}, "\n"),
		},
		{
			ID:         "chunk-q21",
			ChunkIndex: 11,
			Content:    "21. Which team will play the most games? A. Dallas Cowboys. B. Los Angeles Rams.",
		},
	}

	enrichQuestionGroupDraftCandidateMaterial(candidate, chunks)

	if !strings.Contains(candidate.MaterialText, "Upcoming Football Events") {
		t.Fatalf("material text should restore missing table heading:\n%s", candidate.MaterialText)
	}
	if !strings.Contains(candidate.MaterialText, "SoFi Stadium is the go-to destination") {
		t.Fatalf("material text should restore missing introductory sentence:\n%s", candidate.MaterialText)
	}
	if strings.Contains(candidate.MaterialText, "21. Which team") {
		t.Fatalf("material text should not include question text:\n%s", candidate.MaterialText)
	}
	if strings.Index(candidate.MaterialText, "Upcoming Football Events") > strings.Index(candidate.MaterialText, "Los Angeles Rams v Dallas Cowboys") {
		t.Fatalf("restored heading should be placed before table rows:\n%s", candidate.MaterialText)
	}
}

func TestBuildGaokaoEnglishReadingPromptPreservesHeadings(t *testing.T) {
	subjectID := "english"
	strategy := NewQuestionGroupStrategyRegistry().Match(
		&types.ExamMaterial{DomainID: "gaokao", SubjectID: &subjectID},
		&types.ExamStructuringTask{},
	)

	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Material: &types.ExamMaterial{Title: "2026 Gaokao English", DomainID: "gaokao", SubjectID: &subjectID},
		Task:     &types.ExamStructuringTask{QuestionBankID: "bank-1"},
		Chunks: []*types.Chunk{
			{ID: "chunk-a", ChunkIndex: 1, Content: "A. Reading passage\nUpcoming Football Events\n21. Which team? A. Cowboys B. Rams\nAnswers: 21 B"},
		},
	})

	if err != nil {
		t.Fatalf("expected prompt build success, got %v", err)
	}
	if !strings.Contains(prompt, "Preserve standalone passage headings") {
		t.Fatalf("expected prompt to preserve headings, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Do not omit introductory sentences") {
		t.Fatalf("expected prompt to preserve introductory sentences, got:\n%s", prompt)
	}
}

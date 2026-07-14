package searchutil

import (
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestBuildStructuredExamQuestionContextBundleReadingGroup(t *testing.T) {
	t.Parallel()

	bundle := BuildStructuredExamQuestionContextBundle("first reading question answer and explanation", newStructuredReadingGroup())
	if bundle == nil {
		t.Fatal("expected structured context bundle")
	}
	if bundle.Label != "reading_passage:SoFi Stadium Events" {
		t.Fatalf("label = %q", bundle.Label)
	}
	if got := strings.Join(bundle.SourceChunkIDs, ","); got != "chunk-a1,chunk-a2,chunk-21" {
		t.Fatalf("source chunks = %q", got)
	}
	if !strings.Contains(bundle.Content, "[Exam Structured Context]") {
		t.Fatalf("missing structured context header:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "SoFi Stadium is the go-to destination.") {
		t.Fatalf("missing material text:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "21. Which team will play the most games?") {
		t.Fatalf("missing question stem:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "B. Los Angeles Rams") {
		t.Fatalf("missing option:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "Answer: B") {
		t.Fatalf("missing answer:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "Explanation: The passage lists the Rams for multiple events.") {
		t.Fatalf("missing explanation:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "chunk-21") {
		t.Fatalf("missing chunk ref:\n%s", bundle.Content)
	}
}

func TestBuildStructuredExamQuestionContextBundleTrimsLongMaterial(t *testing.T) {
	t.Parallel()

	detail := newStructuredReadingGroup()
	detail.Group.MaterialText = strings.Repeat("long material ", 1000)
	text := BuildStructuredExamQuestionContextText(detail, 240)
	if !strings.Contains(text, "[truncated]") {
		t.Fatalf("expected truncated marker:\n%s", text)
	}
}

func TestBuildStructuredExamQuestionContextBundleBackfillsMissingExplanation(t *testing.T) {
	t.Parallel()

	detail := newStructuredReadingGroup()
	detail.Questions[0].Explanations = nil
	bundle := BuildStructuredExamQuestionContextBundle("answer and explanation", detail)

	if bundle == nil {
		t.Fatal("expected structured context bundle")
	}
	if !strings.Contains(bundle.Content, "Correct answer: B") {
		t.Fatalf("missing fallback answer explanation:\n%s", bundle.Content)
	}
	if !strings.Contains(bundle.Content, "Los Angeles Rams") {
		t.Fatalf("fallback explanation should include the correct option or evidence:\n%s", bundle.Content)
	}
}

func newStructuredReadingGroup() *types.QuestionGroupDetail {
	now := time.Date(2026, 7, 8, 15, 4, 5, 0, time.UTC)
	groupID := "group-reading-a"
	questionID := "question-21"
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:             groupID,
			TenantID:       10000,
			QuestionBankID: "bank-english",
			GroupType:      "reading_passage",
			Title:          "SoFi Stadium Events",
			MaterialText:   "SoFi Stadium is the go-to destination. Upcoming Football Events include Los Angeles Rams games.",
			SourceChunkIDs: types.JSON(`["chunk-a1","chunk-a2"]`),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Questions: []*types.QuestionDetail{
			{
				Question: &types.Question{
					ID:           questionID,
					TenantID:     10000,
					GroupID:      &groupID,
					QuestionNo:   "21",
					OrderInGroup: 1,
					Stem:         "Which team will play the most games?",
					Status:       "active",
					CreatedAt:    now,
					UpdatedAt:    now,
				},
				Options: []*types.QuestionOption{
					{ID: "option-21-a", QuestionID: questionID, OptionKey: "A", Content: "Dallas Cowboys", SortOrder: 1},
					{ID: "option-21-b", QuestionID: questionID, OptionKey: "B", Content: "Los Angeles Rams", SortOrder: 2},
				},
				Answers: []*types.QuestionAnswer{
					{ID: "answer-21", QuestionID: questionID, AnswerText: "B", IsCorrect: true, CreatedAt: now},
				},
				Explanations: []*types.QuestionExplanation{
					{ID: "explain-21", QuestionID: questionID, ExplanationText: "The passage lists the Rams for multiple events.", SourceType: "model", CreatedAt: now},
				},
				ChunkRefs: []*types.QuestionChunkRef{
					{QuestionID: questionID, ChunkID: "chunk-21", RefType: "evidence", Confidence: 0.9, CreatedAt: now},
				},
			},
		},
	}
}

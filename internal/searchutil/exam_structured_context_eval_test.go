package searchutil

import (
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestEvaluateStructuredExamQuestionContextGaokaoEnglishReadingCases(t *testing.T) {
	t.Parallel()

	detail := newGaokaoEnglishReadingEvalGroup()
	cases := []ExamContextEvalCase{
		{
			Name:  "passage_text",
			Query: "first reading passage original text",
			RequiredPhrases: []string{
				"SoFi Stadium is the go-to destination for sports fans.",
				"Upcoming Football Events",
			},
		},
		{
			Name:  "answer_and_explanation",
			Query: "first reading question 21 answer and explanation",
			RequiredPhrases: []string{
				"21. Which team will play the most games?",
				"B. Los Angeles Rams",
				"The schedule names the Rams in more listed events.",
			},
		},
		{
			Name:  "questions_and_options",
			Query: "first reading questions and options",
			RequiredPhrases: []string{
				"22. What can visitors do before buying tickets?",
				"A. Check event dates online",
				"D. Book a private tour only",
			},
		},
		{
			Name:  "all_answers",
			Query: "reading A all answers",
			RequiredPhrases: []string{
				"chunk-21",
				"chunk-22",
				"The passage says visitors can check event dates online.",
			},
		},
	}

	summary := EvaluateStructuredExamQuestionContext(detail, cases)
	if summary.Total != len(cases) {
		t.Fatalf("total = %d", summary.Total)
	}
	if summary.Passed != len(cases) {
		t.Fatalf("passed = %d, results = %#v", summary.Passed, summary.Results)
	}
	if summary.HitRate != 1 {
		t.Fatalf("hit rate = %.2f", summary.HitRate)
	}
	for _, result := range summary.Results {
		if !result.Passed {
			t.Fatalf("case %s failed, missing=%v", result.Name, result.MissingPhrases)
		}
	}
}

func TestEvaluateStructuredExamQuestionContextReportsMissingPhrases(t *testing.T) {
	t.Parallel()

	summary := EvaluateStructuredExamQuestionContext(newGaokaoEnglishReadingEvalGroup(), []ExamContextEvalCase{
		{
			Name:            "missing",
			Query:           "missing evidence",
			RequiredPhrases: []string{"SoFi Stadium", "not in this fixture"},
		},
	})

	if summary.Total != 1 || summary.Passed != 0 {
		t.Fatalf("summary = %#v", summary)
	}
	if summary.HitRate != 0 {
		t.Fatalf("hit rate = %.2f", summary.HitRate)
	}
	if len(summary.Results) != 1 {
		t.Fatalf("results = %#v", summary.Results)
	}
	result := summary.Results[0]
	if result.Score != 0.5 {
		t.Fatalf("score = %.2f", result.Score)
	}
	if got := strings.Join(result.MissingPhrases, ","); got != "not in this fixture" {
		t.Fatalf("missing = %q", got)
	}
}

func TestEvaluateStructuredExamQuestionContextGaokaoMathFigureCase(t *testing.T) {
	detail := newGaokaoMathFigureEvalGroup()
	cases := []ExamContextEvalCase{{
		Name:  "math_figure_and_subquestions",
		Query: "第15题的图形、两个小问、答案和解析是什么？",
		RequiredPhrases: []string{
			"[Assets]",
			"立体几何示意图",
			"15(1). 证明平面 ABC 垂直于平面 BCD",
			"15(2). 求点 A 到平面 BCD 的距离",
			"Answer: 1",
			"Explanation: 点到平面的距离为 1",
		},
	}}

	summary := EvaluateStructuredExamQuestionContext(detail, cases)

	if summary.Passed != 1 {
		t.Fatalf("math eval results = %#v", summary.Results)
	}
}

func newGaokaoMathFigureEvalGroup() *types.QuestionGroupDetail {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	groupID := "group-gaokao-math-15"
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID: groupID, TenantID: 10000, QuestionBankID: "bank-gaokao-math",
			GroupType: "math_problem", Title: "第 15 题", SourceChunkIDs: types.JSON(`["chunk-15"]`),
			CreatedAt: now, UpdatedAt: now,
		},
		Assets: []*types.QuestionGroupAsset{{
			ID: "asset-15", TenantID: 10000, GroupID: groupID, AssetType: "image",
			StorageURI: "local://math/figure-15.png", AltText: "立体几何示意图", SourceChunkID: "chunk-15", SortOrder: 1,
		}},
		Questions: []*types.QuestionDetail{
			newGaokaoMathEvalQuestion(groupID, "q-15-1", "15(1)", 1, "证明平面 ABC 垂直于平面 BCD", "见证明", "由线面垂直判定定理可证", now),
			newGaokaoMathEvalQuestion(groupID, "q-15-2", "15(2)", 2, "求点 A 到平面 BCD 的距离", "1", "点到平面的距离为 1", now),
		},
	}
}

func newGaokaoMathEvalQuestion(groupID, id, no string, order int, stem, answer, explanation string, now time.Time) *types.QuestionDetail {
	return &types.QuestionDetail{
		Question:     &types.Question{ID: id, TenantID: 10000, GroupID: &groupID, QuestionNo: no, OrderInGroup: order, Stem: stem, Status: "active", CreatedAt: now, UpdatedAt: now},
		Answers:      []*types.QuestionAnswer{{ID: id + "-answer", QuestionID: id, AnswerText: answer, IsCorrect: true, CreatedAt: now}},
		Explanations: []*types.QuestionExplanation{{ID: id + "-explanation", QuestionID: id, ExplanationText: explanation, SourceType: "eval_fixture", CreatedAt: now}},
	}
}

func newGaokaoEnglishReadingEvalGroup() *types.QuestionGroupDetail {
	now := time.Date(2026, 7, 8, 16, 20, 0, 0, time.UTC)
	groupID := "group-gaokao-reading-a"
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{
			ID:             groupID,
			TenantID:       10000,
			QuestionBankID: "bank-gaokao-english",
			GroupType:      "reading_passage",
			Title:          "Reading A - SoFi Stadium Events",
			MaterialText:   "SoFi Stadium is the go-to destination for sports fans.\nUpcoming Football Events include Rams games, concerts, and family experiences. Visitors can check event dates online before buying tickets.",
			SourceChunkIDs: types.JSON(`["chunk-a1","chunk-a2"]`),
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		Questions: []*types.QuestionDetail{
			newGaokaoEnglishEvalQuestion(groupID, "question-21", "21", 1,
				"Which team will play the most games?",
				[]string{"Dallas Cowboys", "Los Angeles Rams", "New York Jets", "Miami Dolphins"},
				"B",
				"The schedule names the Rams in more listed events.",
				"chunk-21",
				now,
			),
			newGaokaoEnglishEvalQuestion(groupID, "question-22", "22", 2,
				"What can visitors do before buying tickets?",
				[]string{"Check event dates online", "Meet every player", "Enter without a ticket", "Book a private tour only"},
				"A",
				"The passage says visitors can check event dates online.",
				"chunk-22",
				now,
			),
		},
	}
}

func newGaokaoEnglishEvalQuestion(
	groupID string,
	questionID string,
	no string,
	order int,
	stem string,
	options []string,
	answer string,
	explanation string,
	chunkID string,
	now time.Time,
) *types.QuestionDetail {
	optionRows := make([]*types.QuestionOption, 0, len(options))
	keys := []string{"A", "B", "C", "D"}
	for i, option := range options {
		optionRows = append(optionRows, &types.QuestionOption{
			ID:         questionID + "-option-" + strings.ToLower(keys[i]),
			QuestionID: questionID,
			OptionKey:  keys[i],
			Content:    option,
			SortOrder:  i + 1,
		})
	}
	return &types.QuestionDetail{
		Question: &types.Question{
			ID:           questionID,
			TenantID:     10000,
			GroupID:      &groupID,
			QuestionNo:   no,
			OrderInGroup: order,
			Stem:         stem,
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		Options: optionRows,
		Answers: []*types.QuestionAnswer{
			{ID: questionID + "-answer", QuestionID: questionID, AnswerText: answer, IsCorrect: true, CreatedAt: now},
		},
		Explanations: []*types.QuestionExplanation{
			{ID: questionID + "-explanation", QuestionID: questionID, ExplanationText: explanation, SourceType: "eval_fixture", CreatedAt: now},
		},
		ChunkRefs: []*types.QuestionChunkRef{
			{QuestionID: questionID, ChunkID: chunkID, RefType: "evidence", Confidence: 0.9, CreatedAt: now},
		},
	}
}

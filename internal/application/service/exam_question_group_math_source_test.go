package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestEnrichMathBatchCandidatesBuildsMissingChoiceFromSource(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks: []*types.Chunk{{
			ID: "body-9", ChunkIndex: 9,
			Content: "9. 在正三棱柱中，D为中点，则（ ）\nA. A. 选项甲 B. 选项乙\nC. 选项丙 D. 选项丁",
		}},
		AnswerChunks: []*types.Chunk{{
			ID: "answer-9", Content: "【9题答案】\n【答案】BD",
		}},
		TargetQuestionNumbers: []int{9},
	}

	got := enrichMathBatchCandidates(nil, batch)

	if len(got) != 1 || len(got[0].Questions) != 1 {
		t.Fatalf("expected one deterministic source candidate, got %#v", got)
	}
	question := got[0].Questions[0]
	if question.QuestionNo != "9" || question.QuestionTypeCode != "multiple_choice" {
		t.Fatalf("unexpected question identity: %#v", question)
	}
	if len(question.Options) != 4 || draftAnswerText(question.Answer) != "BD" {
		t.Fatalf("expected four options and answer BD, got %#v", question)
	}
}

func TestEnrichMathBatchCandidatesAddsMissingSubjectiveSubquestion(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks: []*types.Chunk{{
			ID: "body-16", Content: "16. 已知数列。\n（1）证明该数列为等差数列；\n（2）求函数的最大值。",
		}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-16", Content: "【16题答案】【答案】（1）证明见解析；（2）最大值为1。"}},
		TargetQuestionNumbers: []int{16},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{{
		GroupNo: "16", Title: "Question 16",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{
			QuestionNo: "16(1)", Stem: "证明该数列为等差数列", Answer: types.JSONMap{},
		}},
	}}

	got := enrichMathBatchCandidates(candidates, batch)

	if len(got) != 1 || len(got[0].Questions) != 2 {
		t.Fatalf("expected source to add missing subquestion, got %#v", got)
	}
	if got[0].Questions[1].QuestionNo != "16(2)" || !hasMeaningfulAnswer(got[0].Questions[1].Answer) {
		t.Fatalf("unexpected recovered subquestion: %#v", got[0].Questions[1])
	}
}

func TestEnrichMathBatchCandidatesEnrichesEveryCandidateForSameProblem(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks:            []*types.Chunk{{ID: "body-17", Content: "17. 如图。\n（1）证明垂直；（2）求角的余弦值。"}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-17", Content: "【17题答案】【答案】（1）证明见解析；（2）余弦值为1/2。"}},
		TargetQuestionNumbers: []int{17},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{
		{GroupNo: "17", Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{QuestionNo: "17(1)", Stem: "证明垂直"}}},
		{GroupNo: "17", Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{QuestionNo: "17(2)", Stem: "求角"}}},
		{GroupNo: "17", Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{QuestionNo: "17(3)", Stem: "模型臆造小问"}}},
	}

	got := enrichMathBatchCandidates(candidates, batch)

	for _, candidate := range got {
		for _, question := range candidate.Questions {
			if question.QuestionNo == "17(3)" {
				t.Fatalf("source whitelist kept hallucinated subquestion: %#v", got)
			}
			if !hasMeaningfulAnswer(question.Answer) {
				t.Fatalf("candidate was not enriched: %#v", got)
			}
		}
	}
}

func TestEnrichMathBatchCandidatesCorrectsObjectiveAnswerFromSource(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks: []*types.Chunk{{
			ID: "body-10", Content: "10. 已知抛物线，则（ ）\nA. 甲 B. 乙 C. 丙 D. 丁",
		}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-10", Content: "【10题答案】【答案】ACD"}},
		TargetQuestionNumbers: []int{10},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{{
		GroupNo: "10", Title: "Question 10",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{
			QuestionNo: "10", QuestionTypeCode: "single_choice", Stem: "short", Answer: types.JSONMap{"value": "A"},
			Options: []types.ExamQuestionDraftOption{{Key: "A"}, {Key: "B"}, {Key: "C"}, {Key: "D"}},
		}},
	}}

	got := enrichMathBatchCandidates(candidates, batch)

	question := got[0].Questions[0]
	if question.QuestionTypeCode != "multiple_choice" || draftAnswerText(question.Answer) != "ACD" {
		t.Fatalf("expected source type and answer to win, got %#v", question)
	}
	if len(question.Options) != 4 {
		t.Fatalf("expected source options to fill the question, got %#v", question.Options)
	}
	for _, option := range question.Options {
		if option.Content == "" {
			t.Fatalf("expected source to replace equally-sized empty options, got %#v", question.Options)
		}
	}
}

func TestEnrichMathBatchCandidatesAttachesOfficialAnswerToSubquestions(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks:            []*types.Chunk{{ID: "body-15", Content: "15. 解答题\n（1）求概率；\n（2）完成检验。"}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-15", Content: "【15题答案】【答案】（1）0.9；（2）有关。"}},
		TargetQuestionNumbers: []int{15},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{{
		GroupNo: "15", Title: "Question 15",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{QuestionNo: "15(1)", Stem: "求概率", Answer: types.JSONMap{}},
			{QuestionNo: "15(2)", Stem: "完成检验", Answer: types.JSONMap{}},
		},
	}}

	got := enrichMathBatchCandidates(candidates, batch)

	for _, question := range got[0].Questions {
		if !hasMeaningfulAnswer(question.Answer) || question.Explanation == "" {
			t.Fatalf("expected official answer reference on subquestion, got %#v", question)
		}
	}
}

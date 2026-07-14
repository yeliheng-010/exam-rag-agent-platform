package service

import (
	"slices"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestBuildMathQuestionGroupExtractionBatchesMatchesAnswerRanges(t *testing.T) {
	chunks := []*types.Chunk{
		{ID: "body-1", ChunkIndex: 0, Content: "选出每小题答案后填涂答题卡。\n1. Q1\n2. Q2"},
		{ID: "body-2", ChunkIndex: 1, Content: "3. Q3\n4. Q4"},
		{ID: "body-3", ChunkIndex: 2, Content: "5. Q5\n6. Q6"},
		{ID: "body-4", ChunkIndex: 3, Content: "7. Q7\n8. Q8"},
		{ID: "answer-1", ChunkIndex: 4, Content: "【1 题答案】A\n【6 题答案】B"},
		{ID: "answer-2", ChunkIndex: 5, Content: "【7 题答案】C\n【8 题答案】D"},
	}

	batches := buildQuestionGroupExtractionBatches("gaokao_math_basic_v1", chunks)

	if len(batches) != 3 {
		t.Fatalf("expected three question-boundary batches, got %d", len(batches))
	}
	if !slices.Equal(batches[0].TargetQuestionNumbers, []int{1, 2, 3}) {
		t.Fatalf("unexpected first targets: %v", batches[0].TargetQuestionNumbers)
	}
	firstAnswers := questionGroupBatchContents(batches[0].AnswerChunks)
	if !strings.Contains(firstAnswers, "【1 题答案】") || strings.Contains(firstAnswers, "【6 题答案】") {
		t.Fatalf("expected first batch to carry only projected target answers, got %s", firstAnswers)
	}
	last := batches[len(batches)-1]
	if !slices.Equal(last.TargetQuestionNumbers, []int{7, 8}) {
		t.Fatalf("unexpected final targets: %v", last.TargetQuestionNumbers)
	}
	lastAnswers := questionGroupBatchContents(last.AnswerChunks)
	if !strings.Contains(lastAnswers, "【7 题答案】") || strings.Contains(lastAnswers, "【6 题答案】") {
		t.Fatalf("expected final batch to carry only answers 7-8, got %s", lastAnswers)
	}
}

func TestBuildMathQuestionGroupPromptLabelsChunkRolesAndTargets(t *testing.T) {
	strategy := newGaokaoMathBasicStrategy()
	prompt, _, err := strategy.BuildPrompt(QuestionGroupStrategyInput{
		Chunks:                []*types.Chunk{{ID: "body-1", Content: "1. real stem"}},
		ContextChunks:         []*types.Chunk{{ID: "context-1", Content: "previous option"}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-1", Content: "【1题答案】【答案】C"}},
		TargetQuestionNumbers: []int{1},
	})

	if err != nil {
		t.Fatalf("BuildPrompt returned error: %v", err)
	}
	for _, expected := range []string{"Target question numbers: 1", "Core question chunks", "Context-only chunks", "Answer-reference chunks"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", expected, prompt)
		}
	}
}

func TestConstrainMathBatchCandidatesRejectsAnswerOnlyAndSplitsProblems(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks:            []*types.Chunk{{ID: "body-1", Content: "1. real stem\n2. another stem"}},
		AnswerChunks:          []*types.Chunk{{ID: "answer-1", Content: "【1题答案】【答案】C"}},
		TargetQuestionNumbers: []int{1, 2},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{{
		GroupNo: "1-2", Title: "Multiple Choice Questions",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{QuestionNo: "1", Stem: "real stem", SourceChunkIDs: []string{"body-1", "answer-1"}},
			{QuestionNo: "2", Stem: "【2题答案】", SourceChunkIDs: []string{"answer-1"}},
			{QuestionNo: "99", Stem: "outside target"},
		},
		SourceChunkIDs: []string{"body-1", "answer-1"},
	}}

	got := constrainMathBatchCandidates(candidates, batch)

	if len(got) != 1 || len(got[0].Questions) != 1 {
		t.Fatalf("expected one real problem after filtering, got %#v", got)
	}
	if got[0].Questions[0].QuestionNo != "1" {
		t.Fatalf("unexpected question number: %q", got[0].Questions[0].QuestionNo)
	}
	if strings.Contains(strings.Join(got[0].SourceChunkIDs, ","), "answer-1") {
		t.Fatalf("answer source leaked into question sources: %v", got[0].SourceChunkIDs)
	}
}

func TestConstrainMathBatchCandidatesNormalizesSubquestionNumber(t *testing.T) {
	batch := questionGroupExtractionBatch{
		CoreChunks:            []*types.Chunk{{ID: "body-15", Content: "15. problem"}},
		TargetQuestionNumbers: []int{15},
	}
	candidates := []*types.ExamQuestionGroupDraftCandidate{{
		GroupNo: "15", Title: "Question 15",
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{
			{QuestionNo: "1", Stem: "first subquestion"},
			{QuestionNo: "2", Stem: "second subquestion"},
		},
	}}

	got := constrainMathBatchCandidates(candidates, batch)

	if len(got) != 1 || len(got[0].Questions) != 2 {
		t.Fatalf("unexpected normalized candidates: %#v", got)
	}
	if got[0].Questions[0].QuestionNo != "15(1)" || got[0].Questions[1].QuestionNo != "15(2)" {
		t.Fatalf("unexpected subquestion numbers: %#v", got[0].Questions)
	}
}

func questionGroupBatchContents(chunks []*types.Chunk) string {
	contents := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		contents = append(contents, chunk.Content)
	}
	return strings.Join(contents, "\n")
}

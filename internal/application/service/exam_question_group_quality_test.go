package service

import (
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestEvaluateMathDraftQualityFindsBlockingProblems(t *testing.T) {
	draft := newMathPendingDraft("draft-math")
	questions, err := draftGroupQuestions(draft)
	if err != nil {
		t.Fatalf("draftGroupQuestions: %v", err)
	}
	questions[0].Options = []types.ExamQuestionDraftOption{{Key: "A", Content: ""}, {Key: "B", Content: "2"}}
	questions[0].QuestionTypeCode = "single_choice"
	questions[0].Answer = types.JSONMap{"value": ""}
	questions[0].Stem = "如图，求值"
	draft.QuestionsJSON = mustJSONForTest(questions)

	report := evaluateQuestionGroupDraftQuality(draft)

	assertQualityIssue(t, report, "empty_option", types.ExamQualitySeverityError)
	assertQualityIssue(t, report, "missing_answer", types.ExamQualitySeverityError)
	assertQualityIssue(t, report, "missing_figure_asset", types.ExamQualitySeverityError)
	if !report.Blocking {
		t.Fatal("quality report should block approval")
	}
}

func TestEvaluateMathDraftQualityBlocksIncompleteChoiceOptions(t *testing.T) {
	draft := newMathPendingDraft("draft-incomplete-options")
	questions, err := draftGroupQuestions(draft)
	if err != nil {
		t.Fatalf("draftGroupQuestions: %v", err)
	}
	questions[0].QuestionTypeCode = "single_choice"
	questions[0].Options = []types.ExamQuestionDraftOption{{Key: "A", Content: "only option"}}
	draft.QuestionsJSON = mustJSONForTest(questions)

	report := evaluateQuestionGroupDraftQuality(draft)

	assertQualityIssue(t, report, "missing_options", types.ExamQualitySeverityError)
}

func TestEvaluateMathDraftQualityFindsMissingSubquestionAndWarnings(t *testing.T) {
	draft := newMathPendingDraft("draft-subquestions")
	questions := []types.ExamQuestionGroupDraftQuestionCandidate{
		{QuestionNo: "15(1)", QuestionTypeCode: "math_problem", Stem: "证明两个平面垂直", Answer: types.JSONMap{"value": "见证明"}},
		{QuestionNo: "15(3)", QuestionTypeCode: "math_problem", Stem: "求点到平面的距离", Answer: types.JSONMap{"value": "1"}},
	}
	draft.QuestionsJSON = mustJSONForTest(questions)

	report := evaluateQuestionGroupDraftQuality(draft)

	assertQualityIssue(t, report, "missing_subquestion", types.ExamQualitySeverityError)
	assertQualityIssue(t, report, "missing_explanation", types.ExamQualitySeverityWarning)
}

func TestPreflightQuestionGroupChunksWarnsForFormulaHeavyDocx(t *testing.T) {
	chunks := []*types.Chunk{
		{Content: "1. x-wmf/local://formula-1.wmf x-wmf/local://formula-2.wmf"},
		{Content: "2. image/x-wmf local://formula-3.wmf"},
	}

	warnings := preflightQuestionGroupChunks(chunks)

	if len(warnings) != 1 || warnings[0].Code != "formula_heavy_docx" {
		t.Fatalf("warnings = %#v", warnings)
	}
	if warnings[0].ReferenceCount < 3 {
		t.Fatalf("reference count = %d", warnings[0].ReferenceCount)
	}
}

func TestApproveMathDraftRejectsBlockingQualityIssues(t *testing.T) {
	repo := newReadyQuestionGroupDraftRepo()
	draft := newMathPendingDraft("draft-invalid")
	questions, _ := draftGroupQuestions(draft)
	questions[0].Answer = types.JSONMap{}
	draft.QuestionsJSON = mustJSONForTest(questions)
	repo.drafts = []*types.ExamQuestionGroupDraft{draft}
	svc := newTestExamQuestionGroupDraftService(repo, &stubExamQuestionDraftSpace{canRead: true, canWrite: true}, nil, nil, nil)

	_, err := svc.ApproveDraft(t.Context(), 10000, "teacher-1", draft.ID)

	if !errors.Is(err, ErrExamDraftQualityBlocked) {
		t.Fatalf("ApproveDraft error = %v", err)
	}
}

func newMathGroupCandidate() *types.ExamQuestionGroupDraftCandidate {
	return &types.ExamQuestionGroupDraftCandidate{
		GroupNo:        "1",
		GroupType:      "math_problem",
		Title:          "第 1 题",
		MaterialFormat: "latex",
		StrategyCode:   gaokaoMathQuestionGroupStrategy,
		Questions: []types.ExamQuestionGroupDraftQuestionCandidate{{
			QuestionNo:       "1",
			QuestionTypeCode: "math_problem",
			Stem:             "已知 x+1=2，求 x 的值",
			Answer:           types.JSONMap{"value": "1"},
			Explanation:      "移项可得 x=1。",
			OrderInGroup:     1,
		}},
	}
}

func newMathPendingDraft(id string) *types.ExamQuestionGroupDraft {
	candidate := newMathGroupCandidate()
	return &types.ExamQuestionGroupDraft{
		ID:             id,
		TenantID:       10000,
		SpaceID:        "space-1",
		TaskID:         "task-1",
		MaterialID:     "material-1",
		QuestionBankID: "bank-1",
		DomainID:       "gaokao",
		GroupType:      "math_problem",
		Title:          candidate.Title,
		MaterialFormat: "latex",
		QuestionsJSON:  mustJSONForTest(candidate.Questions),
		AssetsJSON:     mustJSONForTest(candidate.Assets),
		SourceChunkIDs: mustJSONForTest([]string{"chunk-1"}),
		StrategyCode:   gaokaoMathQuestionGroupStrategy,
		Status:         types.ExamQuestionGroupDraftStatusPendingReview,
	}
}

func assertQualityIssue(t *testing.T, report types.ExamQuestionGroupQualityReport, code string, severity types.ExamQualitySeverity) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Code == code && issue.Severity == severity {
			return
		}
	}
	t.Fatalf("quality issue %s/%s not found in %#v", code, severity, report.Issues)
}

package examrag

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestMatchEvaluationQuestionGroupSelectsUniqueEnglishPassage(t *testing.T) {
	t.Parallel()

	expected := evaluationCandidate(
		"group-stadium",
		"SoFi Stadium Events",
		"Los Angeles Rams v Dallas Cowboys. Los Angeles Chargers v Los Angeles Rams. Nearby Hotels and Parking.",
		"Which team will play the most games at the stadium this month?",
	)
	other := evaluationCandidate(
		"group-dictionary",
		"Dictionary Curiosity",
		"Kevin asked about the meaning of a word and looked at the bookshelf.",
		"How did Kevin feel while looking up the word?",
	)

	matched, confidence := matchEvaluationQuestionGroup(
		"Which team will play the most games at the stadium this month?",
		[]string{"Events this month: Los Angeles Rams v Dallas Cowboys; Los Angeles Chargers v Los Angeles Rams."},
		[]*types.QuestionGroupDetail{other, expected},
	)

	if matched == nil || matched.Group.ID != "group-stadium" {
		t.Fatalf("matched = %#v", matched)
	}
	if confidence < 0.6 || confidence > 1 {
		t.Fatalf("confidence = %.3f", confidence)
	}
}

func TestMatchEvaluationQuestionGroupNormalizesChineseMarkdown(t *testing.T) {
	t.Parallel()

	expected := evaluationCandidate(
		"group-circles",
		"圆与直线弦长多选题",
		"",
		"已知圆 $C_1:(x+1)^2+y^2=1$，直线 $l:y=kx+b$ 与圆相交，记 $l$ 截得的弦长为 $s_1$。",
	)
	other := evaluationCandidate("group-space", "立体几何", "", "动点 C 到直线 AB 的距离为 2。")

	matched, confidence := matchEvaluationQuestionGroup(
		"直线与三个圆均有两个交点，比较被三个圆截得的弦长。",
		[]string{"已知圆 C_1 : (x + 1)^2 + y^2 = 1，直线 l : y = kx + b 与圆相交，记 l 截得的弦长为 s_1。"},
		[]*types.QuestionGroupDetail{expected, other},
	)

	if matched == nil || matched.Group.ID != "group-circles" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupRejectsAmbiguousCandidates(t *testing.T) {
	t.Parallel()

	shared := "A shared source paragraph with enough stable words to satisfy the evaluation anchor threshold."
	first := evaluationCandidate("group-first", "First", shared, "What is the result?")
	second := evaluationCandidate("group-second", "Second", shared, "What is the result?")

	matched, confidence := matchEvaluationQuestionGroup(
		"What is the result?",
		[]string{shared},
		[]*types.QuestionGroupDetail{first, second},
	)

	if matched != nil || confidence != 0 {
		t.Fatalf("expected ambiguous match rejection, got %#v %.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupRejectsWeakEvidence(t *testing.T) {
	t.Parallel()

	candidate := evaluationCandidate("group-weak", "Reading", "A short unrelated passage.", "What is the answer?")
	matched, confidence := matchEvaluationQuestionGroup(
		"answer?",
		[]string{"short text"},
		[]*types.QuestionGroupDetail{candidate},
	)

	if matched != nil || confidence != 0 {
		t.Fatalf("expected weak match rejection, got %#v %.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupIgnoresIrrelevantRetrievedDocuments(t *testing.T) {
	t.Parallel()

	expected := evaluationCandidate(
		"group-prism", "Question 9", "",
		"在正三棱柱 ![](local://figure.png)中，D 为 BC 的中点，则（ ）",
	)
	irrelevant := evaluationCandidate(
		"group-museum", "AI in Museums",
		"Museums use machine learning to create personalized visitor journeys, protect collections, and reveal hidden stories.",
		"Which option best fits the museum passage?",
	)

	matched, confidence := matchEvaluationQuestionGroup(
		"2025 数学第 9 题中，正三棱柱内 D 为 BC 中点时判断空间关系。",
		[]string{
			"Museums use machine learning to create personalized visitor journeys, protect collections, and reveal hidden stories.",
			"第9题 在正三棱柱 ABC-A1B1C1 中，D为BC的中点，则下列空间关系正确的是",
		},
		[]*types.QuestionGroupDetail{irrelevant, expected},
	)

	if matched == nil || matched.Group.ID != "group-prism" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupRejectsRetrievedDocumentsUnrelatedToQuery(t *testing.T) {
	t.Parallel()

	candidate := evaluationCandidate(
		"group-museum", "AI in Museums",
		"Museums use machine learning to protect collections and reveal hidden stories.",
		"Which option best fits the museum passage?",
	)

	matched, confidence := matchEvaluationQuestionGroup(
		"求三维随机变量的数学期望",
		[]string{"Museums use machine learning to protect collections and reveal hidden stories."},
		[]*types.QuestionGroupDetail{candidate},
	)

	if matched != nil || confidence != 0 {
		t.Fatalf("expected unrelated retrieval rejection, got %#v %.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupWithAnchorsDistinguishesAdjacentQuestions(t *testing.T) {
	t.Parallel()

	previous := evaluationCandidate(
		"group-8", "Question 8", "",
		"若函数图像满足给定条件，则下列结论正确的是",
	)
	expected := evaluationCandidate(
		"group-9", "Question 9", "",
		"在正三棱柱 ![](local://figure.png)中，D 为 BC 的中点，则（ ）",
	)
	content := "第8题 若函数图像满足给定条件。第9题 在正三棱柱 ABC-A1B1C1 中，D为BC的中点，则下列空间关系正确的是。"

	matched, confidence := matchEvaluationQuestionGroupWithAnchors(
		"2025 数学第 9 题中，正三棱柱内 D 为 BC 中点时判断空间关系。",
		[]string{"在正三棱柱"},
		[]string{content},
		[]*types.QuestionGroupDetail{previous, expected},
	)

	if matched == nil || matched.Group.ID != "group-9" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupWithAnchorsIgnoresAdjacentRetrievalDominance(t *testing.T) {
	t.Parallel()

	previousStem := "已知函数图像经过多个特殊点并满足复杂的单调性周期性与对称性条件，则下列结论正确的是"
	previous := evaluationCandidate("group-8", "Question 8", "", previousStem)
	expected := evaluationCandidate(
		"group-9", "Question 9", "",
		"在正三棱柱中，D 为 BC 的中点，则下列空间关系正确的是",
	)
	content := previousStem + previousStem + previousStem + "。第9题：在正三棱柱中。"

	matched, confidence := matchEvaluationQuestionGroupWithAnchors(
		"2025 数学第 9 题中，正三棱柱内 D 为 BC 中点时判断空间关系。",
		[]string{"在正三棱柱"},
		[]string{content},
		[]*types.QuestionGroupDetail{previous, expected},
	)

	if matched == nil || matched.Group.ID != "group-9" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupWithAnchorsCollapsesEquivalentImports(t *testing.T) {
	t.Parallel()

	first := evaluationCandidate(
		"group-reading-a", "A. SoFi Stadium Events This Month",
		"Los Angeles Chargers v Los Angeles Rams.",
		"Which team will play the most games at the stadium this month?",
	)
	duplicate := evaluationCandidate(
		"group-reading-b", "A. SoFi Stadium Events This Month",
		"Los Angeles Chargers v Los Angeles Rams.",
		"Which team will play the most games at the stadium this month?",
	)

	matched, confidence := matchEvaluationQuestionGroupWithAnchors(
		"Which team will play the most games at the stadium this month?",
		[]string{"Los Angeles Chargers v Los Angeles Rams"},
		[]string{"Los Angeles Chargers v Los Angeles Rams"},
		[]*types.QuestionGroupDetail{duplicate, first},
	)

	if matched == nil || matched.Group.ID != "group-reading-a" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestMatchEvaluationQuestionGroupWithAnchorsHandlesParaphrasedSourceQuestion(t *testing.T) {
	t.Parallel()

	expected := evaluationCandidate(
		"group-14", "Question 14", "",
		"有5个相同的球，分别标有数字1，2，3，4，5，从中有放回地随机取3次，每次取1个球，记X为至少被取出1次的球数，则X的数学期望为",
	)
	other := evaluationCandidate(
		"group-13", "Question 13", "",
		"若一个等比数列的各项均为正数，且前4项的和等于4，则求这个数列的公比",
	)

	matched, confidence := matchEvaluationQuestionGroupWithAnchors(
		"5 个编号球有放回抽取 3 次，至少被抽出一次的球数 X 的期望是多少？",
		[]string{"有5个相同的球，分别标有数字1，2，3，4，5"},
		[]string{"有5个相同的球，分别标有数字1，2，3，4，5，从中有放回地随机取3次"},
		[]*types.QuestionGroupDetail{other, expected},
	)

	if matched == nil || matched.Group.ID != "group-14" {
		t.Fatalf("matched = %#v confidence=%.3f", matched, confidence)
	}
}

func TestNormalizeEvaluationAnchorTextRemovesImageURIs(t *testing.T) {
	t.Parallel()

	got := normalizeEvaluationAnchorText(
		"题干 alpha ![](local://10000/exports/2cbc75ab-f254-4904-9e02-1c81a93af71e.png) beta",
	)
	if got != "题干alphabeta" {
		t.Fatalf("normalized text = %q", got)
	}
}

func evaluationCandidate(id string, title string, material string, stem string) *types.QuestionGroupDetail {
	groupID := id
	return &types.QuestionGroupDetail{
		Group: &types.QuestionGroup{ID: id, Title: title, MaterialText: material},
		Questions: []*types.QuestionDetail{{
			Question: &types.Question{ID: id + "-question", GroupID: &groupID, Stem: stem},
			Options:  []*types.QuestionOption{{OptionKey: "A", Content: "First option"}},
		}},
	}
}

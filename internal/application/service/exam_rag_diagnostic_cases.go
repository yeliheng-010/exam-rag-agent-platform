package service

import (
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

func defaultExamRAGDiagnosticCasesForBank(bank *types.QuestionBank) []types.ExamRAGDiagnosticCase {
	if isGaokaoMathQuestionBank(bank) {
		return defaultGaokaoMathRAGDiagnosticCases()
	}
	return defaultExamRAGDiagnosticCases()
}

func isGaokaoMathQuestionBank(bank *types.QuestionBank) bool {
	if bank == nil || bank.SubjectID == nil {
		return false
	}
	domain := strings.TrimSpace(bank.DomainID)
	subject := strings.TrimSpace(*bank.SubjectID)
	domainMatches := domain == "gaokao" || domain == "00000000-0000-0000-0000-000000000101"
	subjectMatches := subject == "math" || subject == "00000000-0000-0000-0000-000000001102"
	return domainMatches && subjectMatches
}

func defaultGaokaoMathRAGDiagnosticCases() []types.ExamRAGDiagnosticCase {
	return []types.ExamRAGDiagnosticCase{
		{
			Name:            "gaokao_math_q1_full_context",
			Query:           "数学试卷第1题的完整题干、选项、答案和解析是什么？",
			RequiredPhrases: []string{"1.", "Answer:", "Explanation:"},
		},
		{
			Name:            "gaokao_math_q5_options",
			Query:           "数学试卷第5题的完整题干、四个选项、答案和解析是什么？",
			RequiredPhrases: []string{"5.", "A.", "B.", "C.", "D.", "Answer:"},
		},
		{
			Name:            "gaokao_math_q14_answer",
			Query:           "数学试卷第14题的题干、答案和解析是什么？",
			RequiredPhrases: []string{"14.", "Answer:", "Explanation:"},
		},
		{
			Name:            "gaokao_math_q15_figure",
			Query:           "数学试卷第15题的图形、全部小问、答案和解析是什么？",
			RequiredPhrases: []string{"[Assets]", "15(1).", "15(2).", "Answer:", "Explanation:"},
		},
	}
}

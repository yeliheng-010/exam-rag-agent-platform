package tools

import (
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func formatExamLearningDiagnosisResult(
	summary examLearningDiagnosisSummary,
	input ExamLearningDiagnosisInput,
) *types.ToolResult {
	if len(summary.Items) == 0 {
		return emptyExamLearningDiagnosisResult(summary, input)
	}
	var out strings.Builder
	writeDiagnosisHeader(&out, summary, input, false)
	for _, item := range summary.Items {
		writeDiagnosisItem(&out, item)
	}
	out.WriteString("<diagnosis_instruction>Use this evidence to identify weak points, explain likely causes, and propose a short review plan. Ask exam_question_context for full passage or grouped material when needed.</diagnosis_instruction>\n")
	out.WriteString("</exam_learning_diagnosis>")
	return &types.ToolResult{Success: true, Output: out.String(), Data: diagnosisResultData(summary, input)}
}

func emptyExamLearningDiagnosisResult(summary examLearningDiagnosisSummary, input ExamLearningDiagnosisInput) *types.ToolResult {
	var out strings.Builder
	writeDiagnosisHeader(&out, summary, input, true)
	out.WriteString("<message>no wrong questions found in the current scope</message>\n")
	out.WriteString("</exam_learning_diagnosis>")
	return &types.ToolResult{Success: true, Output: out.String(), Data: diagnosisResultData(summary, input)}
}

func writeDiagnosisHeader(out *strings.Builder, summary examLearningDiagnosisSummary, input ExamLearningDiagnosisInput, empty bool) {
	out.WriteString(fmt.Sprintf(
		"<exam_learning_diagnosis total_returned=\"%d\" total_considered=\"%d\" omitted_mastered=\"%d\" unreviewed=\"%d\" reviewing=\"%d\" mastered=\"%d\" include_mastered=\"%t\" empty=\"%t\">\n",
		summary.Returned,
		summary.Considered,
		summary.OmittedMastered,
		summary.StatusCounts[types.PracticeAnswerReviewStatusUnreviewed],
		summary.StatusCounts[types.PracticeAnswerReviewStatusReviewing],
		summary.StatusCounts[types.PracticeAnswerReviewStatusMastered],
		input.IncludeMastered,
		empty,
	))
}

func writeDiagnosisItem(out *strings.Builder, item examLearningDiagnosisItem) {
	out.WriteString(fmt.Sprintf(
		"<wrong_question answer_id=\"%s\" attempt_id=\"%s\" group_id=\"%s\" group_type=\"%s\" question_no=\"%s\" review_status=\"%s\" student_answer=\"%s\" correct_answer=\"%s\" answered_at=\"%s\">\n",
		xmlEscape(item.AnswerID),
		xmlEscape(item.AttemptID),
		xmlEscape(item.GroupID),
		xmlEscape(item.GroupType),
		xmlEscape(item.QuestionNo),
		xmlEscape(string(item.ReviewStatus)),
		xmlEscape(item.StudentAnswer),
		xmlEscape(item.CorrectAnswer),
		xmlEscape(item.AnsweredAt.Format(time.RFC3339)),
	))
	out.WriteString(fmt.Sprintf("<source bank=\"%s\" group_title=\"%s\" />\n", xmlEscape(item.BankName), xmlEscape(item.GroupTitle)))
	out.WriteString(fmt.Sprintf("<stem>%s</stem>\n", xmlEscape(item.Stem)))
	writeDiagnosisOptions(out, item.Options)
	out.WriteString(fmt.Sprintf("<explanation>%s</explanation>\n", xmlEscape(item.Explanation)))
	if item.ReviewNote != "" {
		out.WriteString(fmt.Sprintf("<review_note>%s</review_note>\n", xmlEscape(item.ReviewNote)))
	}
	out.WriteString("</wrong_question>\n")
}

func writeDiagnosisOptions(out *strings.Builder, options []string) {
	if len(options) == 0 {
		return
	}
	out.WriteString("<options>\n")
	for _, option := range options {
		out.WriteString(fmt.Sprintf("<option>%s</option>\n", xmlEscape(option)))
	}
	out.WriteString("</options>\n")
}

func diagnosisResultData(summary examLearningDiagnosisSummary, input ExamLearningDiagnosisInput) map[string]interface{} {
	return map[string]interface{}{
		"display_type":        ToolExamLearningDiagnosis,
		"space_id":            strings.TrimSpace(input.SpaceID),
		"group_id":            strings.TrimSpace(input.GroupID),
		"include_mastered":    input.IncludeMastered,
		"total_returned":      summary.Returned,
		"total_considered":    summary.Considered,
		"omitted_mastered":    summary.OmittedMastered,
		"review_status_count": summary.StatusCounts,
		"wrong_questions":     summary.Items,
	}
}

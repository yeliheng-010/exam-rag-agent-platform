package tools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const classTrendThreshold = 0.05

type examClassTrend struct {
	Direction string
	Delta     float64
	Points    []*types.ExamClassAnalyticsAssignment
}

type examAtRiskStudent struct {
	Member             *types.ExamClassMember
	Reasons            []string
	AssignmentCount    int
	StartedCount       int
	CompletedCount     int
	CompletionRate     float64
	AverageCorrectRate float64
}

func formatExamClassSelectionResult(classes []*types.ExamClass) *types.ToolResult {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("<exam_class_selection count=\"%d\">\n", len(classes)))
	for _, class := range classes {
		if class == nil {
			continue
		}
		out.WriteString(fmt.Sprintf("<class class_id=\"%s\" name=\"%s\" />\n", xmlEscape(class.ID), xmlEscape(class.Name)))
	}
	if len(classes) > 1 {
		out.WriteString("<instruction>Select the intended class and call exam_class_diagnosis again with class_id.</instruction>\n")
	} else {
		out.WriteString("<instruction>No analyzable class is available for the current user.</instruction>\n")
	}
	out.WriteString("</exam_class_selection>")
	return &types.ToolResult{Success: true, Output: out.String(), Data: map[string]interface{}{
		"display_type": ToolExamClassDiagnosis,
		"mode":         "class_selection",
		"classes":      classes,
	}}
}

func formatExamClassDiagnosisResult(
	summary *types.ExamClassAnalyticsSummary,
	input ExamClassDiagnosisInput,
) *types.ToolResult {
	if summary == nil || summary.Class == nil {
		return failedExamClassDiagnosisResult("class analytics are unavailable")
	}
	trend := buildExamClassTrend(summary.Assignments)
	atRisk := buildExamAtRiskStudents(summary.Members, input.AtRiskStudents)
	wrong := limitFrequentWrongQuestions(summary.FrequentWrongQuestions, input.TopWrongQuestions)
	output := buildExamClassDiagnosisOutput(summary, trend, atRisk, wrong)
	return &types.ToolResult{Success: true, Output: output, Data: map[string]interface{}{
		"display_type":             ToolExamClassDiagnosis,
		"class":                    summary.Class,
		"summary":                  summary,
		"mastery_trend":            trend,
		"at_risk_students":         atRisk,
		"frequent_wrong_questions": wrong,
	}}
}

func buildExamClassDiagnosisOutput(
	summary *types.ExamClassAnalyticsSummary,
	trend examClassTrend,
	atRisk []examAtRiskStudent,
	wrong []*types.ExamClassFrequentWrongQuestion,
) string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("<exam_class_diagnosis class_id=\"%s\" class_name=\"%s\">\n", xmlEscape(summary.Class.ID), xmlEscape(summary.Class.Name)))
	writeExamClassSummary(&out, summary)
	writeExamClassTrend(&out, trend)
	writeFrequentWrongQuestions(&out, wrong)
	writeAtRiskStudents(&out, atRisk)
	out.WriteString("<diagnosis_instruction>Base teaching priorities only on this evidence. Use exam_question_context for full question, passage, options, answer, or explanation before explaining a frequent wrong question.</diagnosis_instruction>\n")
	out.WriteString("</exam_class_diagnosis>")
	return out.String()
}

func writeExamClassSummary(out *strings.Builder, summary *types.ExamClassAnalyticsSummary) {
	out.WriteString(fmt.Sprintf(
		"<class_summary total_students=\"%d\" assignment_count=\"%d\" started_count=\"%d\" completed_count=\"%d\" completion_rate=\"%s\" average_correct_rate=\"%s\" />\n",
		summary.TotalStudents,
		summary.AssignmentCount,
		summary.StartedCount,
		summary.CompletedCount,
		formatDiagnosisPercent(summary.CompletionRate),
		formatDiagnosisPercent(summary.AverageCorrectRate),
	))
}

func buildExamClassTrend(assignments []*types.ExamClassAnalyticsAssignment) examClassTrend {
	points := make([]*types.ExamClassAnalyticsAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment != nil && assignment.Assignment != nil && assignment.StartedCount > 0 {
			points = append(points, assignment)
		}
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].Assignment.CreatedAt.Before(points[j].Assignment.CreatedAt)
	})
	trend := examClassTrend{Direction: "insufficient_data", Points: points}
	if len(points) < 2 {
		return trend
	}
	mid := len(points) / 2
	trend.Delta = averageAssignmentCorrectRate(points[mid:]) - averageAssignmentCorrectRate(points[:mid])
	trend.Direction = "stable"
	if trend.Delta >= classTrendThreshold {
		trend.Direction = "improving"
	} else if trend.Delta <= -classTrendThreshold {
		trend.Direction = "declining"
	}
	return trend
}

func averageAssignmentCorrectRate(items []*types.ExamClassAnalyticsAssignment) float64 {
	if len(items) == 0 {
		return 0
	}
	total := 0.0
	for _, item := range items {
		total += item.AverageCorrectRate
	}
	return total / float64(len(items))
}

func writeExamClassTrend(out *strings.Builder, trend examClassTrend) {
	out.WriteString(fmt.Sprintf("<mastery_trend direction=\"%s\" delta=\"%s\">\n", trend.Direction, formatDiagnosisPercent(trend.Delta)))
	for _, point := range trend.Points {
		out.WriteString(fmt.Sprintf(
			"<assignment assignment_id=\"%s\" title=\"%s\" completion_rate=\"%s\" average_correct_rate=\"%s\" />\n",
			xmlEscape(point.Assignment.ID), xmlEscape(point.Assignment.Title),
			formatDiagnosisPercent(point.CompletionRate), formatDiagnosisPercent(point.AverageCorrectRate),
		))
	}
	out.WriteString("</mastery_trend>\n")
}

func limitFrequentWrongQuestions(
	items []*types.ExamClassFrequentWrongQuestion,
	limit int,
) []*types.ExamClassFrequentWrongQuestion {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

func writeFrequentWrongQuestions(out *strings.Builder, items []*types.ExamClassFrequentWrongQuestion) {
	out.WriteString(fmt.Sprintf("<frequent_wrong_questions count=\"%d\">\n", len(items)))
	for _, item := range items {
		if item == nil {
			continue
		}
		out.WriteString(fmt.Sprintf(
			"<wrong_question question_id=\"%s\" question_no=\"%s\" answer_count=\"%d\" wrong_count=\"%d\" wrong_rate=\"%s\" affected_students=\"%d\"><stem>%s</stem></wrong_question>\n",
			xmlEscape(item.QuestionID), xmlEscape(item.QuestionNo), item.AnswerCount, item.WrongCount,
			formatDiagnosisPercent(item.WrongRate), item.AffectedStudentCount, xmlEscape(item.Stem),
		))
	}
	out.WriteString("</frequent_wrong_questions>\n")
}

func buildExamAtRiskStudents(items []*types.ExamClassAnalyticsMember, limit int) []examAtRiskStudent {
	out := make([]examAtRiskStudent, 0, len(items))
	for _, item := range items {
		candidate, ok := examAtRiskStudentFromAnalytics(item)
		if ok {
			out = append(out, candidate)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartedCount == 0 && out[j].StartedCount > 0 {
			return true
		}
		if out[i].StartedCount > 0 && out[j].StartedCount == 0 {
			return false
		}
		if out[i].CompletionRate != out[j].CompletionRate {
			return out[i].CompletionRate < out[j].CompletionRate
		}
		return out[i].AverageCorrectRate < out[j].AverageCorrectRate
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func examAtRiskStudentFromAnalytics(item *types.ExamClassAnalyticsMember) (examAtRiskStudent, bool) {
	if item == nil || item.Member == nil || item.AssignmentCount == 0 {
		return examAtRiskStudent{}, false
	}
	reasons := make([]string, 0, 3)
	if item.StartedCount == 0 {
		reasons = append(reasons, "not_started")
	}
	if item.CompletionRate < 0.5 {
		reasons = append(reasons, "low_completion")
	}
	if item.StartedCount > 0 && item.AverageCorrectRate < 0.6 {
		reasons = append(reasons, "low_accuracy")
	}
	return examAtRiskStudent{
		Member: item.Member, Reasons: reasons, AssignmentCount: item.AssignmentCount,
		StartedCount: item.StartedCount, CompletedCount: item.CompletedCount,
		CompletionRate: item.CompletionRate, AverageCorrectRate: item.AverageCorrectRate,
	}, len(reasons) > 0
}

func writeAtRiskStudents(out *strings.Builder, items []examAtRiskStudent) {
	out.WriteString(fmt.Sprintf("<at_risk_students count=\"%d\">\n", len(items)))
	for _, item := range items {
		name := firstNonEmpty(item.Member.DisplayName, item.Member.UserID)
		out.WriteString(fmt.Sprintf(
			"<student user_id=\"%s\" student_name=\"%s\" reason=\"%s\" started=\"%d\" completed=\"%d\" completion_rate=\"%s\" average_correct_rate=\"%s\" />\n",
			xmlEscape(item.Member.UserID), xmlEscape(name), xmlEscape(strings.Join(item.Reasons, ",")),
			item.StartedCount, item.CompletedCount, formatDiagnosisPercent(item.CompletionRate),
			formatDiagnosisPercent(item.AverageCorrectRate),
		))
	}
	out.WriteString("</at_risk_students>\n")
}

func formatDiagnosisPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}

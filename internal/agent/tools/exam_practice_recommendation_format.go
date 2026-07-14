package tools

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

func formatExamPracticeRecommendationResult(
	result *types.ExamPracticeRecommendationResult,
) *types.ToolResult {
	if result == nil {
		return failedExamPracticeRecommendationResult("practice recommendations are unavailable")
	}
	var out strings.Builder
	writePracticeRecommendationHeader(&out, result)
	writePracticeRecommendationDiagnosis(&out, result.Diagnosis)
	writePracticeRecommendations(&out, result.Recommendations)
	writePracticeRecommendationWarnings(&out, result.Warnings)
	writePracticeRecommendationInstruction(&out, result.Scope)
	out.WriteString("</exam_practice_recommendations>")
	return &types.ToolResult{Success: true, Output: out.String(), Data: map[string]interface{}{
		"display_type":    ToolExamPracticeRecommendation,
		"scope":           result.Scope,
		"class":           result.Class,
		"diagnosis":       result.Diagnosis,
		"recommendations": result.Recommendations,
		"warnings":        result.Warnings,
	}}
}

func writePracticeRecommendationHeader(out *strings.Builder, result *types.ExamPracticeRecommendationResult) {
	classID, className := "", ""
	if result.Class != nil {
		classID, className = result.Class.ID, result.Class.Name
	}
	out.WriteString(fmt.Sprintf(
		"<exam_practice_recommendations scope=\"%s\" class_id=\"%s\" class_name=\"%s\" diagnosis_count=\"%d\" recommendation_count=\"%d\">\n",
		xmlEscape(string(result.Scope)), xmlEscape(classID), xmlEscape(className),
		len(result.Diagnosis), len(result.Recommendations),
	))
}

func writePracticeRecommendationDiagnosis(
	out *strings.Builder,
	items []types.ExamPracticeDiagnosisEvidence,
) {
	out.WriteString("<diagnosis_evidence>\n")
	for _, item := range items {
		out.WriteString(fmt.Sprintf(
			"<wrong_question question_id=\"%s\" group_id=\"%s\" question_no=\"%s\" wrong_rate=\"%s\" affected_students=\"%d\"><stem>%s</stem></wrong_question>\n",
			xmlEscape(item.QuestionID), xmlEscape(item.GroupID), xmlEscape(item.QuestionNo),
			formatDiagnosisPercent(item.WrongRate), item.AffectedStudentCount, xmlEscape(item.Stem),
		))
	}
	out.WriteString("</diagnosis_evidence>\n")
}

func writePracticeRecommendations(
	out *strings.Builder,
	items []*types.ExamPracticeRecommendation,
) {
	out.WriteString("<recommendations>\n")
	for _, item := range items {
		writePracticeRecommendation(out, item)
	}
	out.WriteString("</recommendations>\n")
}

func writePracticeRecommendation(out *strings.Builder, item *types.ExamPracticeRecommendation) {
	if item == nil || item.Group == nil || item.Group.Group == nil {
		return
	}
	group := item.Group.Group
	out.WriteString(fmt.Sprintf(
		"<recommendation group_id=\"%s\" bank_name=\"%s\" title=\"%s\" group_type=\"%s\" question_count=\"%d\" score=\"%d\" assigned=\"%t\">\n",
		xmlEscape(group.ID), xmlEscape(item.Group.BankName), xmlEscape(group.Title),
		xmlEscape(group.GroupType), item.Group.QuestionCount, item.Score, item.Assigned,
	))
	for _, reason := range item.Reasons {
		out.WriteString(fmt.Sprintf("<reason code=\"%s\" score=\"%d\" />\n", xmlEscape(string(reason.Code)), reason.Score))
	}
	for _, evidence := range item.Evidence {
		out.WriteString(fmt.Sprintf("<evidence question_id=\"%s\" question_no=\"%s\" />\n", xmlEscape(evidence.QuestionID), xmlEscape(evidence.QuestionNo)))
	}
	out.WriteString("</recommendation>\n")
}

func writePracticeRecommendationWarnings(out *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	out.WriteString("<warnings>\n")
	for _, warning := range warnings {
		out.WriteString(fmt.Sprintf("<warning code=\"%s\" />\n", xmlEscape(warning)))
	}
	out.WriteString("</warnings>\n")
}

func writePracticeRecommendationInstruction(
	out *strings.Builder,
	scope types.ExamPracticeRecommendationScope,
) {
	if scope == types.ExamPracticeRecommendationScopeClass {
		out.WriteString("<instruction>Explain the evidence and recommendation reasons. Teacher confirmation is required before publishing any assignment. Never invent a group_id.</instruction>\n")
		return
	}
	out.WriteString("<instruction>Explain why each group was recommended and suggest an order. Never invent a group_id or claim the student completed it.</instruction>\n")
}

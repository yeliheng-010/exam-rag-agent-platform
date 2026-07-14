package searchutil

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/examtext"
	"github.com/Tencent/WeKnora/internal/types"
)

const defaultStructuredExamContextRuneLimit = 9000

// BuildStructuredExamQuestionContextBundle turns a reviewed question group into
// a compact retrieval context that can be reused by Chat RAG and Agent tools.
func BuildStructuredExamQuestionContextBundle(query string, detail *types.QuestionGroupDetail) *ExamQuestionContextBundle {
	if detail == nil || detail.Group == nil {
		return nil
	}
	content := BuildStructuredExamQuestionContextText(detail, defaultStructuredExamContextRuneLimit)
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return &ExamQuestionContextBundle{
		Label:          structuredExamContextLabel(detail),
		Content:        content,
		SourceChunkIDs: structuredExamSourceChunkIDs(detail),
	}
}

// BuildStructuredExamQuestionContextText serializes the group material,
// questions, answers, explanations and evidence chunk ids into model-ready text.
func BuildStructuredExamQuestionContextText(detail *types.QuestionGroupDetail, runeLimit int) string {
	if detail == nil || detail.Group == nil {
		return ""
	}

	var builder strings.Builder
	group := detail.Group
	builder.WriteString("[Exam Structured Context]\n")
	writeStructuredLine(&builder, "Question group", firstNonEmpty(group.Title, group.GroupType, group.ID))
	writeStructuredLine(&builder, "Group type", group.GroupType)
	if group.SourceYear != nil {
		writeStructuredLine(&builder, "Year", fmt.Sprintf("%d", *group.SourceYear))
	}
	writeStructuredLine(&builder, "Region", group.SourceRegion)
	writeStructuredLine(&builder, "Paper type", group.PaperType)
	if refs := structuredJSONStrings(group.SourceChunkIDs); len(refs) > 0 {
		writeStructuredLine(&builder, "Group chunks", strings.Join(refs, ", "))
	}
	writeStructuredAssets(&builder, detail.Assets)

	if material := strings.TrimSpace(group.MaterialText); material != "" {
		builder.WriteString("\n[Original Material]\n")
		builder.WriteString(material)
		builder.WriteString("\n")
	}

	questions := sortedStructuredQuestions(detail.Questions)
	if len(questions) > 0 {
		builder.WriteString("\n[Questions]\n")
	}
	for _, question := range questions {
		writeStructuredQuestion(&builder, question, group.MaterialText)
	}

	return trimRunes(strings.TrimSpace(builder.String()), runeLimitOrDefault(runeLimit))
}

func writeStructuredQuestion(builder *strings.Builder, detail *types.QuestionDetail, material string) {
	if builder == nil || detail == nil || detail.Question == nil {
		return
	}
	question := detail.Question
	no := strings.TrimSpace(question.QuestionNo)
	if no == "" {
		no = fmt.Sprintf("%d", question.OrderInGroup)
	}
	stem := strings.TrimSpace(question.Stem)
	if stem == "" {
		return
	}
	builder.WriteString("\n")
	builder.WriteString(no)
	builder.WriteString(". ")
	builder.WriteString(stem)
	builder.WriteString("\n")

	for _, option := range sortedStructuredOptions(detail.Options) {
		if option == nil || strings.TrimSpace(option.Content) == "" {
			continue
		}
		builder.WriteString(strings.TrimSpace(option.OptionKey))
		builder.WriteString(". ")
		builder.WriteString(strings.TrimSpace(option.Content))
		builder.WriteString("\n")
	}
	if answers := structuredAnswerTexts(detail.Answers); len(answers) > 0 {
		writeStructuredLine(builder, "Answer", strings.Join(answers, ", "))
	}
	if explanations := structuredExplanationTexts(detail.Explanations); len(explanations) > 0 {
		writeStructuredLine(builder, "Explanation", strings.Join(explanations, " / "))
	}
	if len(structuredExplanationTexts(detail.Explanations)) == 0 {
		if explanation := fallbackStructuredExplanation(detail, material); explanation != "" {
			writeStructuredLine(builder, "Explanation", explanation)
		}
	}
	if refs := structuredChunkRefs(detail.ChunkRefs); len(refs) > 0 {
		writeStructuredLine(builder, "Evidence chunk", strings.Join(refs, ", "))
	}
}

func fallbackStructuredExplanation(detail *types.QuestionDetail, material string) string {
	if detail == nil {
		return ""
	}
	answers := structuredAnswerTexts(detail.Answers)
	if len(answers) == 0 {
		return ""
	}
	return examtext.BuildBaselineExplanation(
		strings.Join(answers, ","),
		structuredExplanationOptions(detail.Options),
		material,
	)
}

func structuredExplanationOptions(options []*types.QuestionOption) []examtext.ExplanationOption {
	out := make([]examtext.ExplanationOption, 0, len(options))
	for _, option := range options {
		if option == nil {
			continue
		}
		out = append(out, examtext.ExplanationOption{
			Key:       strings.TrimSpace(option.OptionKey),
			Content:   strings.TrimSpace(option.Content),
			SortOrder: option.SortOrder,
		})
	}
	return out
}

func structuredExamContextLabel(detail *types.QuestionGroupDetail) string {
	if detail == nil || detail.Group == nil {
		return ""
	}
	groupType := strings.TrimSpace(detail.Group.GroupType)
	title := strings.TrimSpace(detail.Group.Title)
	if groupType == "" {
		return title
	}
	if title == "" {
		return groupType
	}
	return groupType + ":" + title
}

func sortedStructuredQuestions(questions []*types.QuestionDetail) []*types.QuestionDetail {
	out := make([]*types.QuestionDetail, 0, len(questions))
	for _, question := range questions {
		if question != nil && question.Question != nil {
			out = append(out, question)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := out[i].Question
		right := out[j].Question
		if left.OrderInGroup == right.OrderInGroup {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.OrderInGroup < right.OrderInGroup
	})
	return out
}

func sortedStructuredOptions(options []*types.QuestionOption) []*types.QuestionOption {
	out := append([]*types.QuestionOption{}, options...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i] == nil {
			return false
		}
		if out[j] == nil {
			return true
		}
		if out[i].SortOrder == out[j].SortOrder {
			return strings.TrimSpace(out[i].OptionKey) < strings.TrimSpace(out[j].OptionKey)
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}

func structuredAnswerTexts(answers []*types.QuestionAnswer) []string {
	out := make([]string, 0, len(answers))
	seen := make(map[string]bool, len(answers))
	for _, answer := range answers {
		if answer == nil || !answer.IsCorrect {
			continue
		}
		text := strings.TrimSpace(answer.AnswerText)
		if text == "" || seen[text] {
			continue
		}
		seen[text] = true
		out = append(out, text)
	}
	return out
}

func structuredExplanationTexts(explanations []*types.QuestionExplanation) []string {
	out := make([]string, 0, len(explanations))
	for _, explanation := range explanations {
		if explanation == nil {
			continue
		}
		text := strings.TrimSpace(explanation.ExplanationText)
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func structuredChunkRefs(refs []*types.QuestionChunkRef) []string {
	out := make([]string, 0, len(refs))
	seen := make(map[string]bool, len(refs))
	for _, ref := range refs {
		if ref == nil {
			continue
		}
		chunkID := strings.TrimSpace(ref.ChunkID)
		if chunkID == "" || seen[chunkID] {
			continue
		}
		seen[chunkID] = true
		out = append(out, chunkID)
	}
	return out
}

func structuredExamSourceChunkIDs(detail *types.QuestionGroupDetail) []string {
	if detail == nil || detail.Group == nil {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0)
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	for _, id := range structuredJSONStrings(detail.Group.SourceChunkIDs) {
		add(id)
	}
	for _, question := range detail.Questions {
		if question == nil {
			continue
		}
		for _, id := range structuredChunkRefs(question.ChunkRefs) {
			add(id)
		}
	}
	return out
}

func structuredJSONStrings(value types.JSON) []string {
	var out []string
	if len(value) == 0 || !json.Valid(value) {
		return out
	}
	if err := json.Unmarshal(value, &out); err != nil {
		return nil
	}
	cleaned := out[:0]
	for _, item := range out {
		item = strings.TrimSpace(item)
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}

func writeStructuredLine(builder *strings.Builder, label string, value string) {
	value = strings.TrimSpace(value)
	if builder == nil || value == "" {
		return
	}
	builder.WriteString(label)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runeLimitOrDefault(limit int) int {
	if limit <= 0 {
		return defaultStructuredExamContextRuneLimit
	}
	return limit
}

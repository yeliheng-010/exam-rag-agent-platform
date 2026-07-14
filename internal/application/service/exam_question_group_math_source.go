package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	mathOptionPattern        = regexp.MustCompile(`(?m)(?:^|[\n\t ])([A-D])[.．][ \t]*`)
	mathAnswerHeadingPattern = regexp.MustCompile(`(?m)^\s*\*\*[^\n]*(选择题|填空题|解答题)[^\n]*$`)
)

func enrichMathBatchCandidates(candidates []*types.ExamQuestionGroupDraftCandidate, batch questionGroupExtractionBatch) []*types.ExamQuestionGroupDraftCandidate {
	sourceCandidates := buildMathSourceCandidates(batch)
	positions := make(map[int][]int, len(candidates))
	for index, candidate := range candidates {
		if number, ok := mathCandidateTopNumber(candidate); ok {
			positions[number] = append(positions[number], index)
		}
	}
	for _, source := range sourceCandidates {
		number, ok := mathCandidateTopNumber(source)
		if !ok {
			continue
		}
		matching := positions[number]
		if len(matching) == 0 {
			positions[number] = []int{len(candidates)}
			candidates = append(candidates, source)
			continue
		}
		for _, position := range matching {
			enrichMathCandidateFromSource(candidates[position], source, number)
		}
	}
	return candidates
}

func buildMathSourceCandidates(batch questionGroupExtractionBatch) []*types.ExamQuestionGroupDraftCandidate {
	bodyChunks := uniqueSortedMathSourceChunks(batch.CoreChunks, batch.ContextChunks)
	bodyContent := joinMathSourceChunkContent(bodyChunks)
	answerContent := joinMathSourceChunkContent(batch.AnswerChunks)
	sourceIDs := mathSourceChunkIDs(bodyChunks)
	candidates := make([]*types.ExamQuestionGroupDraftCandidate, 0, len(batch.TargetQuestionNumbers))
	for _, number := range batch.TargetQuestionNumbers {
		questionText := extractMathNumberedSection(bodyContent, bodyQuestionNumberPattern, number)
		if strings.TrimSpace(questionText) == "" {
			continue
		}
		answerText := cleanMathAnswerSection(extractMathNumberedSection(answerContent, answerQuestionNumberPattern, number))
		candidates = append(candidates, buildMathSourceCandidate(number, questionText, answerText, sourceIDs))
	}
	return candidates
}

func buildMathSourceCandidate(number int, questionText string, answerText string, sourceIDs []string) *types.ExamQuestionGroupDraftCandidate {
	questions, material := buildMathSourceQuestions(number, questionText, answerText, sourceIDs)
	return &types.ExamQuestionGroupDraftCandidate{
		GroupNo: strconv.Itoa(number), GroupType: "math_problem", Title: "Question " + strconv.Itoa(number),
		MaterialText: material, MaterialFormat: "markdown", SourceChunkIDs: append([]string{}, sourceIDs...),
		Assets: extractMathSourceAssets(questionText, sourceIDs), Questions: questions,
		Confidence: 1,
	}
}

func buildMathSourceQuestions(number int, questionText string, answerText string, sourceIDs []string) ([]types.ExamQuestionGroupDraftQuestionCandidate, string) {
	if number <= 14 {
		questionType := mathQuestionType(number)
		stem, options := splitMathQuestionOptions(questionText, questionType)
		return []types.ExamQuestionGroupDraftQuestionCandidate{
			newMathSourceQuestion(strconv.Itoa(number), questionType, stem, options, answerText, "", sourceIDs, 1),
		}, ""
	}
	material, sections := splitMathSubjectiveSections(questionText, number)
	if len(sections) == 0 {
		sections = []mathSourceSection{{QuestionNo: strconv.Itoa(number), Stem: questionText}}
	}
	questions := make([]types.ExamQuestionGroupDraftQuestionCandidate, 0, len(sections))
	for index, section := range sections {
		questions = append(questions, newMathSourceQuestion(
			section.QuestionNo, "math_problem", section.Stem, nil, answerText, answerText, sourceIDs, index+1,
		))
	}
	return questions, material
}

func newMathSourceQuestion(no string, questionType string, stem string, options []types.ExamQuestionDraftOption, answerText string, explanation string, sourceIDs []string, order int) types.ExamQuestionGroupDraftQuestionCandidate {
	answer := types.JSONMap{}
	if answerText != "" {
		answer = types.JSONMap{"value": answerText}
	}
	return types.ExamQuestionGroupDraftQuestionCandidate{
		QuestionNo: no, QuestionTypeCode: questionType, Stem: strings.TrimSpace(stem), Options: options,
		Answer: answer, Explanation: explanation, Evidence: []types.JSONMap{}, Metadata: types.JSONMap{},
		Difficulty: "unknown", Confidence: 1, OrderInGroup: order, SourceChunkIDs: append([]string{}, sourceIDs...),
	}
}

func enrichMathCandidateFromSource(candidate *types.ExamQuestionGroupDraftCandidate, source *types.ExamQuestionGroupDraftCandidate, number int) {
	if candidate == nil || source == nil || len(source.Questions) == 0 {
		return
	}
	if number <= 14 {
		for index := range candidate.Questions {
			enrichObjectiveMathQuestion(&candidate.Questions[index], source.Questions[0])
		}
	} else {
		enrichSubjectiveMathCandidate(candidate, source)
	}
	if candidate.MaterialText == "" {
		candidate.MaterialText = source.MaterialText
	}
	candidate.SourceChunkIDs = mergeQuestionGroupUniqueStrings(candidate.SourceChunkIDs, source.SourceChunkIDs)
	candidate.Assets = mergeQuestionGroupAssets(candidate.Assets, source.Assets)
}

func enrichSubjectiveMathCandidate(candidate *types.ExamQuestionGroupDraftCandidate, source *types.ExamQuestionGroupDraftCandidate) {
	if len(source.Questions) > 1 {
		candidate.Questions = filterMathQuestionsToSource(candidate.Questions, source.Questions)
	}
	for _, sourceQuestion := range source.Questions {
		position := questionGroupQuestionPosition(candidate.Questions, sourceQuestion.QuestionNo)
		if position < 0 {
			candidate.Questions = append(candidate.Questions, sourceQuestion)
			continue
		}
		enrichSubjectiveMathQuestion(&candidate.Questions[position], sourceQuestion)
	}
	for index := range candidate.Questions {
		candidate.Questions[index].OrderInGroup = index + 1
	}
}

func filterMathQuestionsToSource(questions []types.ExamQuestionGroupDraftQuestionCandidate, source []types.ExamQuestionGroupDraftQuestionCandidate) []types.ExamQuestionGroupDraftQuestionCandidate {
	allowed := make(map[string]bool, len(source))
	for _, question := range source {
		allowed[canonicalSubquestionKey(question)] = true
	}
	out := make([]types.ExamQuestionGroupDraftQuestionCandidate, 0, len(questions))
	for _, question := range questions {
		if allowed[canonicalSubquestionKey(question)] {
			out = append(out, question)
		}
	}
	return out
}

func questionGroupQuestionPosition(questions []types.ExamQuestionGroupDraftQuestionCandidate, questionNo string) int {
	key := canonicalSubquestionKey(types.ExamQuestionGroupDraftQuestionCandidate{QuestionNo: questionNo})
	for index := range questions {
		if canonicalSubquestionKey(questions[index]) == key {
			return index
		}
	}
	return -1
}

func enrichObjectiveMathQuestion(question *types.ExamQuestionGroupDraftQuestionCandidate, source types.ExamQuestionGroupDraftQuestionCandidate) {
	question.QuestionTypeCode = source.QuestionTypeCode
	if len(source.Stem) > len(question.Stem) {
		question.Stem = source.Stem
	}
	if mathOptionCompleteness(source.Options) > mathOptionCompleteness(question.Options) {
		question.Options = source.Options
	}
	if hasMeaningfulAnswer(source.Answer) {
		question.Answer = source.Answer
	}
	question.SourceChunkIDs = mergeQuestionGroupUniqueStrings(question.SourceChunkIDs, source.SourceChunkIDs)
}

func mathOptionCompleteness(options []types.ExamQuestionDraftOption) int {
	score := len(options) * 10
	for _, option := range options {
		score += len(strings.TrimSpace(option.Content))
	}
	return score
}

func enrichSubjectiveMathQuestion(question *types.ExamQuestionGroupDraftQuestionCandidate, source types.ExamQuestionGroupDraftQuestionCandidate) {
	if hasMeaningfulAnswer(source.Answer) {
		if !hasMeaningfulAnswer(question.Answer) || len(draftAnswerText(question.Answer)) < len(draftAnswerText(source.Answer)) {
			question.Answer = source.Answer
		}
		if strings.TrimSpace(question.Explanation) == "" {
			question.Explanation = source.Explanation
		}
	}
	question.SourceChunkIDs = mergeQuestionGroupUniqueStrings(question.SourceChunkIDs, source.SourceChunkIDs)
}

func extractMathNumberedSection(content string, pattern *regexp.Regexp, target int) string {
	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	start, end := -1, len(content)
	for index, match := range matches {
		number, _ := strconv.Atoi(content[match[2]:match[3]])
		if number == target {
			start = match[1]
			end = len(content)
			for next := index + 1; next < len(matches); next++ {
				nextNumber, _ := strconv.Atoi(content[matches[next][2]:matches[next][3]])
				if nextNumber != target {
					end = matches[next][0]
					break
				}
			}
		}
	}
	if start < 0 || start > end {
		return ""
	}
	return strings.TrimSpace(content[start:end])
}

func splitMathQuestionOptions(content string, questionType string) (string, []types.ExamQuestionDraftOption) {
	if questionType != "single_choice" && questionType != "multiple_choice" {
		return strings.TrimSpace(content), []types.ExamQuestionDraftOption{}
	}
	matches := mathOptionPattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(content), []types.ExamQuestionDraftOption{}
	}
	best := make(map[string]string, 4)
	for index, match := range matches {
		end := len(content)
		if index+1 < len(matches) {
			end = matches[index+1][0]
		}
		key := content[match[2]:match[3]]
		value := strings.TrimSpace(content[match[1]:end])
		if len(value) > len(best[key]) {
			best[key] = value
		}
	}
	options := make([]types.ExamQuestionDraftOption, 0, len(best))
	for _, key := range []string{"A", "B", "C", "D"} {
		if value := strings.TrimSpace(best[key]); value != "" {
			options = append(options, types.ExamQuestionDraftOption{Key: key, Content: value})
		}
	}
	return strings.TrimSpace(content[:matches[0][0]]), options
}

func cleanMathAnswerSection(content string) string {
	content = strings.TrimSpace(content)
	if index := strings.Index(content, "【答案】"); index >= 0 {
		content = strings.TrimSpace(content[index+len("【答案】"):])
	}
	if match := mathAnswerHeadingPattern.FindStringIndex(content); match != nil {
		content = strings.TrimSpace(content[:match[0]])
	}
	return content
}

func mathQuestionType(number int) string {
	switch {
	case number <= 8:
		return "single_choice"
	case number <= 11:
		return "multiple_choice"
	case number <= 14:
		return "fill_blank"
	default:
		return "math_problem"
	}
}

func mathCandidateTopNumber(candidate *types.ExamQuestionGroupDraftCandidate) (int, bool) {
	if candidate == nil || len(candidate.Questions) == 0 {
		return 0, false
	}
	number, err := strconv.Atoi(firstQuestionNumber(candidate.Questions[0].QuestionNo))
	return number, err == nil
}

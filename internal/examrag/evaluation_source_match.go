package examrag

import (
	"sort"
	"strings"
	"unicode"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	evaluationSourceQueryAnchorRuneLength = 2
	evaluationSourceMinScore              = 3
	evaluationSourceMinGap                = 2
	evaluationQuestionNumberBonus         = 12
)

func matchEvaluationSourceCandidates(
	query string,
	candidates []*types.QuestionGroupDetail,
) (*types.QuestionGroupDetail, float64) {
	queryAnchors := evaluationAnchorSetWithLength([]string{query}, evaluationSourceQueryAnchorRuneLength)
	scores := make([]evaluationAnchorScore, 0, len(candidates))
	for _, candidate := range deduplicateEquivalentEvaluationCandidates(candidates) {
		candidateAnchors := evaluationAnchorSetWithLength(
			evaluationCandidateQueryTexts(candidate), evaluationSourceQueryAnchorRuneLength,
		)
		score := countSharedEvaluationAnchors(candidateAnchors, queryAnchors)
		score += evaluationQuestionNumberBonus * evaluationQuestionNumberMatches(query, candidate)
		scores = append(scores, evaluationAnchorScore{detail: candidate, score: score})
	}
	if len(scores) == 0 {
		return nil, 0
	}
	sortEvaluationAnchorScores(scores)
	secondScore := 0
	if len(scores) > 1 {
		secondScore = scores[1].score
	}
	if !isUniqueEvaluationSourceScore(scores[0].score, secondScore) {
		return nil, 0
	}
	return scores[0].detail, evaluationAssociationConfidence(scores[0].score, secondScore)
}

func isUniqueEvaluationSourceScore(best int, second int) bool {
	if best < evaluationSourceMinScore {
		return false
	}
	if second == 0 {
		return true
	}
	return best-second >= evaluationSourceMinGap && best*4 >= second*5
}

func evaluationSourcePhraseMatchesContents(phrases []string, contents []string) bool {
	for _, phrase := range phrases {
		phrase = normalizeEvaluationAnchorText(phrase)
		if phrase == "" {
			continue
		}
		for _, content := range contents {
			if strings.Contains(normalizeEvaluationAnchorText(content), phrase) {
				return true
			}
		}
	}
	return false
}

func evaluationCandidateQueryTexts(detail *types.QuestionGroupDetail) []string {
	if detail == nil || detail.Group == nil {
		return nil
	}
	texts := []string{detail.Group.Title}
	for _, question := range detail.Questions {
		if question != nil && question.Question != nil {
			texts = append(texts, question.Question.QuestionNo, question.Question.Stem)
		}
	}
	return texts
}

func evaluationQuestionNumberMatches(query string, detail *types.QuestionGroupDetail) int {
	if detail == nil || detail.Group == nil {
		return 0
	}
	normalizedQuery := normalizeEvaluationAnchorText(query)
	for _, question := range detail.Questions {
		if question == nil || question.Question == nil {
			continue
		}
		number := evaluationLeadingDigits(question.Question.QuestionNo)
		if evaluationQueryContainsQuestionNumber(normalizedQuery, number) {
			return 1
		}
	}
	if evaluationQueryContainsQuestionNumber(normalizedQuery, evaluationTitleQuestionNumber(detail.Group.Title)) {
		return 1
	}
	return 0
}

func evaluationQueryContainsQuestionNumber(query string, number string) bool {
	return number != "" && (strings.Contains(query, "第"+number+"题") ||
		strings.Contains(query, "question"+number))
}

func evaluationTitleQuestionNumber(title string) string {
	normalized := normalizeEvaluationAnchorText(title)
	if strings.HasPrefix(normalized, "question") {
		return evaluationLeadingDigits(strings.TrimPrefix(normalized, "question"))
	}
	if strings.HasPrefix(normalized, "第") {
		return evaluationLeadingDigits(strings.TrimPrefix(normalized, "第"))
	}
	return ""
}

func evaluationLeadingDigits(value string) string {
	var builder strings.Builder
	for _, char := range strings.TrimSpace(value) {
		if !unicode.IsDigit(char) {
			break
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func deduplicateEquivalentEvaluationCandidates(
	candidates []*types.QuestionGroupDetail,
) []*types.QuestionGroupDetail {
	ordered := append([]*types.QuestionGroupDetail{}, candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return evaluationCandidateID(ordered[i]) < evaluationCandidateID(ordered[j])
	})
	seen := make(map[string]bool)
	unique := make([]*types.QuestionGroupDetail, 0, len(ordered))
	for _, candidate := range ordered {
		signature := evaluationCandidateSignature(candidate)
		if signature == "" || seen[signature] {
			continue
		}
		seen[signature] = true
		unique = append(unique, candidate)
	}
	return unique
}

func evaluationCandidateID(detail *types.QuestionGroupDetail) string {
	if detail == nil || detail.Group == nil {
		return ""
	}
	return detail.Group.ID
}

func evaluationCandidateSignature(detail *types.QuestionGroupDetail) string {
	if detail == nil || detail.Group == nil {
		return ""
	}
	texts := evaluationCandidateTexts(detail)
	for _, question := range detail.Questions {
		if question == nil {
			continue
		}
		for _, answer := range question.Answers {
			if answer != nil {
				texts = append(texts, answer.AnswerText)
			}
		}
	}
	for index := range texts {
		texts[index] = normalizeEvaluationAnchorText(texts[index])
	}
	return strings.Join(texts, "\x1f")
}

func sortEvaluationAnchorScores(scores []evaluationAnchorScore) {
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].detail.Group.ID < scores[j].detail.Group.ID
		}
		return scores[i].score > scores[j].score
	})
}

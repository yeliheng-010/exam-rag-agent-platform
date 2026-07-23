package examrag

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	evaluationMarkdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	evaluationURLPattern           = regexp.MustCompile(`(?i)(?:local|https?)://[^\s)]+`)
)

const (
	evaluationRetrievalAnchorRuneLength = 6
	evaluationQueryAnchorRuneLength     = 4
	evaluationRelevantContentLimit      = 8
	evaluationQueryScoreWeight          = 3
	evaluationAnchorMinScore            = 4
	evaluationAnchorMinGap              = 3
)

type evaluationAnchorScore struct {
	detail *types.QuestionGroupDetail
	score  int
}

func matchEvaluationQuestionGroup(
	query string,
	retrievedContents []string,
	candidates []*types.QuestionGroupDetail,
) (*types.QuestionGroupDetail, float64) {
	return matchEvaluationQuestionGroupWithAnchors(query, nil, retrievedContents, candidates)
}

func matchEvaluationQuestionGroupWithAnchors(
	query string,
	evaluationAnchors []string,
	retrievedContents []string,
	candidates []*types.QuestionGroupDetail,
) (*types.QuestionGroupDetail, float64) {
	if evaluationSourcePhraseMatchesContents(evaluationAnchors, retrievedContents) {
		return matchEvaluationSourceCandidates(query, candidates)
	}
	queryTexts := append([]string{query}, evaluationAnchors...)
	relevantContents := selectQueryRelevantEvaluationContents(queryTexts, retrievedContents)
	if len(relevantContents) == 0 {
		return nil, 0
	}
	retrievedAnchors := evaluationAnchorSet(relevantContents)
	queryAnchors := evaluationAnchorSetWithLength(queryTexts, evaluationQueryAnchorRuneLength)
	scores := make([]evaluationAnchorScore, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.Group == nil {
			continue
		}
		candidateTexts := evaluationCandidateTexts(candidate)
		candidateAnchors := evaluationAnchorSet(candidateTexts)
		candidateQueryAnchors := evaluationAnchorSetWithLength(candidateTexts, evaluationQueryAnchorRuneLength)
		score := countSharedEvaluationAnchors(candidateAnchors, retrievedAnchors)
		score += evaluationQueryScoreWeight * countSharedEvaluationAnchors(candidateQueryAnchors, queryAnchors)
		scores = append(scores, evaluationAnchorScore{detail: candidate, score: score})
	}
	if len(scores) == 0 {
		return nil, 0
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].detail.Group.ID < scores[j].detail.Group.ID
		}
		return scores[i].score > scores[j].score
	})
	secondScore := 0
	if len(scores) > 1 {
		secondScore = scores[1].score
	}
	if !isUniqueEvaluationAnchorScore(scores[0].score, secondScore) {
		return nil, 0
	}
	return scores[0].detail, evaluationAssociationConfidence(scores[0].score, secondScore)
}

func evaluationCandidateTexts(detail *types.QuestionGroupDetail) []string {
	texts := make([]string, 0, 2+len(detail.Questions)*2)
	if detail == nil || detail.Group == nil {
		return texts
	}
	texts = append(texts, detail.Group.Title, detail.Group.MaterialText)
	for _, question := range detail.Questions {
		if question == nil || question.Question == nil {
			continue
		}
		texts = append(texts, question.Question.QuestionNo, question.Question.Stem)
		for _, option := range question.Options {
			if option != nil {
				texts = append(texts, option.Content)
			}
		}
	}
	return texts
}

func evaluationAnchorSet(texts []string) map[string]struct{} {
	return evaluationAnchorSetWithLength(texts, evaluationRetrievalAnchorRuneLength)
}

func evaluationAnchorSetWithLength(texts []string, length int) map[string]struct{} {
	anchors := make(map[string]struct{})
	for _, text := range texts {
		runes := []rune(normalizeEvaluationAnchorText(text))
		for index := 0; index+length <= len(runes); index++ {
			anchors[string(runes[index:index+length])] = struct{}{}
		}
	}
	return anchors
}

func selectQueryRelevantEvaluationContents(queryTexts []string, contents []string) []string {
	queryAnchors := evaluationAnchorSetWithLength(queryTexts, evaluationQueryAnchorRuneLength)
	if len(queryAnchors) == 0 {
		return nil
	}
	type scoredContent struct {
		content string
		score   int
		index   int
	}
	scores := make([]scoredContent, 0, len(contents))
	for index, content := range contents {
		contentAnchors := evaluationAnchorSetWithLength([]string{content}, evaluationQueryAnchorRuneLength)
		scores = append(scores, scoredContent{
			content: content, score: countSharedEvaluationAnchors(queryAnchors, contentAnchors), index: index,
		})
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].index < scores[j].index
		}
		return scores[i].score > scores[j].score
	})
	if len(scores) == 0 || scores[0].score == 0 {
		return nil
	}
	minimum := int(math.Ceil(float64(scores[0].score) * 0.5))
	selected := make([]string, 0, evaluationRelevantContentLimit)
	for _, item := range scores {
		if item.score < minimum || len(selected) == evaluationRelevantContentLimit {
			break
		}
		selected = append(selected, item.content)
	}
	return selected
}

func normalizeEvaluationAnchorText(value string) string {
	value = evaluationMarkdownImagePattern.ReplaceAllString(value, " ")
	value = evaluationURLPattern.ReplaceAllString(value, " ")
	var builder strings.Builder
	for _, char := range strings.ToLower(value) {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func countSharedEvaluationAnchors(left map[string]struct{}, right map[string]struct{}) int {
	if len(left) > len(right) {
		left, right = right, left
	}
	matched := 0
	for anchor := range left {
		if _, ok := right[anchor]; ok {
			matched++
		}
	}
	return matched
}

func isUniqueEvaluationAnchorScore(best int, second int) bool {
	if best < evaluationAnchorMinScore {
		return false
	}
	if second == 0 {
		return true
	}
	return best-second >= evaluationAnchorMinGap && best*4 >= second*5
}

func evaluationAssociationConfidence(best int, second int) float64 {
	strength := math.Min(1, float64(best)/40)
	separation := 1.0
	if second > 0 {
		separation = float64(best-second) / float64(best)
	}
	return math.Round((strength*0.6+separation*0.4)*1000) / 1000
}

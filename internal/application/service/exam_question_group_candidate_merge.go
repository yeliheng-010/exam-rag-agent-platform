package service

import (
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var anyQuestionNumberPattern = regexp.MustCompile(`\d{1,2}`)

func mergeQuestionGroupCandidates(candidates []*types.ExamQuestionGroupDraftCandidate) []*types.ExamQuestionGroupDraftCandidate {
	merged := make([]*types.ExamQuestionGroupDraftCandidate, 0, len(candidates))
	positions := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		key := canonicalQuestionGroupCandidateKey(candidate)
		position, exists := positions[key]
		if key == "" || !exists {
			if key != "" {
				positions[key] = len(merged)
			}
			merged = append(merged, candidate)
			continue
		}
		merged[position] = mergeQuestionGroupCandidate(merged[position], candidate)
	}
	return merged
}

func canonicalQuestionGroupCandidateKey(candidate *types.ExamQuestionGroupDraftCandidate) string {
	if candidate == nil {
		return ""
	}
	if len(candidate.Questions) > 0 {
		if number := firstQuestionNumber(candidate.Questions[0].QuestionNo); number != "" {
			return "question:" + number
		}
	}
	if number := firstQuestionNumber(candidate.GroupNo); number != "" {
		return "question:" + number
	}
	return strings.ToLower(strings.TrimSpace(candidate.GroupNo + "|" + candidate.Title))
}

func mergeQuestionGroupCandidate(current *types.ExamQuestionGroupDraftCandidate, incoming *types.ExamQuestionGroupDraftCandidate) *types.ExamQuestionGroupDraftCandidate {
	primary, secondary := preferredQuestionGroupCandidate(current, incoming)
	if primary == nil {
		return secondary
	}
	for _, question := range secondary.Questions {
		primary.Questions = mergeQuestionGroupQuestion(primary.Questions, question)
	}
	primary.SourceChunkIDs = mergeQuestionGroupUniqueStrings(primary.SourceChunkIDs, secondary.SourceChunkIDs)
	primary.Assets = mergeQuestionGroupAssets(primary.Assets, secondary.Assets)
	return primary
}

func preferredQuestionGroupCandidate(first *types.ExamQuestionGroupDraftCandidate, second *types.ExamQuestionGroupDraftCandidate) (*types.ExamQuestionGroupDraftCandidate, *types.ExamQuestionGroupDraftCandidate) {
	if first == nil {
		return second, first
	}
	if second == nil {
		return first, second
	}
	if len(second.Questions) > len(first.Questions) ||
		(len(second.Questions) == len(first.Questions) && questionGroupCandidateCompleteness(second) > questionGroupCandidateCompleteness(first)) {
		return second, first
	}
	return first, second
}

func mergeQuestionGroupQuestion(existing []types.ExamQuestionGroupDraftQuestionCandidate, candidate types.ExamQuestionGroupDraftQuestionCandidate) []types.ExamQuestionGroupDraftQuestionCandidate {
	key := canonicalSubquestionKey(candidate)
	for index := range existing {
		if canonicalSubquestionKey(existing[index]) != key {
			continue
		}
		if questionGroupQuestionCompleteness(candidate) > questionGroupQuestionCompleteness(existing[index]) {
			existing[index] = candidate
		}
		return existing
	}
	return append(existing, candidate)
}

func canonicalSubquestionKey(question types.ExamQuestionGroupDraftQuestionCandidate) string {
	value := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(question.QuestionNo), " ", ""))
	if value != "" {
		return value
	}
	return strings.ToLower(strings.TrimSpace(question.Stem))
}

func questionGroupCandidateCompleteness(candidate *types.ExamQuestionGroupDraftCandidate) int {
	if candidate == nil {
		return -1
	}
	score := len(candidate.MaterialText) + len(candidate.Assets)*20 + len(candidate.SourceChunkIDs)*5
	for _, question := range candidate.Questions {
		score += questionGroupQuestionCompleteness(question)
	}
	return score
}

func questionGroupQuestionCompleteness(question types.ExamQuestionGroupDraftQuestionCandidate) int {
	answer := draftAnswerText(question.Answer)
	return len(question.Stem) + len(question.Explanation)*2 + len(question.Options)*20 + len(answer)*20 + len(question.SourceChunkIDs)*5
}

func mergeQuestionGroupUniqueStrings(first []string, second []string) []string {
	seen := make(map[string]bool, len(first)+len(second))
	merged := make([]string, 0, len(first)+len(second))
	for _, values := range [][]string{first, second} {
		for _, value := range values {
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			merged = append(merged, value)
		}
	}
	return merged
}

func mergeQuestionGroupAssets(first []types.ExamQuestionGroupDraftAssetCandidate, second []types.ExamQuestionGroupDraftAssetCandidate) []types.ExamQuestionGroupDraftAssetCandidate {
	seen := make(map[string]bool, len(first)+len(second))
	merged := make([]types.ExamQuestionGroupDraftAssetCandidate, 0, len(first)+len(second))
	for _, assets := range [][]types.ExamQuestionGroupDraftAssetCandidate{first, second} {
		for _, asset := range assets {
			key := asset.StorageURI + "|" + asset.SourceChunkID + "|" + asset.AltText
			if seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, asset)
		}
	}
	return merged
}

func firstQuestionNumber(value string) string {
	return anyQuestionNumberPattern.FindString(strings.TrimSpace(value))
}

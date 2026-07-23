package searchutil

import "strings"

func applyRetrievalGold(
	result *ExamContextRetrievalEvalResult,
	evalCase ExamContextRetrievalEvalCase,
	resolution *ExamContextResolution,
) {
	if result == nil || resolution == nil {
		return
	}
	retrieved := rankedItemsOrLegacy(
		resolution.RetrievedItems, resolution.RetrievedChunkIDs, resolution.RetrievedContents,
	)
	candidates := rankedItemsOrLegacy(
		resolution.CandidateItems, resolution.CandidateChunkIDs, nil,
	)
	if len(candidates) == 0 {
		candidates = retrieved
	}
	strict := matchRetrievalGold(retrieved, evalCase)
	candidate := matchRetrievalGold(candidates, evalCase)
	result.MatchedChunkIDs, result.MissingChunkIDs = strict.matchedChunkIDs, strict.missingChunkIDs
	result.MatchedRetrievalPhrases = strict.matchedPhrases
	result.MissingRetrievalPhrases = strict.missingPhrases
	result.RetrievalScore, result.RetrievalPassed = strict.score, strict.passed
	result.CandidateRetrievalScore = candidate.score
	result.CandidateRetrievalPassed = candidate.passed
	result.FirstRelevantRank = strict.firstRelevantRank
	if result.FirstRelevantRank > 0 {
		result.ReciprocalRank = 1 / float64(result.FirstRelevantRank)
	}
}

func matchExamEvalPhrases(content string, phrases []string) ([]string, []string) {
	var matched, missing []string
	for _, phrase := range cleanEvalPhrases(phrases) {
		if strings.Contains(content, phrase) {
			matched = append(matched, phrase)
		} else {
			missing = append(missing, phrase)
		}
	}
	return matched, missing
}

func hasRetrievalGold(evalCase ExamContextRetrievalEvalCase) bool {
	return len(cleanEvalPhrases(evalCase.ExpectedChunkIDs)) > 0 ||
		len(cleanEvalPhrases(evalCase.RequiredRetrievalPhrases)) > 0
}

func matchEvalItems(haystack []string, needles []string) ([]string, []string) {
	available := make(map[string]bool)
	for _, item := range cleanEvalPhrases(haystack) {
		available[item] = true
	}
	var matched, missing []string
	for _, item := range cleanEvalPhrases(needles) {
		if available[item] {
			matched = append(matched, item)
		} else {
			missing = append(missing, item)
		}
	}
	return matched, missing
}

func missingEvalItems(expected []string, actual []string) []string {
	_, missing := matchEvalItems(actual, expected)
	return missing
}

func cleanEvalPhrases(phrases []string) []string {
	out := make([]string, 0, len(phrases))
	seen := make(map[string]bool, len(phrases))
	for _, phrase := range phrases {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" || seen[phrase] {
			continue
		}
		seen[phrase] = true
		out = append(out, phrase)
	}
	return out
}

func examEvalScore(matched int, total int) float64 {
	if total == 0 {
		return 1
	}
	return float64(matched) / float64(total)
}

func examEvalHitRate(passed int, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(passed) / float64(total)
}

func examEvalAverage(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

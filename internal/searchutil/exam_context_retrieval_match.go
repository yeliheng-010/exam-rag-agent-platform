package searchutil

import "strings"

type retrievalGoldMatch struct {
	matchedChunkIDs   []string
	missingChunkIDs   []string
	matchedPhrases    []string
	missingPhrases    []string
	score             float64
	passed            bool
	firstRelevantRank int
}

func matchRetrievalGold(
	items []ExamContextRankedItem,
	evalCase ExamContextRetrievalEvalCase,
) retrievalGoldMatch {
	expectedIDs := cleanEvalPhrases(evalCase.ExpectedChunkIDs)
	phrases := cleanEvalPhrases(evalCase.RequiredRetrievalPhrases)
	matchedIDs, missingIDs := matchRankedItemIDs(items, expectedIDs)
	matchedPhrases, missingPhrases := matchRankedItemPhrases(items, phrases)
	totalGold := len(expectedIDs) + len(phrases)
	totalMatched := len(matchedIDs) + len(matchedPhrases)
	return retrievalGoldMatch{
		matchedChunkIDs: matchedIDs, missingChunkIDs: missingIDs,
		matchedPhrases: matchedPhrases, missingPhrases: missingPhrases,
		score:             examEvalScore(totalMatched, totalGold),
		passed:            totalGold > 0 && len(missingIDs) == 0 && len(missingPhrases) == 0,
		firstRelevantRank: firstRelevantItemRank(items, expectedIDs, phrases),
	}
}

func matchRankedItemIDs(items []ExamContextRankedItem, expected []string) ([]string, []string) {
	available := make(map[string]bool)
	for _, item := range items {
		for _, id := range cleanEvalPhrases(item.ChunkIDs) {
			available[id] = true
		}
	}
	var matched, missing []string
	for _, id := range expected {
		if available[id] {
			matched = append(matched, id)
		} else {
			missing = append(missing, id)
		}
	}
	return matched, missing
}

func matchRankedItemPhrases(items []ExamContextRankedItem, phrases []string) ([]string, []string) {
	var matched, missing []string
	for _, phrase := range phrases {
		if rankedItemsContainPhrase(items, phrase) {
			matched = append(matched, phrase)
		} else {
			missing = append(missing, phrase)
		}
	}
	return matched, missing
}

func rankedItemsContainPhrase(items []ExamContextRankedItem, phrase string) bool {
	for _, item := range items {
		for _, content := range item.Contents {
			if strings.Contains(content, phrase) {
				return true
			}
		}
	}
	return false
}

func firstRelevantItemRank(items []ExamContextRankedItem, expectedIDs, phrases []string) int {
	expectedSet := make(map[string]bool, len(expectedIDs))
	for _, id := range expectedIDs {
		expectedSet[id] = true
	}
	for index, item := range items {
		if rankedItemMatchesGold(item, expectedSet, phrases) {
			if item.Rank > 0 {
				return item.Rank
			}
			return index + 1
		}
	}
	return 0
}

func rankedItemMatchesGold(item ExamContextRankedItem, expectedIDs map[string]bool, phrases []string) bool {
	for _, id := range cleanEvalPhrases(item.ChunkIDs) {
		if expectedIDs[id] {
			return true
		}
	}
	for _, content := range item.Contents {
		for _, phrase := range phrases {
			if strings.Contains(content, phrase) {
				return true
			}
		}
	}
	return false
}

func rankedItemsOrLegacy(
	items []ExamContextRankedItem,
	ids []string,
	contents []string,
) []ExamContextRankedItem {
	if len(items) > 0 {
		return items
	}
	count := max(len(ids), len(contents))
	out := make([]ExamContextRankedItem, 0, count)
	for index := 0; index < count; index++ {
		item := ExamContextRankedItem{Rank: index + 1, LocalRank: index + 1}
		if index < len(ids) {
			item.ChunkIDs = []string{ids[index]}
		}
		if index < len(contents) {
			item.Contents = []string{contents[index]}
		}
		out = append(out, item)
	}
	return out
}

func flattenRankedItemIDs(items []ExamContextRankedItem) []string {
	var ids []string
	for _, item := range items {
		ids = append(ids, item.ChunkIDs...)
	}
	return cleanEvalPhrases(ids)
}

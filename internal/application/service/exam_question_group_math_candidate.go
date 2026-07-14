package service

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	answerOnlyQuestionPattern   = regexp.MustCompile(`【\s*\d{1,2}\s*题答案\s*】`)
	mathSubquestionLabelPattern = regexp.MustCompile(`^[\s(（]*(\d+|[ivxIVX]+)[\s)）]*$`)
)

func constrainMathBatchCandidates(candidates []*types.ExamQuestionGroupDraftCandidate, batch questionGroupExtractionBatch) []*types.ExamQuestionGroupDraftCandidate {
	targets := intSet(batch.TargetQuestionNumbers)
	if len(targets) == 0 {
		return candidates
	}
	bodyChunks := append(append([]*types.Chunk{}, batch.CoreChunks...), batch.ContextChunks...)
	bodyIDs := questionGroupChunkIDSet(bodyChunks)
	answerIDs := questionGroupChunkIDSet(batch.AnswerChunks)
	constrained := make([]*types.ExamQuestionGroupDraftCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		constrained = append(constrained, constrainMathCandidate(candidate, targets, bodyIDs, answerIDs)...)
	}
	return constrained
}

func constrainMathCandidate(candidate *types.ExamQuestionGroupDraftCandidate, targets map[int]bool, bodyIDs map[string]bool, answerIDs map[string]bool) []*types.ExamQuestionGroupDraftCandidate {
	if candidate == nil {
		return nil
	}
	candidateTarget := inferMathCandidateTarget(candidate, targets)
	questionsByNumber := make(map[int][]types.ExamQuestionGroupDraftQuestionCandidate)
	for _, question := range candidate.Questions {
		if answerOnlyQuestionPattern.MatchString(question.Stem) {
			continue
		}
		number, normalized, ok := normalizeMathBatchQuestionNo(question.QuestionNo, candidateTarget, targets)
		if !ok {
			continue
		}
		question.QuestionNo = normalized
		question.SourceChunkIDs = filterQuestionGroupSourceIDs(question.SourceChunkIDs, bodyIDs)
		if len(question.SourceChunkIDs) == 0 {
			question.SourceChunkIDs = sortedQuestionGroupSourceIDs(bodyIDs)
		}
		questionsByNumber[number] = append(questionsByNumber[number], question)
	}
	return splitConstrainedMathCandidate(candidate, questionsByNumber, bodyIDs, answerIDs)
}

func splitConstrainedMathCandidate(candidate *types.ExamQuestionGroupDraftCandidate, questions map[int][]types.ExamQuestionGroupDraftQuestionCandidate, bodyIDs map[string]bool, answerIDs map[string]bool) []*types.ExamQuestionGroupDraftCandidate {
	numbers := make([]int, 0, len(questions))
	for number := range questions {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	out := make([]*types.ExamQuestionGroupDraftCandidate, 0, len(numbers))
	for _, number := range numbers {
		clone := *candidate
		clone.GroupNo = strconv.Itoa(number)
		clone.Title = fmt.Sprintf("Question %d", number)
		clone.Questions = questions[number]
		for index := range clone.Questions {
			clone.Questions[index].OrderInGroup = index + 1
		}
		clone.SourceChunkIDs = filterQuestionGroupSourceIDs(candidate.SourceChunkIDs, bodyIDs)
		if len(clone.SourceChunkIDs) == 0 {
			clone.SourceChunkIDs = sortedQuestionGroupSourceIDs(bodyIDs)
		}
		clone.Assets = filterMathBatchAssets(candidate.Assets, answerIDs)
		out = append(out, &clone)
	}
	return out
}

func inferMathCandidateTarget(candidate *types.ExamQuestionGroupDraftCandidate, targets map[int]bool) int {
	for _, value := range []string{candidate.Title, candidate.GroupNo} {
		if number, err := strconv.Atoi(firstQuestionNumber(value)); err == nil && targets[number] {
			return number
		}
	}
	if len(targets) == 1 {
		for number := range targets {
			return number
		}
	}
	return 0
}

func normalizeMathBatchQuestionNo(value string, candidateTarget int, targets map[int]bool) (int, string, bool) {
	value = strings.TrimSpace(value)
	if number, err := strconv.Atoi(firstQuestionNumber(value)); err == nil && targets[number] {
		return number, value, true
	}
	match := mathSubquestionLabelPattern.FindStringSubmatch(value)
	if candidateTarget < 15 || len(match) < 2 {
		return 0, "", false
	}
	return candidateTarget, fmt.Sprintf("%d(%s)", candidateTarget, strings.ToLower(match[1])), true
}

func filterQuestionGroupSourceIDs(ids []string, allowed map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if allowed[id] {
			out = append(out, id)
		}
	}
	return out
}

func filterMathBatchAssets(assets []types.ExamQuestionGroupDraftAssetCandidate, answerIDs map[string]bool) []types.ExamQuestionGroupDraftAssetCandidate {
	out := make([]types.ExamQuestionGroupDraftAssetCandidate, 0, len(assets))
	for _, asset := range assets {
		if !answerIDs[asset.SourceChunkID] {
			out = append(out, asset)
		}
	}
	return out
}

func questionGroupChunkIDSet(chunks []*types.Chunk) map[string]bool {
	ids := make(map[string]bool, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil && chunk.ID != "" {
			ids[chunk.ID] = true
		}
	}
	return ids
}

func sortedQuestionGroupSourceIDs(ids map[string]bool) []string {
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

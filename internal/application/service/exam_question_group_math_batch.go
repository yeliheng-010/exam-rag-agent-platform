package service

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	gaokaoMathMaxQuestionsPerBatch = 3
	gaokaoMathMaxCoreChunks        = 4
	gaokaoMathFallbackChunkCount   = 3
)

var (
	bodyQuestionNumberPattern   = regexp.MustCompile(`(?m)(?:^|\n)\s*(\d{1,2})\s*[.．、]\s+`)
	answerQuestionNumberPattern = regexp.MustCompile(`【\s*(\d{1,2})\s*题答案\s*】`)
)

type questionGroupExtractionBatch struct {
	CoreChunks            []*types.Chunk
	ContextChunks         []*types.Chunk
	AnswerChunks          []*types.Chunk
	TargetQuestionNumbers []int
}

func (b questionGroupExtractionBatch) allChunks() []*types.Chunk {
	all := make([]*types.Chunk, 0, len(b.CoreChunks)+len(b.ContextChunks)+len(b.AnswerChunks))
	all = append(all, b.CoreChunks...)
	all = append(all, b.ContextChunks...)
	all = append(all, b.AnswerChunks...)
	return all
}

type mathQuestionStart struct {
	Number        int
	ChunkPosition int
}

func buildQuestionGroupExtractionBatches(strategyCode string, chunks []*types.Chunk) []questionGroupExtractionBatch {
	if strategyCode != gaokaoMathQuestionGroupStrategy {
		return []questionGroupExtractionBatch{{CoreChunks: chunks}}
	}
	body, answers := splitMathBodyAndAnswerChunks(sortedQuestionGroupChunks(chunks))
	if len(body) == 0 {
		return []questionGroupExtractionBatch{{CoreChunks: chunks}}
	}
	starts := findMathQuestionStarts(body)
	if len(starts) == 0 {
		return buildFallbackMathBatches(body, answers)
	}
	return buildQuestionBoundaryMathBatches(body, answers, starts)
}

func buildQuestionBoundaryMathBatches(body []*types.Chunk, answers []*types.Chunk, starts []mathQuestionStart) []questionGroupExtractionBatch {
	batches := make([]questionGroupExtractionBatch, 0, len(starts))
	for first := 0; first < len(starts); {
		last := first
		for last+1 < len(starts) && last-first+1 < gaokaoMathMaxQuestionsPerBatch {
			candidateEnd := mathBatchCoreEnd(starts, last+1, len(body))
			if candidateEnd-starts[first].ChunkPosition > gaokaoMathMaxCoreChunks {
				break
			}
			last++
		}
		coreStart := starts[first].ChunkPosition
		coreEnd := mathBatchCoreEnd(starts, last, len(body))
		targets := mathBatchTargets(starts[first : last+1])
		batch := questionGroupExtractionBatch{
			CoreChunks:            body[coreStart:coreEnd],
			ContextChunks:         mathBatchContext(body, coreStart, coreEnd),
			AnswerChunks:          projectMathAnswerChunks(answers, targets),
			TargetQuestionNumbers: targets,
		}
		batches = append(batches, batch)
		first = last + 1
	}
	return batches
}

func mathBatchCoreEnd(starts []mathQuestionStart, last int, bodyLen int) int {
	end := bodyLen
	if last+1 < len(starts) {
		end = starts[last+1].ChunkPosition
	}
	end = max(end, starts[last].ChunkPosition+1)
	return min(end, bodyLen)
}

func mathBatchTargets(starts []mathQuestionStart) []int {
	targets := make([]int, 0, len(starts))
	for _, start := range starts {
		targets = append(targets, start.Number)
	}
	return targets
}

func mathBatchContext(body []*types.Chunk, start int, end int) []*types.Chunk {
	contextChunks := make([]*types.Chunk, 0, 2)
	if start > 0 {
		contextChunks = append(contextChunks, body[start-1])
	}
	if end < len(body) {
		contextChunks = append(contextChunks, body[end])
	}
	return contextChunks
}

func buildFallbackMathBatches(body []*types.Chunk, answers []*types.Chunk) []questionGroupExtractionBatch {
	batches := make([]questionGroupExtractionBatch, 0, (len(body)+gaokaoMathFallbackChunkCount-1)/gaokaoMathFallbackChunkCount)
	for start := 0; start < len(body); start += gaokaoMathFallbackChunkCount {
		end := min(start+gaokaoMathFallbackChunkCount, len(body))
		targets := uniqueQuestionNumbers(body[start:end])
		batches = append(batches, questionGroupExtractionBatch{
			CoreChunks:            body[start:end],
			ContextChunks:         mathBatchContext(body, start, end),
			AnswerChunks:          projectMathAnswerChunks(answers, targets),
			TargetQuestionNumbers: targets,
		})
	}
	return batches
}

func findMathQuestionStarts(chunks []*types.Chunk) []mathQuestionStart {
	starts := make([]mathQuestionStart, 0)
	lastNumber := 0
	for position, chunk := range chunks {
		if chunk == nil {
			continue
		}
		for _, number := range extractQuestionNumbers(bodyQuestionNumberPattern, chunk.Content) {
			if number <= lastNumber || number > 99 {
				continue
			}
			if lastNumber == 0 && number != 1 {
				continue
			}
			if lastNumber > 0 && number > lastNumber+3 {
				continue
			}
			starts = append(starts, mathQuestionStart{Number: number, ChunkPosition: position})
			lastNumber = number
		}
	}
	return starts
}

func projectMathAnswerChunks(chunks []*types.Chunk, targets []int) []*types.Chunk {
	targetSet := intSet(targets)
	selected := make([]*types.Chunk, 0)
	activeQuestion := 0
	for _, chunk := range chunks {
		projected, active := projectMathAnswerChunk(chunk, targetSet, activeQuestion)
		activeQuestion = active
		if projected != nil {
			selected = append(selected, projected)
		}
	}
	return selected
}

func projectMathAnswerChunk(chunk *types.Chunk, targets map[int]bool, activeQuestion int) (*types.Chunk, int) {
	if chunk == nil {
		return nil, activeQuestion
	}
	content := chunk.Content
	matches := answerQuestionNumberPattern.FindAllStringSubmatchIndex(content, -1)
	parts := make([]string, 0)
	if len(matches) == 0 {
		if targets[activeQuestion] && strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	} else {
		if matches[0][0] > 0 && targets[activeQuestion] {
			parts = append(parts, content[:matches[0][0]])
		}
		for index, match := range matches {
			activeQuestion, _ = strconv.Atoi(content[match[2]:match[3]])
			end := len(content)
			if index+1 < len(matches) {
				end = matches[index+1][0]
			}
			if targets[activeQuestion] {
				parts = append(parts, content[match[0]:end])
			}
		}
	}
	projectedContent := strings.TrimSpace(strings.Join(parts, "\n"))
	if projectedContent == "" {
		return nil, activeQuestion
	}
	projected := *chunk
	projected.Content = projectedContent
	return &projected, activeQuestion
}

func uniqueQuestionNumbers(chunks []*types.Chunk) []int {
	seen := map[int]bool{}
	numbers := make([]int, 0)
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		for _, number := range extractQuestionNumbers(bodyQuestionNumberPattern, chunk.Content) {
			if !seen[number] {
				seen[number] = true
				numbers = append(numbers, number)
			}
		}
	}
	return numbers
}

func extractQuestionNumbers(pattern *regexp.Regexp, content string) []int {
	matches := pattern.FindAllStringSubmatch(content, -1)
	numbers := make([]int, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if number, err := strconv.Atoi(match[1]); err == nil && number > 0 {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

func intSet(values []int) map[int]bool {
	set := make(map[int]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func sortedQuestionGroupChunks(chunks []*types.Chunk) []*types.Chunk {
	ordered := append([]*types.Chunk{}, chunks...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i] == nil {
			return false
		}
		if ordered[j] == nil {
			return true
		}
		return ordered[i].ChunkIndex < ordered[j].ChunkIndex
	})
	return ordered
}

func splitMathBodyAndAnswerChunks(chunks []*types.Chunk) ([]*types.Chunk, []*types.Chunk) {
	for index, chunk := range chunks {
		if chunk != nil && isLikelyMathAnswerSectionStart(chunk.Content) {
			return chunks[:index], chunks[index:]
		}
	}
	return chunks, nil
}

func isLikelyMathAnswerSectionStart(content string) bool {
	normalized := strings.ToLower(content)
	return answerQuestionNumberPattern.MatchString(content) ||
		strings.Contains(content, "参考答案") ||
		strings.Contains(content, "答案及解析") ||
		strings.Contains(normalized, "answer key")
}

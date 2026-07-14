package searchutil

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const examQuestionContextRuneLimit = 9000

var markdownPassageMarker = map[string]*regexp.Regexp{
	"A": regexp.MustCompile(`(?m)(^|\s)\*{0,2}A\*{0,2}(\s|$)`),
	"B": regexp.MustCompile(`(?m)(^|\s)\*{0,2}B\*{0,2}(\s|$)`),
	"C": regexp.MustCompile(`(?m)(^|\s)\*{0,2}C\*{0,2}(\s|$)`),
	"D": regexp.MustCompile(`(?m)(^|\s)\*{0,2}D\*{0,2}(\s|$)`),
}

var examReadingAnswerRanges = map[string]struct {
	current string
	next    string
}{
	"A": {current: "【21~23题答案】", next: "【24~27题答案】"},
	"B": {current: "【24~27题答案】", next: "【28~31题答案】"},
	"C": {current: "【28~31题答案】", next: "【32~35题答案】"},
	"D": {current: "【32~35题答案】", next: "【36~40题答案】"},
}

// ExamQuestionContextBundle is a compact synthetic retrieval result that keeps
// a passage body and its distant answer section together for exam-paper RAG.
type ExamQuestionContextBundle struct {
	Label          string
	Content        string
	BodyChunks     []*types.Chunk
	AnswerChunks   []*types.Chunk
	SourceChunkIDs []string
}

// ShouldEnrichExamQuestionContext returns true for exam-paper questions where
// the user asks for a reading passage and/or its answers.
func ShouldEnrichExamQuestionContext(query string) bool {
	q := compactExamQuery(query)
	if q == "" {
		return false
	}

	hasReading := strings.Contains(q, "阅读") ||
		strings.Contains(q, "reading") ||
		strings.Contains(q, "passage")
	if !hasReading {
		return false
	}

	hasContextNeed := strings.Contains(q, "答案") ||
		strings.Contains(q, "answer") ||
		strings.Contains(q, "原文") ||
		strings.Contains(q, "全文") ||
		strings.Contains(q, "题目")
	if !hasContextNeed {
		return false
	}

	return strings.Contains(q, "第一篇") ||
		strings.Contains(q, "第二篇") ||
		strings.Contains(q, "第三篇") ||
		strings.Contains(q, "第四篇") ||
		strings.Contains(q, "a篇") ||
		strings.Contains(q, "b篇") ||
		strings.Contains(q, "c篇") ||
		strings.Contains(q, "d篇") ||
		strings.Contains(q, "passage1") ||
		strings.Contains(q, "passagea") ||
		strings.Contains(q, "firstreading") ||
		strings.Contains(q, "原文") ||
		strings.Contains(q, "答案")
}

// BuildExamQuestionContextBundle selects the requested reading passage and its
// answer key from a parsed exam document. It currently focuses on objective
// reading passages A-D, which matches Gaokao-style English papers.
func BuildExamQuestionContextBundle(query string, chunks []*types.Chunk) *ExamQuestionContextBundle {
	if !ShouldEnrichExamQuestionContext(query) {
		return nil
	}

	textChunks := sortedTextChunks(chunks)
	if len(textChunks) == 0 {
		return nil
	}

	label := examPassageLabelFromQuery(query)
	bodyChunks := selectExamPassageBodyChunks(textChunks, label)
	if len(bodyChunks) == 0 {
		return nil
	}

	answerChunks := selectExamAnswerChunks(textChunks, label)
	answerText := extractExamAnswerText(label, answerChunks)

	bodyText := strings.TrimSpace(MergeTextChunks(bodyChunks, "\n"))
	if bodyText == "" {
		return nil
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[Exam Passage %s]\n", label))
	builder.WriteString("[原文]\n")
	builder.WriteString(bodyText)
	if answerText != "" {
		builder.WriteString("\n\n[答案]\n")
		builder.WriteString(answerText)
	}

	return &ExamQuestionContextBundle{
		Label:        label,
		Content:      trimRunes(strings.TrimSpace(builder.String()), examQuestionContextRuneLimit),
		BodyChunks:   bodyChunks,
		AnswerChunks: answerChunks,
	}
}

func compactExamQuery(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), ""))
}

func examPassageLabelFromQuery(query string) string {
	q := compactExamQuery(query)
	switch {
	case strings.Contains(q, "第四篇"), strings.Contains(q, "d篇"), strings.Contains(q, "passaged"):
		return "D"
	case strings.Contains(q, "第三篇"), strings.Contains(q, "c篇"), strings.Contains(q, "passagec"):
		return "C"
	case strings.Contains(q, "第二篇"), strings.Contains(q, "b篇"), strings.Contains(q, "passageb"), strings.Contains(q, "passage2"):
		return "B"
	default:
		return "A"
	}
}

func sortedTextChunks(chunks []*types.Chunk) []*types.Chunk {
	out := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk == nil || chunk.Content == "" {
			continue
		}
		if chunk.ChunkType != "" && chunk.ChunkType != types.ChunkTypeText {
			continue
		}
		out = append(out, chunk)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ChunkIndex == out[j].ChunkIndex {
			return out[i].StartAt < out[j].StartAt
		}
		return out[i].ChunkIndex < out[j].ChunkIndex
	})
	return out
}

func selectExamPassageBodyChunks(chunks []*types.Chunk, label string) []*types.Chunk {
	start := findPassageStart(chunks, label)
	if start < 0 {
		return nil
	}

	nextLabel := nextExamPassageLabel(label)
	end := len(chunks) - 1
	if nextLabel != "" {
		for i := start + 1; i < len(chunks); i++ {
			if isLikelyExamAnswerChunk(chunks[i].Content) {
				continue
			}
			if hasExamPassageMarker(chunks[i].Content, nextLabel) {
				end = i
				break
			}
		}
	}

	const maxBodyChunks = 10
	selected := make([]*types.Chunk, 0, maxBodyChunks)
	for i := start; i <= end && len(selected) < maxBodyChunks; i++ {
		if isLikelyExamAnswerChunk(chunks[i].Content) {
			continue
		}
		selected = append(selected, chunks[i])
	}
	return selected
}

func findPassageStart(chunks []*types.Chunk, label string) int {
	for i, chunk := range chunks {
		if isLikelyExamAnswerChunk(chunk.Content) {
			continue
		}
		if hasExamPassageMarker(chunk.Content, label) {
			return i
		}
	}
	if label != "A" {
		return -1
	}
	for i, chunk := range chunks {
		if isLikelyExamAnswerChunk(chunk.Content) {
			continue
		}
		if strings.Contains(chunk.Content, "阅读下列短文") ||
			strings.Contains(chunk.Content, "阅读理解") {
			if i+1 < len(chunks) {
				return i + 1
			}
			return i
		}
	}
	return -1
}

func hasExamPassageMarker(content, label string) bool {
	re := markdownPassageMarker[label]
	return re != nil && re.MatchString(content)
}

func nextExamPassageLabel(label string) string {
	switch label {
	case "A":
		return "B"
	case "B":
		return "C"
	case "C":
		return "D"
	default:
		return ""
	}
}

func selectExamAnswerChunks(chunks []*types.Chunk, label string) []*types.Chunk {
	rangeInfo, ok := examReadingAnswerRanges[label]
	if !ok {
		return nil
	}

	out := make([]*types.Chunk, 0, 2)
	for _, chunk := range chunks {
		content := chunk.Content
		if !isLikelyExamAnswerChunk(content) {
			continue
		}
		if strings.Contains(content, rangeInfo.current) ||
			(hasExamPassageMarker(content, label) && strings.Contains(content, "【答案】")) {
			out = append(out, chunk)
		}
	}
	return out
}

func extractExamAnswerText(label string, chunks []*types.Chunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var combined strings.Builder
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		if combined.Len() > 0 {
			combined.WriteString("\n")
		}
		combined.WriteString(chunk.Content)
	}
	content := combined.String()

	startMarker := "**" + label + "**"
	if start := strings.Index(content, startMarker); start >= 0 {
		end := len(content)
		if nextLabel := nextExamPassageLabel(label); nextLabel != "" {
			nextMarker := "**" + nextLabel + "**"
			if rel := strings.Index(content[start+len(startMarker):], nextMarker); rel >= 0 {
				end = start + len(startMarker) + rel
			}
		}
		return strings.TrimSpace(content[start:end])
	}

	rangeInfo, ok := examReadingAnswerRanges[label]
	if !ok {
		return ""
	}
	start := strings.Index(content, rangeInfo.current)
	if start < 0 {
		return ""
	}
	end := len(content)
	if rangeInfo.next != "" {
		if rel := strings.Index(content[start+len(rangeInfo.current):], rangeInfo.next); rel >= 0 {
			end = start + len(rangeInfo.current) + rel
		}
	}
	return strings.TrimSpace(content[start:end])
}

func isLikelyExamAnswerChunk(content string) bool {
	if content == "" {
		return false
	}
	lowered := strings.ToLower(content)
	return strings.Contains(content, "【答案】") ||
		strings.Contains(content, "参考答案") ||
		strings.Contains(content, "题答案】") ||
		strings.Contains(lowered, "answer key") ||
		strings.Contains(lowered, "answers")
}

func trimRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "\n[truncated]"
}

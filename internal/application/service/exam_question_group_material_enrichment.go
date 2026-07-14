package service

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	questionBoundaryLinePattern = regexp.MustCompile(`^\s*\d{1,3}[\.\)、)]\s+`)
	passageLabelPattern         = regexp.MustCompile(`^\s*[A-D][\.\s]+`)
	spacePattern                = regexp.MustCompile(`\s+`)
)

func enrichQuestionGroupDraftCandidateMaterial(candidate *types.ExamQuestionGroupDraftCandidate, chunks []*types.Chunk) {
	if !shouldEnrichQuestionGroupMaterial(candidate) {
		return
	}
	sourceLines := extractQuestionGroupMaterialLines(candidate, chunks)
	if len(sourceLines) == 0 {
		return
	}
	material := mergeMissingQuestionGroupMaterialLines(candidate.MaterialText, sourceLines)
	if strings.TrimSpace(material) != "" {
		candidate.MaterialText = material
	}
}

func shouldEnrichQuestionGroupMaterial(candidate *types.ExamQuestionGroupDraftCandidate) bool {
	if candidate == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(candidate.GroupType), "reading_passage")
}

func extractQuestionGroupMaterialLines(candidate *types.ExamQuestionGroupDraftCandidate, chunks []*types.Chunk) []string {
	selected := selectQuestionGroupSourceChunks(candidate, chunks)
	if len(selected) == 0 {
		return nil
	}
	rawLines := flattenChunkMaterialLines(selected)
	start := questionGroupMaterialStartIndex(candidate, rawLines)
	if start < 0 {
		start = 0
	}
	return collectQuestionGroupMaterialLines(rawLines[start:])
}

func selectQuestionGroupSourceChunks(candidate *types.ExamQuestionGroupDraftCandidate, chunks []*types.Chunk) []*types.Chunk {
	sourceIDs := questionGroupCandidateSourceIDSet(candidate)
	if len(sourceIDs) == 0 {
		return sortQuestionGroupChunks(chunks)
	}
	minIndex, maxIndex, ok := questionGroupSourceChunkIndexRange(sourceIDs, chunks)
	if !ok {
		return sortQuestionGroupChunks(chunks)
	}
	return selectQuestionGroupChunkIndexWindow(chunks, minIndex-1, maxIndex)
}

func questionGroupCandidateSourceIDSet(candidate *types.ExamQuestionGroupDraftCandidate) map[string]bool {
	ids := map[string]bool{}
	if candidate == nil {
		return ids
	}
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id != "" {
			ids[id] = true
		}
	}
	for _, id := range candidate.SourceChunkIDs {
		add(id)
	}
	for _, question := range candidate.Questions {
		for _, id := range question.SourceChunkIDs {
			add(id)
		}
	}
	return ids
}

func questionGroupSourceChunkIndexRange(sourceIDs map[string]bool, chunks []*types.Chunk) (int, int, bool) {
	minIndex, maxIndex := 0, 0
	ok := false
	for _, chunk := range chunks {
		if chunk == nil || !sourceIDs[strings.TrimSpace(chunk.ID)] {
			continue
		}
		if !ok || chunk.ChunkIndex < minIndex {
			minIndex = chunk.ChunkIndex
		}
		if !ok || chunk.ChunkIndex > maxIndex {
			maxIndex = chunk.ChunkIndex
		}
		ok = true
	}
	return minIndex, maxIndex, ok
}

func selectQuestionGroupChunkIndexWindow(chunks []*types.Chunk, minIndex int, maxIndex int) []*types.Chunk {
	selected := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk == nil || chunk.ChunkIndex < minIndex || chunk.ChunkIndex > maxIndex {
			continue
		}
		selected = append(selected, chunk)
	}
	return sortQuestionGroupChunks(selected)
}

func sortQuestionGroupChunks(chunks []*types.Chunk) []*types.Chunk {
	selected := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil {
			selected = append(selected, chunk)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].ChunkIndex < selected[j].ChunkIndex
	})
	return selected
}

func flattenChunkMaterialLines(chunks []*types.Chunk) []string {
	lines := make([]string, 0)
	for _, chunk := range chunks {
		if chunk == nil {
			continue
		}
		for _, line := range strings.Split(chunk.Content, "\n") {
			line = cleanQuestionGroupMaterialLine(line)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return lines
}

func questionGroupMaterialStartIndex(candidate *types.ExamQuestionGroupDraftCandidate, lines []string) int {
	needles := questionGroupMaterialAnchors(candidate)
	for i, line := range lines {
		for _, needle := range needles {
			if materialLineContains(line, needle) || materialLineContains(needle, line) {
				return i
			}
		}
	}
	return -1
}

func questionGroupMaterialAnchors(candidate *types.ExamQuestionGroupDraftCandidate) []string {
	if candidate == nil {
		return nil
	}
	anchors := make([]string, 0, 4)
	add := func(value string) {
		value = materialLineKey(value)
		if len([]rune(value)) > 1 {
			anchors = append(anchors, value)
		}
	}
	add(candidate.Title)
	add(candidate.GroupNo)
	for _, line := range strings.Split(candidate.MaterialText, "\n") {
		add(line)
		if len(anchors) >= 4 {
			break
		}
	}
	return anchors
}

func collectQuestionGroupMaterialLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if shouldStopQuestionGroupMaterial(line, len(out) > 0) {
			break
		}
		if shouldSkipQuestionGroupMaterialLine(line) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func shouldStopQuestionGroupMaterial(line string, hasMaterial bool) bool {
	if !hasMaterial {
		return false
	}
	return questionBoundaryLinePattern.MatchString(line) ||
		strings.HasPrefix(strings.ToLower(line), "answers:") ||
		strings.Contains(line, "【答案】")
}

func shouldSkipQuestionGroupMaterialLine(line string) bool {
	key := materialLineKey(line)
	if key == "" {
		return true
	}
	if len([]rune(key)) == 1 && key >= "a" && key <= "d" {
		return true
	}
	return strings.Contains(key, "read the following") ||
		strings.Contains(line, "阅读下列短文")
}

func mergeMissingQuestionGroupMaterialLines(material string, sourceLines []string) string {
	result := splitQuestionGroupMaterialLines(material)
	if len(result) == 0 {
		return strings.Join(sourceLines, "\n")
	}
	for i, line := range sourceLines {
		if questionGroupMaterialHasLine(result, line) {
			continue
		}
		insertAt := missingQuestionGroupMaterialInsertIndex(result, sourceLines, i)
		result = insertQuestionGroupMaterialLine(result, insertAt, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func splitQuestionGroupMaterialLines(material string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(material, "\n") {
		line = cleanQuestionGroupMaterialLine(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func missingQuestionGroupMaterialInsertIndex(result []string, sourceLines []string, sourceIndex int) int {
	for i := sourceIndex + 1; i < len(sourceLines); i++ {
		if idx := questionGroupMaterialLineIndex(result, sourceLines[i]); idx >= 0 {
			return idx
		}
	}
	for i := sourceIndex - 1; i >= 0; i-- {
		if idx := questionGroupMaterialLineIndex(result, sourceLines[i]); idx >= 0 {
			return idx + 1
		}
	}
	return len(result)
}

func insertQuestionGroupMaterialLine(lines []string, index int, line string) []string {
	if index < 0 || index > len(lines) {
		index = len(lines)
	}
	lines = append(lines, "")
	copy(lines[index+1:], lines[index:])
	lines[index] = line
	return lines
}

func questionGroupMaterialHasLine(lines []string, sourceLine string) bool {
	return questionGroupMaterialLineIndex(lines, sourceLine) >= 0
}

func questionGroupMaterialLineIndex(lines []string, sourceLine string) int {
	key := materialLineKey(sourceLine)
	if key == "" {
		return -1
	}
	for i, line := range lines {
		if materialLineContains(line, key) || materialLineContains(key, line) {
			return i
		}
	}
	return -1
}

func cleanQuestionGroupMaterialLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "*`")
	line = strings.TrimSpace(strings.ReplaceAll(line, "**", ""))
	return line
}

func materialLineContains(line string, key string) bool {
	lineKey := materialLineKey(line)
	key = materialLineKey(key)
	return lineKey != "" && key != "" && strings.Contains(lineKey, key)
}

func materialLineKey(line string) string {
	line = cleanQuestionGroupMaterialLine(line)
	line = passageLabelPattern.ReplaceAllString(line, "")
	line = strings.ToLower(line)
	line = strings.ReplaceAll(line, "|", " ")
	line = spacePattern.ReplaceAllString(line, " ")
	return strings.TrimSpace(line)
}

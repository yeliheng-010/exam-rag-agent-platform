package examtext

import (
	"strings"
	"unicode"
)

const ExplanationSourceAutoBaseline = "auto_baseline"

type ExplanationOption struct {
	Key       string
	Content   string
	SortOrder int
}

func BuildBaselineExplanation(answer string, options []ExplanationOption, material string) string {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return ""
	}
	matches := matchingAnswerOptions(answer, options)
	answerLabel := answer
	if len(matches) > 0 {
		answerLabel = strings.Join(optionLabels(matches), "; ")
	}

	var builder strings.Builder
	builder.WriteString("Correct answer: ")
	builder.WriteString(answerLabel)
	builder.WriteString(".")
	if evidence := BestEvidenceSnippet(material, optionContents(matches)); evidence != "" {
		builder.WriteString(" Evidence: ")
		builder.WriteString(evidence)
	}
	return builder.String()
}

func BestEvidenceSnippet(material string, hints []string) string {
	candidates := evidenceCandidates(material)
	if len(candidates) == 0 || len(hints) == 0 {
		return ""
	}
	foldedHints := foldedSearchHints(hints)
	if len(foldedHints) == 0 {
		return ""
	}
	matches := make([]string, 0, 3)
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		foldedCandidate := foldSearchText(candidate)
		for _, hint := range foldedHints {
			if strings.Contains(foldedCandidate, hint) {
				addEvidenceMatch(&matches, seen, candidate)
				break
			}
		}
		if len(matches) >= 3 {
			break
		}
	}
	if len(matches) == 0 {
		tokenGroups := foldedHintTokenGroups(hints)
		for _, candidate := range candidates {
			foldedCandidate := foldSearchText(candidate)
			for _, tokens := range tokenGroups {
				if containsEnoughHintTokens(foldedCandidate, tokens) {
					addEvidenceMatch(&matches, seen, candidate)
					break
				}
			}
			if len(matches) >= 3 {
				break
			}
		}
	}
	return trimRunes(strings.Join(matches, " / "), 420)
}

func matchingAnswerOptions(answer string, options []ExplanationOption) []ExplanationOption {
	keys := splitAnswerKeys(answer)
	out := make([]ExplanationOption, 0, len(keys))
	for _, key := range keys {
		for _, option := range options {
			if strings.EqualFold(strings.TrimSpace(option.Key), key) {
				out = append(out, option)
				break
			}
		}
	}
	return out
}

func splitAnswerKeys(answer string) []string {
	parts := strings.FieldsFunc(answer, func(r rune) bool {
		return r == ',' || r == ';' || r == '/' || r == '|' || unicode.IsSpace(r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), ".:()[]{}")
		if part != "" {
			out = append(out, strings.ToUpper(part))
		}
	}
	return out
}

func optionLabels(options []ExplanationOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		key := strings.TrimSpace(option.Key)
		content := cleanOptionContent(option.Content)
		if key == "" {
			out = append(out, content)
		} else if content == "" {
			out = append(out, key)
		} else {
			out = append(out, key+" ("+content+")")
		}
	}
	return out
}

func optionContents(options []ExplanationOption) []string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		if content := cleanOptionContent(option.Content); content != "" {
			out = append(out, content)
		}
	}
	return out
}

func evidenceCandidates(material string) []string {
	material = strings.ReplaceAll(material, "\r\n", "\n")
	lines := strings.Split(material, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		for _, candidate := range splitLongEvidenceLine(line) {
			candidate = compactWhitespace(candidate)
			if candidate != "" {
				out = append(out, candidate)
			}
		}
	}
	return out
}

func splitLongEvidenceLine(line string) []string {
	line = strings.TrimSpace(line)
	if len([]rune(line)) <= 260 {
		return []string{line}
	}
	return strings.FieldsFunc(line, func(r rune) bool {
		return r == '.' || r == '?' || r == '!' || r == ';'
	})
}

func foldedSearchHints(hints []string) []string {
	out := make([]string, 0, len(hints))
	for _, hint := range hints {
		hint = foldSearchText(cleanOptionContent(hint))
		if hint != "" {
			out = append(out, hint)
		}
	}
	return out
}

func foldedHintTokenGroups(hints []string) [][]string {
	groups := make([][]string, 0, len(hints))
	for _, hint := range hints {
		tokens := significantTokens(foldSearchText(cleanOptionContent(hint)))
		if len(tokens) > 0 {
			groups = append(groups, tokens)
		}
	}
	return groups
}

func containsEnoughHintTokens(candidate string, tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}
	need := 2
	if len(tokens) == 1 {
		need = 1
	}
	matches := 0
	for _, token := range tokens {
		if strings.Contains(candidate, token) {
			matches++
		}
	}
	return matches >= need
}

func significantTokens(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if len([]rune(part)) > 2 && !isWeakEvidenceToken(part) {
			out = append(out, part)
		}
	}
	return out
}

func isWeakEvidenceToken(token string) bool {
	switch token {
	case "the", "and", "for", "with", "from", "into", "onto", "that", "this", "your":
		return true
	default:
		return false
	}
}

func addEvidenceMatch(matches *[]string, seen map[string]bool, candidate string) {
	candidate = compactWhitespace(candidate)
	if candidate == "" || seen[candidate] {
		return
	}
	seen[candidate] = true
	*matches = append(*matches, candidate)
}

func foldSearchText(value string) string {
	value = strings.ToLower(compactWhitespace(value))
	replacer := strings.NewReplacer(
		"ü", "u", "ö", "o", "ä", "a",
		"é", "e", "è", "e", "ê", "e",
		"á", "a", "à", "a", "â", "a",
		"í", "i", "ì", "i",
		"ó", "o", "ò", "o", "ô", "o",
		"ú", "u", "ù", "u",
	)
	return replacer.Replace(value)
}

func cleanOptionContent(value string) string {
	return strings.Trim(strings.TrimSpace(value), " \t\r\n.")
}

func compactWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func trimRunes(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + " [truncated]"
}

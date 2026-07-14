package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

func snapshotString(snapshot types.JSONMap, key string) string {
	if snapshot == nil {
		return ""
	}
	if value, ok := snapshot[key]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func snapshotOptions(snapshot types.JSONMap) []string {
	if snapshot == nil {
		return nil
	}
	value, ok := snapshot["options"]
	if !ok {
		return nil
	}
	return normalizeSnapshotOptions(value)
}

func normalizeSnapshotOptions(value any) []string {
	switch typed := value.(type) {
	case []*types.QuestionOption:
		return optionLinesFromPointers(typed)
	case []types.QuestionOption:
		return optionLinesFromValues(typed)
	case []map[string]any:
		return optionLinesFromMaps(typed)
	case []any:
		return optionLinesFromAny(typed)
	default:
		return nil
	}
}

func optionLinesFromPointers(options []*types.QuestionOption) []string {
	lines := make([]string, 0, len(options))
	for _, option := range options {
		if option != nil {
			lines = appendOptionLine(lines, option.OptionKey, option.Content)
		}
	}
	return lines
}

func optionLinesFromValues(options []types.QuestionOption) []string {
	lines := make([]string, 0, len(options))
	for _, option := range options {
		lines = appendOptionLine(lines, option.OptionKey, option.Content)
	}
	return lines
}

func optionLinesFromMaps(options []map[string]any) []string {
	lines := make([]string, 0, len(options))
	for _, option := range options {
		lines = appendOptionLine(lines, mapString(option, "option_key"), mapString(option, "content"))
	}
	return lines
}

func optionLinesFromAny(options []any) []string {
	lines := make([]string, 0, len(options))
	for _, option := range options {
		if m, ok := option.(map[string]any); ok {
			lines = appendOptionLine(lines, mapString(m, "option_key"), mapString(m, "content"))
		}
	}
	return lines
}

func appendOptionLine(lines []string, key string, content string) []string {
	key = strings.TrimSpace(key)
	content = strings.TrimSpace(content)
	if key == "" && content == "" {
		return lines
	}
	if key == "" {
		return append(lines, content)
	}
	return append(lines, fmt.Sprintf("%s. %s", key, content))
}

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	return ""
}

func firstExplanation(raw types.JSON) string {
	if len(raw) == 0 {
		return ""
	}
	var explanations []types.QuestionExplanation
	if err := json.Unmarshal(raw, &explanations); err != nil {
		return ""
	}
	parts := make([]string, 0, len(explanations))
	for _, explanation := range explanations {
		if text := strings.TrimSpace(explanation.ExplanationText); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

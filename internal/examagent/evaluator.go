package examagent

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const toolOutputExcerptLimit = 4000

func Summarize(results []types.ExamAgentEvaluationCaseResult) types.ExamAgentEvaluationSummary {
	summary := types.ExamAgentEvaluationSummary{Total: len(results)}
	if len(results) == 0 {
		return summary
	}
	for _, result := range results {
		if result.Passed {
			summary.Passed++
		}
		summary.ToolSequenceRate += result.ToolSequenceScore
		summary.ToolArgumentsRate += result.ToolArgumentsScore
		summary.EvidenceRate += result.EvidenceScore
		summary.CitationRate += result.CitationScore
		summary.AnswerRate += result.AnswerScore
		summary.GroundednessRate += result.GroundednessScore
		summary.AverageDurationMS += float64(result.DurationMS)
	}
	summary.Failed = summary.Total - summary.Passed
	count := float64(summary.Total)
	summary.PassRate = float64(summary.Passed) / count
	summary.ToolSequenceRate /= count
	summary.ToolArgumentsRate /= count
	summary.EvidenceRate /= count
	summary.CitationRate /= count
	summary.AnswerRate /= count
	summary.GroundednessRate /= count
	summary.AverageDurationMS /= count
	return summary
}

func EvaluateScenario(
	scenario types.ExamAgentEvaluationCase,
	state *types.AgentState,
	executionErr error,
	durationMS int64,
) types.ExamAgentEvaluationCaseResult {
	if state == nil {
		state = &types.AgentState{}
	}
	actual, evidence, toolFailure := collectToolCalls(state.RoundSteps)
	result := types.ExamAgentEvaluationCaseResult{
		Name: scenario.Name, Input: scenario.Input, DurationMS: durationMS,
		ActualToolCalls: actual, FinalAnswer: state.FinalAnswer,
	}
	result.ToolSequenceScore = toolSequenceScore(scenario.ExpectedToolCalls, actual)
	result.ToolArgumentsScore = toolArgumentsScore(scenario.ExpectedToolCalls, actual)
	result.EvidenceScore = phraseCoverage(scenario.ExpectedEvidencePhrases, evidence)
	result.CitationScore = dualPhraseCoverage(scenario.ExpectedCitations, evidence, state.FinalAnswer)
	result.AnswerScore = phraseCoverage(scenario.ExpectedAnswerPhrases, state.FinalAnswer)
	result.GroundednessScore = groundednessScore(scenario, evidence, state.FinalAnswer)
	result.AssertionFailures = assertionFailures(result, toolFailure)
	if strings.TrimSpace(state.FinalAnswer) == "" {
		result.AssertionFailures = append(result.AssertionFailures, "Final answer is empty")
	}
	if executionErr != nil {
		result.Error = executionErr.Error()
		result.AssertionFailures = append(result.AssertionFailures, "Agent execution failed")
	}
	result.Passed = len(result.AssertionFailures) == 0
	return result
}

func collectToolCalls(steps []types.AgentStep) ([]types.ExamAgentActualToolCall, string, bool) {
	actual := make([]types.ExamAgentActualToolCall, 0)
	var evidence strings.Builder
	toolFailure := false
	for _, step := range steps {
		for _, call := range step.ToolCalls {
			item := types.ExamAgentActualToolCall{
				Name: call.Name, Arguments: call.Args, DurationMS: call.Duration,
			}
			if call.Result != nil {
				item.Success = call.Result.Success
				item.Error = call.Result.Error
				item.OutputExcerpt = truncateRunes(call.Result.Output, toolOutputExcerptLimit)
				if call.Result.Success {
					evidence.WriteString("\n")
					evidence.WriteString(call.Result.Output)
				}
			}
			if !item.Success {
				toolFailure = true
			}
			actual = append(actual, item)
		}
	}
	return actual, evidence.String(), toolFailure
}

func toolSequenceScore(expected []types.ExamAgentExpectedToolCall, actual []types.ExamAgentActualToolCall) float64 {
	if len(expected) == 0 {
		return 1
	}
	if len(expected) != len(actual) {
		return 0
	}
	for i := range expected {
		if strings.TrimSpace(expected[i].Name) != actual[i].Name {
			return 0
		}
	}
	return 1
}

func toolArgumentsScore(expected []types.ExamAgentExpectedToolCall, actual []types.ExamAgentActualToolCall) float64 {
	if len(expected) == 0 {
		return 1
	}
	matched := 0
	for i, call := range expected {
		if i < len(actual) && call.Name == actual[i].Name && isSubset(call.Arguments, actual[i].Arguments) {
			matched++
		}
	}
	return float64(matched) / float64(len(expected))
}

func isSubset(expected, actual map[string]interface{}) bool {
	for key, expectedValue := range expected {
		actualValue, ok := actual[key]
		if !ok || !valueMatches(expectedValue, actualValue) {
			return false
		}
	}
	return true
}

func valueMatches(expected, actual interface{}) bool {
	if expectedMap, ok := expected.(map[string]interface{}); ok {
		actualMap, ok := actual.(map[string]interface{})
		return ok && isSubset(expectedMap, actualMap)
	}
	if expectedSlice, ok := expected.([]interface{}); ok {
		actualSlice, ok := actual.([]interface{})
		if !ok || len(expectedSlice) != len(actualSlice) {
			return false
		}
		for i := range expectedSlice {
			if !valueMatches(expectedSlice[i], actualSlice[i]) {
				return false
			}
		}
		return true
	}
	if expectedNumber, ok := numberValue(expected); ok {
		actualNumber, ok := numberValue(actual)
		return ok && math.Abs(expectedNumber-actualNumber) < 1e-9
	}
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)
	return string(expectedJSON) == string(actualJSON)
}

func numberValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		number, err := strconv.ParseFloat(string(v), 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func phraseCoverage(phrases []string, text string) float64 {
	if len(phrases) == 0 {
		return 1
	}
	matched := 0
	for _, phrase := range phrases {
		if containsPhrase(text, phrase) {
			matched++
		}
	}
	return float64(matched) / float64(len(phrases))
}

func dualPhraseCoverage(phrases []string, first, second string) float64 {
	if len(phrases) == 0 {
		return 1
	}
	matched := 0
	for _, phrase := range phrases {
		if containsPhrase(first, phrase) && containsPhrase(second, phrase) {
			matched++
		}
	}
	return float64(matched) / float64(len(phrases))
}

func groundednessScore(scenario types.ExamAgentEvaluationCase, evidence, answer string) float64 {
	if len(scenario.GroundedPhrases) > 0 {
		return dualPhraseCoverage(scenario.GroundedPhrases, evidence, answer)
	}
	return phraseCoverage(scenario.ExpectedAnswerPhrases, answer)
}

func containsPhrase(text, phrase string) bool {
	phrase = normalizeText(phrase)
	return phrase != "" && strings.Contains(normalizeText(text), phrase)
}

func normalizeText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func assertionFailures(result types.ExamAgentEvaluationCaseResult, toolFailure bool) []string {
	checks := []struct {
		label string
		score float64
	}{
		{"Tool sequence", result.ToolSequenceScore},
		{"Tool arguments", result.ToolArgumentsScore},
		{"Evidence", result.EvidenceScore},
		{"Citations", result.CitationScore},
		{"Answer", result.AnswerScore},
		{"Groundedness", result.GroundednessScore},
	}
	failures := make([]string, 0)
	for _, check := range checks {
		if check.score < 1 {
			failures = append(failures, fmt.Sprintf("%s assertion failed (%.0f%%)", check.label, check.score*100))
		}
	}
	if toolFailure {
		failures = append(failures, "One or more tools failed")
	}
	return failures
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

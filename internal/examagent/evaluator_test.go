package examagent

import (
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestEvaluateScenarioPassesDeterministicAgentAssertions(t *testing.T) {
	t.Parallel()
	scenario := types.ExamAgentEvaluationCase{
		Name:  "class diagnosis and question context",
		Input: "分析班级后给出第21题依据",
		ExpectedToolCalls: []types.ExamAgentExpectedToolCall{
			{Name: "exam_class_diagnosis", Arguments: map[string]interface{}{"class_id": "class-1"}},
			{Name: "exam_question_context", Arguments: map[string]interface{}{"query": "第21题"}},
		},
		ExpectedEvidencePhrases: []string{"高频错题", "第21题"},
		ExpectedCitations:       []string{"chunk-21"},
		ExpectedAnswerPhrases:   []string{"重点复习第21题"},
		GroundedPhrases:         []string{"Los Angeles Rams"},
	}
	state := &types.AgentState{
		FinalAnswer: "重点复习第21题，证据来自 chunk-21，答案涉及 Los Angeles Rams。",
		RoundSteps: []types.AgentStep{
			{Iteration: 0, ToolCalls: []types.ToolCall{{
				Name: "exam_class_diagnosis", Args: map[string]interface{}{
					"class_id": "class-1", "top_wrong_questions": float64(10),
				}, Result: &types.ToolResult{Success: true, Output: "高频错题：第21题"},
			}}},
			{Iteration: 1, ToolCalls: []types.ToolCall{{
				Name: "exam_question_context", Args: map[string]interface{}{
					"query": "第21题", "options": map[string]interface{}{"include_answer": true},
				}, Result: &types.ToolResult{Success: true, Output: "第21题 chunk-21 Los Angeles Rams"},
			}}},
		},
	}

	result := EvaluateScenario(scenario, state, nil, 1250)

	require.True(t, result.Passed)
	require.Equal(t, float64(1), result.ToolSequenceScore)
	require.Equal(t, float64(1), result.ToolArgumentsScore)
	require.Equal(t, float64(1), result.EvidenceScore)
	require.Equal(t, float64(1), result.CitationScore)
	require.Equal(t, float64(1), result.AnswerScore)
	require.Equal(t, float64(1), result.GroundednessScore)
	require.Len(t, result.ActualToolCalls, 2)
	require.Empty(t, result.AssertionFailures)
}

func TestEvaluateScenarioReportsSequenceArgumentsAndEvidenceFailures(t *testing.T) {
	t.Parallel()
	scenario := types.ExamAgentEvaluationCase{
		Name: "strict assertions", Input: "query",
		ExpectedToolCalls: []types.ExamAgentExpectedToolCall{
			{Name: "exam_class_diagnosis", Arguments: map[string]interface{}{
				"class_id": "class-1", "filters": map[string]interface{}{"subject": "math"},
			}},
		},
		ExpectedEvidencePhrases: []string{"correct evidence"},
		ExpectedCitations:       []string{"question-9"},
		ExpectedAnswerPhrases:   []string{"correct answer"},
		GroundedPhrases:         []string{"shared fact"},
	}
	state := &types.AgentState{
		FinalAnswer: "question-8 shared fact",
		RoundSteps: []types.AgentStep{{ToolCalls: []types.ToolCall{
			{Name: "exam_question_context", Args: map[string]interface{}{"query": "wrong"}, Result: &types.ToolResult{Success: true, Output: "question-8"}},
			{Name: "exam_class_diagnosis", Args: map[string]interface{}{
				"class_id": "class-2", "filters": map[string]interface{}{"subject": "english"},
			}, Result: &types.ToolResult{Success: false, Error: "denied"}},
		}}},
	}

	result := EvaluateScenario(scenario, state, errors.New("agent stopped"), 25)

	require.False(t, result.Passed)
	require.Equal(t, float64(0), result.ToolSequenceScore)
	require.Equal(t, float64(0), result.ToolArgumentsScore)
	require.Equal(t, float64(0), result.EvidenceScore)
	require.Equal(t, float64(0), result.CitationScore)
	require.Equal(t, float64(0), result.AnswerScore)
	require.Equal(t, float64(0), result.GroundednessScore)
	require.Contains(t, result.Error, "agent stopped")
	require.NotEmpty(t, result.AssertionFailures)
}

func TestEvaluateScenarioTreatsUnconfiguredAssertionsAsNeutral(t *testing.T) {
	t.Parallel()
	result := EvaluateScenario(
		types.ExamAgentEvaluationCase{Name: "answer only", Input: "hello"},
		&types.AgentState{FinalAnswer: "你好"}, nil, 8,
	)

	require.True(t, result.Passed)
	require.Equal(t, float64(1), result.ToolSequenceScore)
	require.Equal(t, float64(1), result.ToolArgumentsScore)
	require.Equal(t, float64(1), result.EvidenceScore)
	require.Equal(t, float64(1), result.CitationScore)
	require.Equal(t, float64(1), result.AnswerScore)
	require.Equal(t, float64(1), result.GroundednessScore)
}

func TestEvaluateScenarioRejectsEmptyFinalAnswer(t *testing.T) {
	t.Parallel()
	scenario := types.ExamAgentEvaluationCase{
		Name: "tool only", Input: "diagnose",
		ExpectedToolCalls: []types.ExamAgentExpectedToolCall{{Name: "exam_class_diagnosis"}},
	}
	state := &types.AgentState{RoundSteps: []types.AgentStep{{ToolCalls: []types.ToolCall{{
		Name: "exam_class_diagnosis", Result: &types.ToolResult{Success: true, Output: "evidence"},
	}}}}}

	result := EvaluateScenario(scenario, state, nil, 10)

	require.False(t, result.Passed)
	require.Contains(t, result.AssertionFailures, "Final answer is empty")
}

func TestSummarizeAveragesAllAgentEvaluationMetrics(t *testing.T) {
	t.Parallel()
	summary := Summarize([]types.ExamAgentEvaluationCaseResult{
		{
			Passed: true, ToolSequenceScore: 1, ToolArgumentsScore: 1,
			EvidenceScore: 1, CitationScore: 1, AnswerScore: 1,
			GroundednessScore: 1, DurationMS: 100,
		},
		{
			Passed: false, ToolSequenceScore: 0, ToolArgumentsScore: 0.5,
			EvidenceScore: 0.5, CitationScore: 0, AnswerScore: 1,
			GroundednessScore: 0.5, DurationMS: 300,
		},
	})

	require.Equal(t, 2, summary.Total)
	require.Equal(t, 1, summary.Passed)
	require.Equal(t, 1, summary.Failed)
	require.Equal(t, 0.5, summary.PassRate)
	require.Equal(t, 0.5, summary.ToolSequenceRate)
	require.Equal(t, 0.75, summary.ToolArgumentsRate)
	require.Equal(t, 0.5, summary.CitationRate)
	require.Equal(t, 0.75, summary.GroundednessRate)
	require.Equal(t, float64(200), summary.AverageDurationMS)
}

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

func TestExamAgentEvaluationServiceCreatesSafeQueuedSnapshot(t *testing.T) {
	t.Parallel()
	repo := newExamRAGEvaluationRunRepoStub()
	enqueuer := &examRAGEvaluationEnqueuerStub{}
	svc := NewExamAgentEvaluationService(
		repo, testRAGQuestionService(), &examAgentCustomAgentServiceStub{agent: testEvaluationAgent()},
		testRAGTenantService(), nil, enqueuer,
	)
	req := testAgentEvaluationRequest()

	run, err := svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &req)

	require.NoError(t, err)
	require.Equal(t, types.ExamEvaluationKindAgent, run.EvaluationKind)
	require.Equal(t, "agent-1", run.AgentID)
	require.Equal(t, types.ExamRAGEvaluationRunStatusQueued, run.Status)
	require.Equal(t, types.TypeExamAgentEvaluationRun, enqueuer.task.Type())
	var snapshot types.ExamAgentEvaluationRequestSnapshot
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &snapshot))
	require.Equal(t, []string{"exam_class_diagnosis", "exam_question_context"}, snapshot.Agent.AllowedTools)
	require.Equal(t, "none", snapshot.Agent.Config.MCPSelectionMode)
	require.Equal(t, "none", snapshot.Agent.Config.SkillsSelectionMode)
	require.False(t, snapshot.Agent.Config.WebSearchEnabled)
	require.Len(t, snapshot.Cases, 1)
}

func TestExamAgentEvaluationServiceCreatesRunFromStoredSnapshot(t *testing.T) {
	t.Parallel()

	currentAgent := testEvaluationAgent()
	currentAgent.Config.ModelID = "current-model-v2"
	enqueuer := &examRAGEvaluationEnqueuerStub{}
	svc := NewExamAgentEvaluationService(
		newExamRAGEvaluationRunRepoStub(), testRAGQuestionService(),
		&examAgentCustomAgentServiceStub{agent: currentAgent},
		testRAGTenantService(), nil, enqueuer,
	)
	snapshot := &types.ExamAgentEvaluationRequestSnapshot{
		Agent: types.ExamAgentSnapshot{
			ID: "agent-1", Name: "版本化诊断助手", ModelID: "snapshot-model-v1",
			AllowedTools: []string{"exam_question_context"},
			Config: types.CustomAgentConfig{
				AgentMode: types.AgentModeSmartReasoning, ModelID: "snapshot-model-v1",
				AllowedTools:     []string{"exam_question_context"},
				MCPSelectionMode: "none", SkillsSelectionMode: "none",
			},
		},
		Cases: []types.ExamAgentEvaluationCase{{Name: "固定场景", Input: "诊断这道题"}},
	}
	runner, ok := svc.(interface {
		CreateRunFromSnapshot(context.Context, uint64, string, string, *types.ExamAgentEvaluationRequestSnapshot) (*types.ExamRAGEvaluationRun, error)
	})
	require.True(t, ok)

	run, err := runner.CreateRunFromSnapshot(context.Background(), 10000, "teacher-1", "bank-1", snapshot)
	require.NoError(t, err)
	require.Equal(t, types.TypeExamAgentEvaluationRun, enqueuer.task.Type())
	var stored types.ExamAgentEvaluationRequestSnapshot
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &stored))
	require.Equal(t, "snapshot-model-v1", stored.Agent.ModelID)
	require.Equal(t, "snapshot-model-v1", stored.Agent.Config.ModelID)
	require.Equal(t, "版本化诊断助手", stored.Agent.Name)
}

func TestExamAgentEvaluationServiceRejectsUnsafeExpectedTool(t *testing.T) {
	t.Parallel()
	svc := NewExamAgentEvaluationService(
		newExamRAGEvaluationRunRepoStub(), testRAGQuestionService(),
		&examAgentCustomAgentServiceStub{agent: testEvaluationAgent()},
		testRAGTenantService(), nil, &examRAGEvaluationEnqueuerStub{},
	)
	req := testAgentEvaluationRequest()
	req.Cases[0].ExpectedToolCalls = []types.ExamAgentExpectedToolCall{{Name: "wiki_write_page"}}

	_, err := svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &req)

	require.ErrorIs(t, err, ErrExamInvalidRequest)
}

func TestExamAgentEvaluationServiceUsesProductionDefaultTools(t *testing.T) {
	t.Parallel()
	agent := testEvaluationAgent()
	agent.Config.AllowedTools = nil
	svc := NewExamAgentEvaluationService(
		newExamRAGEvaluationRunRepoStub(), testRAGQuestionService(),
		&examAgentCustomAgentServiceStub{agent: agent},
		testRAGTenantService(), nil, &examRAGEvaluationEnqueuerStub{},
	)
	req := testAgentEvaluationRequest()

	run, err := svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &req)

	require.NoError(t, err)
	var snapshot types.ExamAgentEvaluationRequestSnapshot
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &snapshot))
	require.Contains(t, snapshot.Agent.AllowedTools, "exam_class_diagnosis")
	require.Contains(t, snapshot.Agent.AllowedTools, "exam_question_context")
}

func TestSafeAgentEvaluationConfigExcludesUnscopedKnowledgeTools(t *testing.T) {
	t.Parallel()
	config := safeAgentEvaluationConfig(types.CustomAgentConfig{AllowedTools: []string{
		"exam_question_context", "data_analysis", "data_schema", "wiki_read_source_doc",
	}})

	require.Equal(t, []string{"exam_question_context"}, config.AllowedTools)
}

func TestExamAgentEvaluationServiceRejectsWrongAgentModeAndCaseBounds(t *testing.T) {
	t.Parallel()
	agent := testEvaluationAgent()
	agent.Config.AgentMode = types.AgentModeQuickAnswer
	svc := NewExamAgentEvaluationService(
		newExamRAGEvaluationRunRepoStub(), testRAGQuestionService(),
		&examAgentCustomAgentServiceStub{agent: agent},
		testRAGTenantService(), nil, &examRAGEvaluationEnqueuerStub{},
	)
	req := testAgentEvaluationRequest()

	_, err := svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &req)
	require.ErrorIs(t, err, ErrExamInvalidRequest)

	agent.Config.AgentMode = types.AgentModeSmartReasoning
	req.Cases = make([]types.ExamAgentEvaluationCase, maxExamAgentEvaluationCases+1)
	_, err = svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &req)
	require.ErrorIs(t, err, ErrExamInvalidRequest)
}

func TestExamAgentEvaluationServiceListsOnlyAgentRunsWithCompactedResults(t *testing.T) {
	t.Parallel()
	repo := newExamRAGEvaluationRunRepoStub()
	run := testQueuedAgentEvaluationRun(t, "agent-run-compact")
	run.ResultSnapshot, _ = json.Marshal(types.ExamAgentEvaluationResult{
		QuestionBankID: "bank-1",
		Summary:        types.ExamAgentEvaluationSummary{Total: 1, Passed: 1, PassRate: 1},
		Results: []types.ExamAgentEvaluationCaseResult{{
			Name: "class diagnosis", Passed: true, FinalAnswer: "full trace",
		}},
	})
	repo.runs[run.ID] = run
	repo.runs["rag-run"] = testQueuedRAGEvaluationRun("rag-run")
	svc := NewExamAgentEvaluationService(
		repo, testRAGQuestionService(), nil, testRAGTenantService(), nil, &examRAGEvaluationEnqueuerStub{},
	)

	runs, err := svc.ListRuns(context.Background(), 10000, "teacher-1", "bank-1", 20)

	require.NoError(t, err)
	require.Len(t, runs, 1)
	var request types.ExamAgentEvaluationRequestSnapshot
	require.NoError(t, json.Unmarshal(runs[0].RequestSnapshot, &request))
	require.Empty(t, request.Cases)
	require.Equal(t, "agent-1", request.Agent.ID)
	var result types.ExamAgentEvaluationResult
	require.NoError(t, json.Unmarshal(runs[0].ResultSnapshot, &result))
	require.Equal(t, 1, result.Summary.Passed)
	require.Empty(t, result.Results)
	var stored types.ExamAgentEvaluationResult
	require.NoError(t, json.Unmarshal(repo.runs[run.ID].ResultSnapshot, &stored))
	require.Len(t, stored.Results, 1)
}

func testEvaluationAgent() *types.CustomAgent {
	return &types.CustomAgent{
		ID: "agent-1", Name: "Class diagnosis assistant", TenantID: 10000,
		Config: types.CustomAgentConfig{
			AgentMode: types.AgentModeSmartReasoning, ModelID: "model-1",
			AllowedTools:   []string{"exam_class_diagnosis", "exam_question_context", "wiki_write_page"},
			KnowledgeBases: []string{"kb-1"}, KBSelectionMode: "selected",
			MCPSelectionMode: "all", SkillsSelectionMode: "all", WebSearchEnabled: true,
		},
	}
}

func testAgentEvaluationRequest() types.RunExamAgentEvaluationRequest {
	return types.RunExamAgentEvaluationRequest{
		AgentID: "agent-1",
		Cases: []types.ExamAgentEvaluationCase{{
			Name: "class diagnosis", Input: "diagnose class-1 and explain question 21",
			ExpectedToolCalls: []types.ExamAgentExpectedToolCall{
				{Name: "exam_class_diagnosis", Arguments: map[string]interface{}{"class_id": "class-1"}},
				{Name: "exam_question_context"},
			},
			ExpectedEvidencePhrases: []string{"question 21"},
		}},
	}
}

type examAgentCustomAgentServiceStub struct {
	interfaces.CustomAgentService
	agent *types.CustomAgent
	err   error
}

func (s *examAgentCustomAgentServiceStub) GetAgentByID(context.Context, string) (*types.CustomAgent, error) {
	return s.agent, s.err
}

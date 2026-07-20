package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

func TestExecuteAgentEvaluationReturnsProductionAgentState(t *testing.T) {
	t.Parallel()
	expected := &types.AgentState{
		IsComplete: true, FinalAnswer: "grounded answer",
		RoundSteps: []types.AgentStep{{ToolCalls: []types.ToolCall{{Name: "exam_question_context"}}}},
	}
	svc := newAgentEvaluationSessionService(&agentEvaluationEngineStub{state: expected})

	state, err := svc.ExecuteAgentEvaluation(context.Background(), testAgentEvaluationQARequest(), event.NewEventBus())

	require.NoError(t, err)
	require.Same(t, expected, state)
}

func TestAgentQAPreservesErrorEventBehaviorAfterEvaluationRefactor(t *testing.T) {
	t.Parallel()
	executionErr := errors.New("agent engine unavailable")
	svc := newAgentEvaluationSessionService(&agentEvaluationEngineStub{err: executionErr})
	bus := event.NewEventBus()
	var emitted error
	bus.On(event.EventError, func(_ context.Context, evt event.Event) error {
		data, ok := evt.Data.(event.ErrorData)
		require.True(t, ok)
		emitted = errors.New(data.Error)
		return nil
	})

	err := svc.AgentQA(context.Background(), testAgentEvaluationQARequest(), bus)

	require.NoError(t, err)
	require.EqualError(t, emitted, executionErr.Error())
}

func newAgentEvaluationSessionService(engine interfaces.AgentEngine) *sessionService {
	return &sessionService{
		modelService: &stubModelService{
			modelsByID: map[string]*types.Model{
				"model-1": {ID: "model-1", Type: types.ModelTypeKnowledgeQA},
			},
		},
		tenantService: &examRAGEvaluationTenantServiceStub{tenant: &types.Tenant{ID: 10000}},
		agentService:  &agentEvaluationServiceStub{engine: engine},
	}
}

func testAgentEvaluationQARequest() *types.QARequest {
	return &types.QARequest{
		Session: &types.Session{ID: "eval-session", TenantID: 10000, UserID: "teacher-1"},
		Query:   "evaluate this scenario", AssistantMessageID: "eval-message", DisableHistory: true,
		CustomAgent: &types.CustomAgent{
			ID: "agent-1", TenantID: 10000,
			Config: types.CustomAgentConfig{
				AgentMode: "smart-reasoning", ModelID: "model-1", MaxIterations: 2,
				AllowedTools: []string{"exam_question_context"}, KBSelectionMode: "none",
				MCPSelectionMode: "none", SkillsSelectionMode: "none",
				WebSearchProviderID: "disabled",
			},
		},
	}
}

type agentEvaluationServiceStub struct {
	interfaces.AgentService
	engine interfaces.AgentEngine
}

func (s *agentEvaluationServiceStub) CreateAgentEngine(
	context.Context, *types.AgentConfig, chat.Chat, rerank.Reranker,
	*event.EventBus, string, string,
) (interfaces.AgentEngine, error) {
	return s.engine, nil
}

type agentEvaluationEngineStub struct {
	state *types.AgentState
	err   error
}

func (s *agentEvaluationEngineStub) Execute(
	context.Context, string, string, string, []chat.Message, ...[]string,
) (*types.AgentState, error) {
	return s.state, s.err
}

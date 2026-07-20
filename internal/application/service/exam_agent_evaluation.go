package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	agenttools "github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const maxExamAgentEvaluationCases = 20

var examAgentEvaluationReadOnlyTools = map[string]struct{}{
	agenttools.ToolThinking: {}, agenttools.ToolTodoWrite: {},
	agenttools.ToolGrepChunks: {}, agenttools.ToolKnowledgeSearch: {},
	agenttools.ToolExamQuestionContext: {}, agenttools.ToolExamLearningDiagnosis: {},
	agenttools.ToolExamClassDiagnosis: {}, agenttools.ToolExamPracticeRecommendation: {},
	agenttools.ToolListKnowledgeChunks: {}, agenttools.ToolQueryKnowledgeGraph: {},
	agenttools.ToolGetDocumentInfo: {}, agenttools.ToolDatabaseQuery: {},
	agenttools.ToolWikiReadPage: {}, agenttools.ToolWikiSearch: {},
	agenttools.ToolWikiReadIssue: {},
}

type examAgentEvaluationService struct {
	repo            interfaces.ExamRAGEvaluationRepository
	questionService interfaces.ExamQuestionService
	agentService    interfaces.CustomAgentService
	tenantService   interfaces.TenantService
	runner          interfaces.SessionService
	taskEnqueuer    interfaces.TaskEnqueuer
}

func NewExamAgentEvaluationService(
	repo interfaces.ExamRAGEvaluationRepository,
	questionService interfaces.ExamQuestionService,
	agentService interfaces.CustomAgentService,
	tenantService interfaces.TenantService,
	runner interfaces.SessionService,
	taskEnqueuer interfaces.TaskEnqueuer,
) interfaces.ExamAgentEvaluationService {
	return &examAgentEvaluationService{
		repo: repo, questionService: questionService, agentService: agentService,
		tenantService: tenantService, runner: runner, taskEnqueuer: taskEnqueuer,
	}
}

func (s *examAgentEvaluationService) CreateRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	req *types.RunExamAgentEvaluationRequest,
) (*types.ExamRAGEvaluationRun, error) {
	bankID = strings.TrimSpace(bankID)
	userID = strings.TrimSpace(userID)
	if tenantID == 0 || userID == "" || req == nil || bankID == "" || s.agentService == nil {
		return nil, ErrExamInvalidRequest
	}
	bank, err := s.questionService.GetQuestionBank(ctx, tenantID, userID, bankID)
	if err != nil {
		return nil, err
	}
	agentCtx := context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	agentCtx = context.WithValue(agentCtx, types.UserIDContextKey, userID)
	agent, err := s.agentService.GetAgentByID(agentCtx, strings.TrimSpace(req.AgentID))
	if err != nil {
		return nil, err
	}
	if bank == nil || agent == nil || agent.TenantID != tenantID || agent.Config.AgentMode != types.AgentModeSmartReasoning {
		return nil, ErrExamInvalidRequest
	}
	safeAgent := cloneAgentForEvaluation(agent)
	safeAgent.Config = safeAgentEvaluationConfig(safeAgent.Config)
	cases, err := normalizeAgentEvaluationCases(req.Cases, safeAgent.Config.AllowedTools)
	if err != nil {
		return nil, err
	}
	run, err := newExamAgentEvaluationRun(tenantID, userID, bank.ID, safeAgent, cases)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	if err := s.enqueueRun(run.ID); err != nil {
		return nil, s.failQueuedRun(ctx, run, err)
	}
	return run, nil
}

func (s *examAgentEvaluationService) ListRuns(
	ctx context.Context, tenantID uint64, userID, bankID string, limit int,
) ([]*types.ExamRAGEvaluationRun, error) {
	if err := s.authorizeBank(ctx, tenantID, userID, bankID); err != nil {
		return nil, err
	}
	runs, err := s.repo.ListRuns(ctx, tenantID, strings.TrimSpace(bankID), types.ExamEvaluationKindAgent, limit)
	if err != nil {
		return nil, err
	}
	return compactAgentEvaluationRuns(runs), nil
}

func (s *examAgentEvaluationService) GetRun(
	ctx context.Context, tenantID uint64, userID, bankID, runID string,
) (*types.ExamRAGEvaluationRun, error) {
	if err := s.authorizeBank(ctx, tenantID, userID, bankID); err != nil {
		return nil, err
	}
	return s.repo.GetRun(ctx, tenantID, strings.TrimSpace(bankID), types.ExamEvaluationKindAgent, strings.TrimSpace(runID))
}

func (s *examAgentEvaluationService) authorizeBank(
	ctx context.Context, tenantID uint64, userID, bankID string,
) error {
	_, err := s.questionService.GetQuestionBank(ctx, tenantID, strings.TrimSpace(userID), strings.TrimSpace(bankID))
	return err
}

func newExamAgentEvaluationRun(
	tenantID uint64, userID, bankID string, agent *types.CustomAgent, cases []types.ExamAgentEvaluationCase,
) (*types.ExamRAGEvaluationRun, error) {
	request, err := json.Marshal(types.ExamAgentEvaluationRequestSnapshot{
		Agent: newExamAgentSnapshot(agent), Cases: cases,
	})
	if err != nil {
		return nil, err
	}
	progress, err := json.Marshal(types.ExamRAGEvaluationProgress{TotalCases: len(cases)})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &types.ExamRAGEvaluationRun{
		ID: uuid.NewString(), TenantID: tenantID, QuestionBankID: bankID,
		EvaluationKind: types.ExamEvaluationKindAgent, AgentID: agent.ID,
		CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
		Progress: progress, RequestSnapshot: request, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func normalizeAgentEvaluationCases(
	cases []types.ExamAgentEvaluationCase, allowedTools []string,
) ([]types.ExamAgentEvaluationCase, error) {
	if len(cases) == 0 || len(cases) > maxExamAgentEvaluationCases || len(allowedTools) == 0 {
		return nil, ErrExamInvalidRequest
	}
	allowed := make(map[string]struct{}, len(allowedTools))
	for _, name := range allowedTools {
		allowed[name] = struct{}{}
	}
	normalized := make([]types.ExamAgentEvaluationCase, len(cases))
	for i, scenario := range cases {
		scenario.Name = strings.TrimSpace(scenario.Name)
		scenario.Input = strings.TrimSpace(scenario.Input)
		if scenario.Name == "" || scenario.Input == "" {
			return nil, ErrExamInvalidRequest
		}
		for j := range scenario.ExpectedToolCalls {
			scenario.ExpectedToolCalls[j].Name = strings.TrimSpace(scenario.ExpectedToolCalls[j].Name)
			if _, ok := allowed[scenario.ExpectedToolCalls[j].Name]; !ok {
				return nil, ErrExamInvalidRequest
			}
		}
		normalized[i] = scenario
	}
	return normalized, nil
}

func safeAgentEvaluationConfig(config types.CustomAgentConfig) types.CustomAgentConfig {
	tools := config.AllowedTools
	if len(tools) == 0 {
		tools = agenttools.DefaultAllowedTools()
	}
	config.AllowedTools = filterAgentEvaluationTools(tools)
	config.MCPSelectionMode = "none"
	config.MCPServices = nil
	config.MCPAuthWaitTimeout = 0
	config.SkillsSelectionMode = "none"
	config.SelectedSkills = nil
	config.WebSearchEnabled = false
	config.WebSearchProviderID = ""
	config.MultiTurnEnabled = false
	config.HistoryTurns = 0
	return config
}

func filterAgentEvaluationTools(names []string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		_, safe := examAgentEvaluationReadOnlyTools[name]
		_, duplicate := seen[name]
		if !safe || duplicate {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func cloneAgentForEvaluation(agent *types.CustomAgent) *types.CustomAgent {
	clone := *agent
	clone.Config.AllowedTools = append([]string(nil), agent.Config.AllowedTools...)
	clone.Config.KnowledgeBases = append([]string(nil), agent.Config.KnowledgeBases...)
	clone.Config.MCPServices = append([]string(nil), agent.Config.MCPServices...)
	clone.Config.SelectedSkills = append([]string(nil), agent.Config.SelectedSkills...)
	return &clone
}

func newExamAgentSnapshot(agent *types.CustomAgent) types.ExamAgentSnapshot {
	return types.ExamAgentSnapshot{
		ID: agent.ID, Name: agent.Name, ModelID: agent.Config.ModelID,
		AllowedTools:   append([]string(nil), agent.Config.AllowedTools...),
		KnowledgeBases: append([]string(nil), agent.Config.KnowledgeBases...),
		Config:         agent.Config,
	}
}

func compactAgentEvaluationRuns(runs []*types.ExamRAGEvaluationRun) []*types.ExamRAGEvaluationRun {
	compacted := make([]*types.ExamRAGEvaluationRun, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		clone := *run
		clone.RequestSnapshot = compactAgentEvaluationRequestSnapshot(run.RequestSnapshot)
		clone.ResultSnapshot = compactAgentEvaluationResultSnapshot(run.ResultSnapshot)
		compacted = append(compacted, &clone)
	}
	return compacted
}

func compactAgentEvaluationRequestSnapshot(snapshot types.JSON) types.JSON {
	var request types.ExamAgentEvaluationRequestSnapshot
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &request) != nil {
		return nil
	}
	request.Cases = nil
	compacted, _ := json.Marshal(request)
	return compacted
}

func compactAgentEvaluationResultSnapshot(snapshot types.JSON) types.JSON {
	var result types.ExamAgentEvaluationResult
	if len(snapshot) == 0 || json.Unmarshal(snapshot, &result) != nil {
		return nil
	}
	result.Results = nil
	compacted, _ := json.Marshal(result)
	return compacted
}

func (s *examAgentEvaluationService) enqueueRun(runID string) error {
	if s.taskEnqueuer == nil {
		return errors.New("exam Agent evaluation task enqueuer is not configured")
	}
	payload, err := json.Marshal(types.ExamAgentEvaluationTaskPayload{RunID: runID})
	if err != nil {
		return err
	}
	task := asynq.NewTask(types.TypeExamAgentEvaluationRun, payload)
	_, err = s.taskEnqueuer.Enqueue(task, asynq.Queue(types.QueueQuestion), asynq.MaxRetry(0))
	return err
}

func (s *examAgentEvaluationService) failQueuedRun(
	ctx context.Context, run *types.ExamRAGEvaluationRun, cause error,
) error {
	now := time.Now().UTC()
	run.Status = types.ExamRAGEvaluationRunStatusFailed
	run.ErrorMessage = cause.Error()
	run.CompletedAt = &now
	run.UpdatedAt = now
	if err := s.repo.UpdateRun(context.WithoutCancel(ctx), run.TenantID, run.QuestionBankID, types.ExamEvaluationKindAgent, run); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

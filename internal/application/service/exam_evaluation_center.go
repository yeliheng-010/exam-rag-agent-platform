package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type examEvaluationCenterService struct {
	repo        interfaces.ExamEvaluationCenterRepository
	questions   interfaces.ExamQuestionService
	ragRunner   interfaces.ExamRAGEvaluationService
	agentRunner interfaces.ExamAgentEvaluationService
}

func NewExamEvaluationCenterService(
	repo interfaces.ExamEvaluationCenterRepository,
	questions interfaces.ExamQuestionService,
	ragRunner interfaces.ExamRAGEvaluationService,
	agentRunner interfaces.ExamAgentEvaluationService,
) interfaces.ExamEvaluationCenterService {
	return &examEvaluationCenterService{
		repo: repo, questions: questions, ragRunner: ragRunner, agentRunner: agentRunner,
	}
}

func (s *examEvaluationCenterService) GetCenter(
	ctx context.Context,
	tenantID uint64,
	userID string,
	filter types.ExamEvaluationCenterFilter,
) (*types.ExamEvaluationCenterResult, error) {
	filter, err := normalizeEvaluationCenterFilter(filter)
	if err != nil {
		return nil, err
	}
	bankIDs, banks, err := s.resolveCenterBankScope(ctx, tenantID, userID, filter.BankID)
	if err != nil {
		return nil, err
	}
	requestedLimit := filter.Limit
	filter.Limit = 200
	runs, err := s.repo.ListCenterRuns(ctx, tenantID, bankIDs, filter)
	if err != nil {
		return nil, err
	}
	baselines, err := s.repo.ListBaselines(ctx, tenantID, bankIDs, filter.Kind, filter.AgentID)
	if err != nil {
		return nil, err
	}
	sets, err := s.repo.ListEvaluationSets(ctx, tenantID, bankIDs, filter)
	if err != nil {
		return nil, err
	}
	items := buildEvaluationCenterItems(runs, baselines, banks, sets)
	items = filterEvaluationCenterItems(items, filter.Regression, requestedLimit)
	return &types.ExamEvaluationCenterResult{
		Summary: calculateEvaluationCenterSummary(items),
		Items:   items, QuestionBanks: banks, Agents: collectEvaluationAgentOptions(runs),
	}, nil
}

func (s *examEvaluationCenterService) SetBaseline(
	ctx context.Context,
	tenantID uint64,
	userID string,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	run, err := s.authorizedCenterRun(ctx, tenantID, userID, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != types.ExamRAGEvaluationRunStatusCompleted {
		return nil, ErrExamInvalidRequest
	}
	if err := s.repo.SetBaseline(ctx, tenantID, run.ID); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetCenterRun(ctx, tenantID, run.ID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *examEvaluationCenterService) resolveCenterBankScope(
	ctx context.Context,
	tenantID uint64,
	userID string,
	requestedBankID string,
) ([]string, []*types.QuestionBank, error) {
	requestedBankID = strings.TrimSpace(requestedBankID)
	role := types.TenantRoleFromContext(ctx)
	if role.HasPermission(types.TenantRoleAdmin) {
		var scope []string
		if requestedBankID != "" {
			if _, err := s.questions.GetQuestionBank(ctx, tenantID, userID, requestedBankID); err != nil {
				return nil, nil, err
			}
			scope = []string{requestedBankID}
		}
		banks, err := s.repo.ListCenterQuestionBanks(ctx, tenantID, scope)
		return scope, banks, err
	}
	banks, err := s.questions.ListQuestionBanks(ctx, tenantID, userID, "")
	if err != nil {
		return nil, nil, err
	}
	if requestedBankID != "" {
		banks = selectVisibleCenterBank(banks, requestedBankID)
		if len(banks) == 0 {
			return nil, nil, ErrExamPermissionDenied
		}
	}
	return centerBankIDs(banks), banks, nil
}

func (s *examEvaluationCenterService) authorizedCenterRun(
	ctx context.Context,
	tenantID uint64,
	userID string,
	runID string,
) (*types.ExamRAGEvaluationRun, error) {
	run, err := s.repo.GetCenterRun(ctx, tenantID, strings.TrimSpace(runID))
	if err != nil {
		return nil, ErrExamNotFound
	}
	if _, err := s.questions.GetQuestionBank(ctx, tenantID, userID, run.QuestionBankID); err != nil {
		return nil, err
	}
	return run, nil
}

func normalizeEvaluationCenterFilter(filter types.ExamEvaluationCenterFilter) (types.ExamEvaluationCenterFilter, error) {
	if filter.Kind == "" {
		filter.Kind = types.ExamEvaluationKindRAG
	}
	if filter.Kind != types.ExamEvaluationKindRAG && filter.Kind != types.ExamEvaluationKindAgent {
		return filter, ErrExamInvalidRequest
	}
	switch filter.Status {
	case "", types.ExamRAGEvaluationRunStatusQueued, types.ExamRAGEvaluationRunStatusRunning,
		types.ExamRAGEvaluationRunStatusCompleted, types.ExamRAGEvaluationRunStatusFailed:
	default:
		return filter, ErrExamInvalidRequest
	}
	if filter.Regression == "" {
		filter.Regression = types.ExamEvaluationRegressionAll
	}
	switch filter.Regression {
	case types.ExamEvaluationRegressionAll, types.ExamEvaluationRegressionRegressed,
		types.ExamEvaluationRegressionStable, types.ExamEvaluationRegressionUncompared:
	default:
		return filter, ErrExamInvalidRequest
	}
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	return filter, nil
}

func selectVisibleCenterBank(banks []*types.QuestionBank, bankID string) []*types.QuestionBank {
	for _, bank := range banks {
		if bank != nil && bank.ID == bankID {
			return []*types.QuestionBank{bank}
		}
	}
	return []*types.QuestionBank{}
}

func centerBankIDs(banks []*types.QuestionBank) []string {
	ids := make([]string, 0, len(banks))
	for _, bank := range banks {
		if bank != nil {
			ids = append(ids, bank.ID)
		}
	}
	return ids
}

func collectEvaluationAgentOptions(runs []*types.ExamRAGEvaluationRun) []types.ExamEvaluationAgentOption {
	names := make(map[string]string)
	for _, run := range runs {
		if run == nil || run.AgentID == "" {
			continue
		}
		name := evaluationRunAgentName(run)
		if name == "" {
			name = run.AgentID
		}
		names[run.AgentID] = name
	}
	options := make([]types.ExamEvaluationAgentOption, 0, len(names))
	for id, name := range names {
		options = append(options, types.ExamEvaluationAgentOption{ID: id, Name: name})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Name < options[j].Name })
	return options
}

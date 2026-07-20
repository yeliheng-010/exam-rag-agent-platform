package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

func TestEvaluationCenterScopesContributorAndAdminRuns(t *testing.T) {
	repo := newEvaluationCenterRepositoryStub()
	repo.banks = []*types.QuestionBank{
		{ID: "bank-1", TenantID: 10000, Name: "英语题库"},
		{ID: "bank-2", TenantID: 10000, Name: "数学题库"},
	}
	repo.runs = []*types.ExamRAGEvaluationRun{
		completedRAGCenterRun("run-1", "bank-1", 0.9, 100),
		completedRAGCenterRun("run-2", "bank-2", 0.8, 120),
	}
	questions := &evaluationCenterQuestionServiceStub{visible: []*types.QuestionBank{repo.banks[0]}}
	svc := NewExamEvaluationCenterService(repo, questions, nil, nil)

	contributorCtx := evaluationCenterRoleContext(types.TenantRoleContributor)
	result, err := svc.GetCenter(contributorCtx, 10000, "teacher-1", types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKindRAG, Limit: 20,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"bank-1"}, repo.lastBankIDs)
	require.Len(t, result.Items, 1)
	require.Equal(t, "英语题库", result.Items[0].QuestionBankName)

	adminCtx := evaluationCenterRoleContext(types.TenantRoleAdmin)
	result, err = svc.GetCenter(adminCtx, 10000, "admin-1", types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKindRAG, Limit: 20,
	})
	require.NoError(t, err)
	require.Nil(t, repo.lastBankIDs)
	require.Len(t, result.Items, 2)
}

func TestEvaluationCenterComputesDeterministicRegressionAgainstBaseline(t *testing.T) {
	repo := newEvaluationCenterRepositoryStub()
	repo.banks = []*types.QuestionBank{{ID: "bank-1", TenantID: 10000, Name: "英语题库"}}
	baseline := completedRAGCenterRun("run-base", "bank-1", 0.9, 100)
	baseline.IsBaseline = true
	current := completedRAGCenterRun("run-current", "bank-1", 0.85, 260)
	repo.runs = []*types.ExamRAGEvaluationRun{current, baseline}
	repo.baselines = []*types.ExamRAGEvaluationRun{baseline}
	questions := &evaluationCenterQuestionServiceStub{visible: repo.banks}
	svc := NewExamEvaluationCenterService(repo, questions, nil, nil)

	result, err := svc.GetCenter(
		evaluationCenterRoleContext(types.TenantRoleContributor),
		10000,
		"teacher-1",
		types.ExamEvaluationCenterFilter{Kind: types.ExamEvaluationKindRAG, Regression: types.ExamEvaluationRegressionRegressed, Limit: 20},
	)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	item := result.Items[0]
	require.Equal(t, "run-current", item.Run.ID)
	require.Equal(t, types.ExamEvaluationComparisonRegressed, item.ComparisonStatus)
	require.InDelta(t, -0.05, item.MetricDeltas["pass_rate"], 0.0001)
	require.InDelta(t, 160.0, item.MetricDeltas["average_duration_ms"], 0.0001)
	reasonMetrics := make(map[string]bool, len(item.RegressionReasons))
	for _, reason := range item.RegressionReasons {
		reasonMetrics[reason.Metric] = true
	}
	require.True(t, reasonMetrics["pass_rate"])
	require.True(t, reasonMetrics["average_duration_ms"])
	require.Equal(t, 1, result.Summary.RegressionRuns)
}

func TestEvaluationCenterCreatesVersionsAndRunsStoredRAGAndAgentDefinitions(t *testing.T) {
	repo := newEvaluationCenterRepositoryStub()
	repo.banks = []*types.QuestionBank{{ID: "bank-1", TenantID: 10000, Name: "综合题库"}}
	ragSource := completedRAGCenterRun("rag-source", "bank-1", 1, 100)
	ragSource.RequestSnapshot = types.JSON(`{"knowledge_base_ids":["kb-1"],"cases":[{"name":"A篇","query":"第一篇"}],"match_count":8}`)
	agentSource := completedAgentCenterRun("agent-source", "bank-1", "agent-1")
	repo.runs = []*types.ExamRAGEvaluationRun{ragSource, agentSource}
	questions := &evaluationCenterQuestionServiceStub{visible: repo.banks}
	ragRunner := &evaluationCenterRAGRunnerStub{}
	agentRunner := &evaluationCenterAgentRunnerStub{}
	svc := NewExamEvaluationCenterService(repo, questions, ragRunner, agentRunner)
	ctx := evaluationCenterRoleContext(types.TenantRoleContributor)

	ragSet, err := svc.CreateEvaluationSet(ctx, 10000, "teacher-1", &types.CreateExamEvaluationSetRequest{
		RunID: "rag-source", Name: "英语基础回归", Description: "阅读理解固定案例",
	})
	require.NoError(t, err)
	require.Equal(t, 1, ragSet.CurrentVersion)
	require.Len(t, ragSet.Versions, 1)

	run, err := svc.RunEvaluationSet(ctx, 10000, "teacher-1", ragSet.ID, &types.RunExamEvaluationSetRequest{})
	require.NoError(t, err)
	require.Equal(t, "kb-1", ragRunner.request.KnowledgeBaseIDs[0])
	require.Equal(t, ragSet.ID, run.EvaluationSetID)
	require.Equal(t, 1, run.EvaluationSetVersion)

	agentSet, err := svc.CreateEvaluationSet(ctx, 10000, "teacher-1", &types.CreateExamEvaluationSetRequest{
		RunID: "agent-source", Name: "诊断 Agent 回归",
	})
	require.NoError(t, err)
	_, err = svc.RunEvaluationSet(ctx, 10000, "teacher-1", agentSet.ID, &types.RunExamEvaluationSetRequest{})
	require.NoError(t, err)
	require.Empty(t, agentRunner.request.AgentID)
	require.Equal(t, "agent-1", agentRunner.snapshot.Agent.ID)
	require.Equal(t, "snapshot-model-v1", agentRunner.snapshot.Agent.ModelID)
	require.Equal(t, "snapshot-model-v1", agentRunner.snapshot.Agent.Config.ModelID)
	require.Len(t, agentRunner.snapshot.Cases, 1)

	second, err := svc.CreateEvaluationSetVersion(ctx, 10000, "teacher-1", ragSet.ID, &types.CreateExamEvaluationSetVersionRequest{RunID: "rag-source"})
	require.NoError(t, err)
	require.Equal(t, 2, second.Version)

	_, err = svc.CreateEvaluationSetVersion(ctx, 10000, "teacher-1", ragSet.ID, &types.CreateExamEvaluationSetVersionRequest{RunID: "agent-source"})
	require.ErrorIs(t, err, ErrExamInvalidRequest)
}

func completedRAGCenterRun(id, bankID string, passRate, duration float64) *types.ExamRAGEvaluationRun {
	result, _ := json.Marshal(types.ExamRAGDiagnosticResult{Summary: types.ExamRAGDiagnosticSummary{
		HitRate: passRate, RetrievalHitRate: passRate, AnswerHitRate: passRate,
		RecallAtK: passRate, MeanReciprocalRank: passRate, StructuredResolutionRate: passRate,
		AverageDurationMS: duration,
	}})
	return &types.ExamRAGEvaluationRun{
		ID: id, TenantID: 10000, QuestionBankID: bankID, EvaluationKind: types.ExamEvaluationKindRAG,
		CreatedBy: "teacher-1", Status: types.ExamRAGEvaluationRunStatusCompleted,
		RequestSnapshot: types.JSON(`{"cases":[{"name":"case-1","query":"query"}]}`),
		ResultSnapshot:  result, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
}

func completedAgentCenterRun(id, bankID, agentID string) *types.ExamRAGEvaluationRun {
	request, _ := json.Marshal(types.ExamAgentEvaluationRequestSnapshot{
		Agent: types.ExamAgentSnapshot{
			ID: agentID, Name: "诊断助手", ModelID: "snapshot-model-v1",
			AllowedTools: []string{"exam_question_context"},
			Config: types.CustomAgentConfig{
				AgentMode: types.AgentModeSmartReasoning, ModelID: "snapshot-model-v1",
				AllowedTools:     []string{"exam_question_context"},
				MCPSelectionMode: "none", SkillsSelectionMode: "none",
			},
		},
		Cases: []types.ExamAgentEvaluationCase{{Name: "班级诊断", Input: "诊断班级"}},
	})
	result, _ := json.Marshal(types.ExamAgentEvaluationResult{
		Agent:   types.ExamAgentSnapshot{ID: agentID, Name: "诊断助手"},
		Summary: types.ExamAgentEvaluationSummary{PassRate: 1, AverageDurationMS: 100},
	})
	return &types.ExamRAGEvaluationRun{
		ID: id, TenantID: 10000, QuestionBankID: bankID, EvaluationKind: types.ExamEvaluationKindAgent,
		AgentID: agentID, CreatedBy: "teacher-1", Status: types.ExamRAGEvaluationRunStatusCompleted,
		RequestSnapshot: request, ResultSnapshot: result, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
}

func evaluationCenterRoleContext(role types.TenantRole) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantRoleContextKey, role)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, uint64(10000))
	return context.WithValue(ctx, types.UserIDContextKey, "teacher-1")
}

type evaluationCenterQuestionServiceStub struct {
	interfaces.ExamQuestionService
	visible []*types.QuestionBank
}

func (s *evaluationCenterQuestionServiceStub) ListQuestionBanks(context.Context, uint64, string, string) ([]*types.QuestionBank, error) {
	return s.visible, nil
}

func (s *evaluationCenterQuestionServiceStub) GetQuestionBank(_ context.Context, tenantID uint64, _ string, bankID string) (*types.QuestionBank, error) {
	for _, bank := range s.visible {
		if bank != nil && bank.TenantID == tenantID && bank.ID == bankID {
			return bank, nil
		}
	}
	return nil, ErrExamPermissionDenied
}

type evaluationCenterRepositoryStub struct {
	interfaces.ExamEvaluationCenterRepository
	runs        []*types.ExamRAGEvaluationRun
	baselines   []*types.ExamRAGEvaluationRun
	banks       []*types.QuestionBank
	sets        map[string]*types.ExamEvaluationSet
	versions    map[string][]*types.ExamEvaluationSetVersion
	lastBankIDs []string
}

func newEvaluationCenterRepositoryStub() *evaluationCenterRepositoryStub {
	return &evaluationCenterRepositoryStub{
		sets: map[string]*types.ExamEvaluationSet{}, versions: map[string][]*types.ExamEvaluationSetVersion{},
	}
}

func (r *evaluationCenterRepositoryStub) ListCenterQuestionBanks(_ context.Context, _ uint64, bankIDs []string) ([]*types.QuestionBank, error) {
	return filterCenterBanks(r.banks, bankIDs), nil
}

func (r *evaluationCenterRepositoryStub) ListCenterRuns(_ context.Context, tenantID uint64, bankIDs []string, filter types.ExamEvaluationCenterFilter) ([]*types.ExamRAGEvaluationRun, error) {
	r.lastBankIDs = append([]string(nil), bankIDs...)
	if bankIDs == nil {
		r.lastBankIDs = nil
	}
	var out []*types.ExamRAGEvaluationRun
	for _, run := range r.runs {
		if run.TenantID != tenantID || run.EvaluationKind != filter.Kind || !centerBankAllowed(run.QuestionBankID, bankIDs) {
			continue
		}
		out = append(out, run)
	}
	return out, nil
}

func (r *evaluationCenterRepositoryStub) ListEvaluationSets(_ context.Context, tenantID uint64, bankIDs []string, filter types.ExamEvaluationCenterFilter) ([]*types.ExamEvaluationSet, error) {
	var out []*types.ExamEvaluationSet
	for _, set := range r.sets {
		if set.TenantID == tenantID && set.EvaluationKind == filter.Kind && centerBankAllowed(set.QuestionBankID, bankIDs) {
			out = append(out, set)
		}
	}
	return out, nil
}

func (r *evaluationCenterRepositoryStub) ListEvaluationSetVersions(_ context.Context, tenantID uint64, setID string) ([]*types.ExamEvaluationSetVersion, error) {
	if set := r.sets[setID]; set == nil || set.TenantID != tenantID {
		return nil, ErrExamNotFound
	}
	return r.versions[setID], nil
}

func (r *evaluationCenterRepositoryStub) ListBaselines(_ context.Context, tenantID uint64, bankIDs []string, kind types.ExamEvaluationKind, _ string) ([]*types.ExamRAGEvaluationRun, error) {
	var out []*types.ExamRAGEvaluationRun
	for _, run := range r.baselines {
		if run.TenantID == tenantID && run.EvaluationKind == kind && centerBankAllowed(run.QuestionBankID, bankIDs) {
			out = append(out, run)
		}
	}
	return out, nil
}

func (r *evaluationCenterRepositoryStub) GetCenterRun(_ context.Context, tenantID uint64, runID string) (*types.ExamRAGEvaluationRun, error) {
	for _, run := range r.runs {
		if run.TenantID == tenantID && run.ID == runID {
			return run, nil
		}
	}
	return nil, ErrExamNotFound
}

func (r *evaluationCenterRepositoryStub) CreateEvaluationSet(_ context.Context, set *types.ExamEvaluationSet, version *types.ExamEvaluationSetVersion) error {
	r.sets[set.ID] = set
	r.versions[set.ID] = []*types.ExamEvaluationSetVersion{version}
	return nil
}

func (r *evaluationCenterRepositoryStub) GetEvaluationSet(_ context.Context, tenantID uint64, setID string) (*types.ExamEvaluationSet, error) {
	set := r.sets[setID]
	if set == nil || set.TenantID != tenantID {
		return nil, ErrExamNotFound
	}
	return set, nil
}

func (r *evaluationCenterRepositoryStub) GetEvaluationSetVersion(_ context.Context, tenantID uint64, setID string, version int) (*types.ExamEvaluationSetVersion, error) {
	for _, item := range r.versions[setID] {
		if item.TenantID == tenantID && item.Version == version {
			return item, nil
		}
	}
	return nil, ErrExamNotFound
}

func (r *evaluationCenterRepositoryStub) CreateEvaluationSetVersion(_ context.Context, tenantID uint64, setID string, item *types.ExamEvaluationSetVersion) (*types.ExamEvaluationSetVersion, error) {
	set := r.sets[setID]
	if set == nil || set.TenantID != tenantID {
		return nil, ErrExamNotFound
	}
	item.TenantID = tenantID
	item.EvaluationSetID = setID
	item.Version = set.CurrentVersion + 1
	set.CurrentVersion = item.Version
	r.versions[setID] = append(r.versions[setID], item)
	return item, nil
}

func (r *evaluationCenterRepositoryStub) AttachRunEvaluationSet(_ context.Context, _ uint64, runID, setID string, version int) error {
	for _, run := range r.runs {
		if run.ID == runID {
			run.EvaluationSetID = setID
			run.EvaluationSetVersion = version
		}
	}
	return nil
}

func filterCenterBanks(banks []*types.QuestionBank, bankIDs []string) []*types.QuestionBank {
	var out []*types.QuestionBank
	for _, bank := range banks {
		if bank != nil && centerBankAllowed(bank.ID, bankIDs) {
			out = append(out, bank)
		}
	}
	return out
}

func centerBankAllowed(bankID string, bankIDs []string) bool {
	if bankIDs == nil {
		return true
	}
	for _, allowed := range bankIDs {
		if allowed == bankID {
			return true
		}
	}
	return false
}

type evaluationCenterRAGRunnerStub struct {
	interfaces.ExamRAGEvaluationService
	request types.RunExamRAGDiagnosticRequest
}

func (s *evaluationCenterRAGRunnerStub) CreateRun(_ context.Context, tenantID uint64, userID, bankID string, req *types.RunExamRAGDiagnosticRequest) (*types.ExamRAGEvaluationRun, error) {
	s.request = *req
	return &types.ExamRAGEvaluationRun{
		ID: "rag-rerun", TenantID: tenantID, QuestionBankID: bankID, EvaluationKind: types.ExamEvaluationKindRAG,
		CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
	}, nil
}

type evaluationCenterAgentRunnerStub struct {
	interfaces.ExamAgentEvaluationService
	request  types.RunExamAgentEvaluationRequest
	snapshot types.ExamAgentEvaluationRequestSnapshot
}

func (s *evaluationCenterAgentRunnerStub) CreateRun(_ context.Context, tenantID uint64, userID, bankID string, req *types.RunExamAgentEvaluationRequest) (*types.ExamRAGEvaluationRun, error) {
	s.request = *req
	return &types.ExamRAGEvaluationRun{
		ID: "agent-rerun", TenantID: tenantID, QuestionBankID: bankID, EvaluationKind: types.ExamEvaluationKindAgent,
		AgentID: req.AgentID, CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
	}, nil
}

func (s *evaluationCenterAgentRunnerStub) CreateRunFromSnapshot(
	_ context.Context,
	tenantID uint64,
	userID string,
	bankID string,
	snapshot *types.ExamAgentEvaluationRequestSnapshot,
) (*types.ExamRAGEvaluationRun, error) {
	s.snapshot = *snapshot
	return &types.ExamRAGEvaluationRun{
		ID: "agent-rerun", TenantID: tenantID, QuestionBankID: bankID, EvaluationKind: types.ExamEvaluationKindAgent,
		AgentID: snapshot.Agent.ID, CreatedBy: userID, Status: types.ExamRAGEvaluationRunStatusQueued,
	}, nil
}

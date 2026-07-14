package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
)

func TestExamRAGEvaluationServiceCreatesQueuedRun(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	diagnostic := &examRAGEvaluationDiagnosticStub{preparation: testRAGPreparation()}
	enqueuer := &examRAGEvaluationEnqueuerStub{}
	svc := NewExamRAGEvaluationService(repo, testRAGQuestionService(), diagnostic, testRAGTenantService(), enqueuer)

	run, err := svc.CreateRun(context.Background(), 10000, "teacher-1", "bank-1", &types.RunExamRAGDiagnosticRequest{})

	require.NoError(t, err)
	require.Equal(t, types.ExamRAGEvaluationRunStatusQueued, run.Status)
	require.Equal(t, "bank-1", run.QuestionBankID)
	require.NotEmpty(t, run.ID)
	require.NotNil(t, enqueuer.task)
	require.Equal(t, types.TypeExamRAGEvaluationRun, enqueuer.task.Type())
	var request types.RunExamRAGDiagnosticRequest
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &request))
	require.Equal(t, 12, request.MatchCount)
	require.Equal(t, []string{"kb-1"}, request.KnowledgeBaseIDs)
	require.Len(t, request.Cases, 2)
	var snapshot struct {
		UsedDefaultCases bool `json:"used_default_cases"`
	}
	require.NoError(t, json.Unmarshal(run.RequestSnapshot, &snapshot))
	require.True(t, snapshot.UsedDefaultCases)
}

func TestExamRAGEvaluationServiceScopesReadByQuestionBankPermission(t *testing.T) {
	repo := newExamRAGEvaluationRunRepoStub()
	run := testQueuedRAGEvaluationRun("run-1")
	run.ResultSnapshot, _ = json.Marshal(types.ExamRAGDiagnosticResult{
		Cases: []types.ExamRAGDiagnosticCase{{Name: "case-1", Query: "query one"}},
		Summary: types.ExamRAGDiagnosticSummary{
			Total: 1, Passed: 1, HitRate: 1,
			Results: []types.ExamRAGDiagnosticResultItem{{Name: "case-1", Passed: true}},
		},
	})
	repo.runs["run-1"] = run
	questionSvc := testRAGQuestionService()
	svc := NewExamRAGEvaluationService(repo, questionSvc, &examRAGEvaluationDiagnosticStub{}, testRAGTenantService(), &examRAGEvaluationEnqueuerStub{})

	runs, err := svc.ListRuns(context.Background(), 10000, "teacher-1", "bank-1", 20)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	var compactResult types.ExamRAGDiagnosticResult
	require.NoError(t, json.Unmarshal(runs[0].ResultSnapshot, &compactResult))
	require.Equal(t, 1, compactResult.Summary.Passed)
	require.Empty(t, compactResult.Summary.Results)
	require.Empty(t, compactResult.Cases)
	require.NotEmpty(t, repo.runs["run-1"].ResultSnapshot)
	_, err = svc.GetRun(context.Background(), 10000, "teacher-1", "bank-2", "run-1")
	require.ErrorIs(t, err, repository.ErrQuestionBankNotFound)
}

func testRAGQuestionService() *stubExamRAGDiagnosticQuestionService {
	return &stubExamRAGDiagnosticQuestionService{bank: &types.QuestionBank{
		ID: "bank-1", TenantID: 10000, SpaceID: "space-1", DomainID: "gaokao",
	}}
}

func testRAGTenantService() *examRAGEvaluationTenantServiceStub {
	return &examRAGEvaluationTenantServiceStub{tenant: &types.Tenant{ID: 10000}}
}

func testRAGPreparation() *types.ExamRAGDiagnosticPreparation {
	vectorThreshold := 0.45
	keywordThreshold := 0.25
	return &types.ExamRAGDiagnosticPreparation{
		QuestionBank: &types.QuestionBank{ID: "bank-1", TenantID: 10000},
		Request: types.RunExamRAGDiagnosticRequest{
			KnowledgeBaseIDs: []string{"kb-1"},
			Cases: []types.ExamRAGDiagnosticCase{
				{Name: "case-1", Query: "query one"},
				{Name: "case-2", Query: "query two"},
			},
			MatchCount:       12,
			VectorThreshold:  &vectorThreshold,
			KeywordThreshold: &keywordThreshold,
		},
		UsedDefaultCases: true,
	}
}

type examRAGEvaluationDiagnosticStub struct {
	interfaces.ExamRAGDiagnosticService
	preparation *types.ExamRAGDiagnosticPreparation
	result      *types.ExamRAGDiagnosticResult
	err         error
	calls       int
	tenantInfo  *types.Tenant
}

func (s *examRAGEvaluationDiagnosticStub) PrepareQuestionBank(
	context.Context, uint64, string, string, *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGDiagnosticPreparation, error) {
	if s.preparation == nil {
		return nil, ErrExamInvalidRequest
	}
	return s.preparation, nil
}

func (s *examRAGEvaluationDiagnosticStub) EvaluateQuestionBank(
	ctx context.Context, _ uint64, _ string, _ string, _ *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGDiagnosticResult, error) {
	s.calls++
	s.tenantInfo, _ = types.TenantInfoFromContext(ctx)
	return s.result, s.err
}

type examRAGEvaluationTenantServiceStub struct {
	interfaces.TenantService
	tenant *types.Tenant
	err    error
}

func (s *examRAGEvaluationTenantServiceStub) GetTenantByID(context.Context, uint64) (*types.Tenant, error) {
	return s.tenant, s.err
}

type examRAGEvaluationEnqueuerStub struct {
	task *asynq.Task
	err  error
}

func (s *examRAGEvaluationEnqueuerStub) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	s.task = task
	if s.err != nil {
		return nil, s.err
	}
	return &asynq.TaskInfo{ID: "task-1"}, nil
}

type examRAGEvaluationRunRepoStub struct {
	runs map[string]*types.ExamRAGEvaluationRun
}

func newExamRAGEvaluationRunRepoStub() *examRAGEvaluationRunRepoStub {
	return &examRAGEvaluationRunRepoStub{runs: map[string]*types.ExamRAGEvaluationRun{}}
}

func (r *examRAGEvaluationRunRepoStub) CreateRun(_ context.Context, run *types.ExamRAGEvaluationRun) error {
	r.runs[run.ID] = run
	return nil
}

func (r *examRAGEvaluationRunRepoStub) GetRun(_ context.Context, tenantID uint64, bankID string, runID string) (*types.ExamRAGEvaluationRun, error) {
	run := r.runs[runID]
	if run == nil || run.TenantID != tenantID || run.QuestionBankID != bankID {
		return nil, repository.ErrExamRAGEvaluationRunNotFound
	}
	return run, nil
}

func (r *examRAGEvaluationRunRepoStub) GetRunForTask(_ context.Context, runID string) (*types.ExamRAGEvaluationRun, error) {
	run := r.runs[runID]
	if run == nil {
		return nil, repository.ErrExamRAGEvaluationRunNotFound
	}
	return run, nil
}

func (r *examRAGEvaluationRunRepoStub) ListRuns(_ context.Context, tenantID uint64, bankID string, _ int) ([]*types.ExamRAGEvaluationRun, error) {
	var runs []*types.ExamRAGEvaluationRun
	for _, run := range r.runs {
		if run.TenantID == tenantID && run.QuestionBankID == bankID {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func (r *examRAGEvaluationRunRepoStub) UpdateRun(_ context.Context, tenantID uint64, bankID string, run *types.ExamRAGEvaluationRun) error {
	if run == nil || run.TenantID != tenantID || run.QuestionBankID != bankID {
		return repository.ErrExamRAGEvaluationRunNotFound
	}
	r.runs[run.ID] = run
	return nil
}

var errRAGEvaluationWorker = errors.New("evaluation unavailable")

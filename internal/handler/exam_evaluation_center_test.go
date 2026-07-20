package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExamEvaluationCenterHandlerParsesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &examEvaluationCenterHandlerServiceStub{center: &types.ExamEvaluationCenterResult{}}
	h := NewExamEvaluationCenterHandler(svc)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/exam/evaluation-center?kind=agent&bank_id=bank-1&agent_id=agent-1&status=completed&regression=regressed&limit=25", nil)
	setExamEvaluationHandlerIdentity(ctx)

	h.GetCenter(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, types.ExamEvaluationKindAgent, svc.filter.Kind)
	require.Equal(t, "bank-1", svc.filter.BankID)
	require.Equal(t, "agent-1", svc.filter.AgentID)
	require.Equal(t, types.ExamRAGEvaluationRunStatusCompleted, svc.filter.Status)
	require.Equal(t, types.ExamEvaluationRegressionRegressed, svc.filter.Regression)
	require.Equal(t, 25, svc.filter.Limit)
}

func TestExamEvaluationCenterHandlerCreatesSetAndQueuesVersionRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &examEvaluationCenterHandlerServiceStub{
		set: &types.ExamEvaluationSet{ID: "set-1", Name: "英语回归集"},
		run: &types.ExamRAGEvaluationRun{ID: "run-new", Status: types.ExamRAGEvaluationRunStatusQueued},
	}
	h := NewExamEvaluationCenterHandler(svc)

	createRecorder := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(createRecorder)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/exam/evaluation-sets", strings.NewReader(`{"run_id":"run-source","name":"英语回归集"}`))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	setExamEvaluationHandlerIdentity(createCtx)
	h.CreateEvaluationSet(createCtx)
	require.Equal(t, http.StatusCreated, createRecorder.Code)
	require.Equal(t, "run-source", svc.createSetRequest.RunID)

	runRecorder := httptest.NewRecorder()
	runCtx, _ := gin.CreateTestContext(runRecorder)
	runCtx.Request = httptest.NewRequest(http.MethodPost, "/exam/evaluation-sets/set-1/runs", strings.NewReader(`{"version":2}`))
	runCtx.Request.Header.Set("Content-Type", "application/json")
	runCtx.Params = gin.Params{{Key: "set_id", Value: "set-1"}}
	setExamEvaluationHandlerIdentity(runCtx)
	h.RunEvaluationSet(runCtx)
	require.Equal(t, http.StatusAccepted, runRecorder.Code)
	require.Equal(t, 2, svc.runSetRequest.Version)
}

func setExamEvaluationHandlerIdentity(ctx *gin.Context) {
	ctx.Set(types.TenantIDContextKey.String(), uint64(10000))
	ctx.Set(types.UserIDContextKey.String(), "teacher-1")
}

type examEvaluationCenterHandlerServiceStub struct {
	interfaces.ExamEvaluationCenterService
	center           *types.ExamEvaluationCenterResult
	set              *types.ExamEvaluationSet
	run              *types.ExamRAGEvaluationRun
	filter           types.ExamEvaluationCenterFilter
	createSetRequest types.CreateExamEvaluationSetRequest
	runSetRequest    types.RunExamEvaluationSetRequest
}

func (s *examEvaluationCenterHandlerServiceStub) GetCenter(_ context.Context, _ uint64, _ string, filter types.ExamEvaluationCenterFilter) (*types.ExamEvaluationCenterResult, error) {
	s.filter = filter
	return s.center, nil
}

func (s *examEvaluationCenterHandlerServiceStub) CreateEvaluationSet(_ context.Context, _ uint64, _ string, req *types.CreateExamEvaluationSetRequest) (*types.ExamEvaluationSet, error) {
	s.createSetRequest = *req
	return s.set, nil
}

func (s *examEvaluationCenterHandlerServiceStub) RunEvaluationSet(_ context.Context, _ uint64, _ string, _ string, req *types.RunExamEvaluationSetRequest) (*types.ExamRAGEvaluationRun, error) {
	s.runSetRequest = *req
	return s.run, nil
}

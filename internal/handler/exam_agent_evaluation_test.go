package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExamAgentEvaluationHandlerCreateReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &examAgentEvaluationHandlerServiceStub{run: &types.ExamRAGEvaluationRun{
		ID: "agent-run-1", QuestionBankID: "bank-1", EvaluationKind: types.ExamEvaluationKindAgent,
		Status: types.ExamRAGEvaluationRunStatusQueued,
	}}
	h := NewExamAgentEvaluationHandler(svc)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/exam/question-banks/bank-1/agent-evaluation-runs",
		strings.NewReader(`{"agent_id":"agent-1","cases":[{"name":"case-1","input":"diagnose"}]}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "bank_id", Value: "bank-1"}}
	ctx.Set(types.TenantIDContextKey.String(), uint64(10000))
	ctx.Set(types.UserIDContextKey.String(), "teacher-1")

	h.CreateRun(ctx)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	require.Equal(t, "agent-1", svc.request.AgentID)
	require.Len(t, svc.request.Cases, 1)
	var response struct {
		Success bool                        `json:"success"`
		Data    *types.ExamRAGEvaluationRun `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, "agent-run-1", response.Data.ID)
}

type examAgentEvaluationHandlerServiceStub struct {
	interfaces.ExamAgentEvaluationService
	run     *types.ExamRAGEvaluationRun
	request types.RunExamAgentEvaluationRequest
}

func (s *examAgentEvaluationHandlerServiceStub) CreateRun(
	_ context.Context,
	_ uint64,
	_ string,
	_ string,
	req *types.RunExamAgentEvaluationRequest,
) (*types.ExamRAGEvaluationRun, error) {
	s.request = *req
	return s.run, nil
}

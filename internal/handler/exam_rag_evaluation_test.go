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

func TestExamRAGEvaluationHandlerCreateReturnsAccepted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &examRAGEvaluationHandlerServiceStub{run: &types.ExamRAGEvaluationRun{
		ID: "run-1", QuestionBankID: "bank-1", Status: types.ExamRAGEvaluationRunStatusQueued,
	}}
	handler := NewExamRAGEvaluationHandler(svc)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/exam/question-banks/bank-1/rag-evaluation-runs", strings.NewReader(`{}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "bank_id", Value: "bank-1"}}
	ctx.Set(types.TenantIDContextKey.String(), uint64(10000))
	ctx.Set(types.UserIDContextKey.String(), "teacher-1")

	handler.CreateRun(ctx)

	require.Equal(t, http.StatusAccepted, recorder.Code)
	var response struct {
		Success bool                        `json:"success"`
		Data    *types.ExamRAGEvaluationRun `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, "run-1", response.Data.ID)
}

type examRAGEvaluationHandlerServiceStub struct {
	interfaces.ExamRAGEvaluationService
	run *types.ExamRAGEvaluationRun
}

func (s *examRAGEvaluationHandlerServiceStub) CreateRun(
	context.Context, uint64, string, string, *types.RunExamRAGDiagnosticRequest,
) (*types.ExamRAGEvaluationRun, error) {
	return s.run, nil
}

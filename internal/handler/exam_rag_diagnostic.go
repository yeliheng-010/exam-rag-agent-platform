package handler

import (
	"errors"
	"io"
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamRAGDiagnosticHandler struct {
	diagnosticService interfaces.ExamRAGDiagnosticService
}

func NewExamRAGDiagnosticHandler(diagnosticService interfaces.ExamRAGDiagnosticService) *ExamRAGDiagnosticHandler {
	return &ExamRAGDiagnosticHandler{diagnosticService: diagnosticService}
}

func (h *ExamRAGDiagnosticHandler) EvaluateQuestionBank(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	bankID := c.Param("bank_id")

	var req types.RunExamRAGDiagnosticRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.diagnosticService.EvaluateQuestionBank(ctx, tenantID, userID, bankID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to evaluate exam RAG diagnostics: %v", err)
		writeExamError(c, err, "Failed to evaluate exam RAG diagnostics")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

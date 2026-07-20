package handler

import (
	"net/http"
	"strconv"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamAgentEvaluationHandler struct {
	service interfaces.ExamAgentEvaluationService
}

func NewExamAgentEvaluationHandler(service interfaces.ExamAgentEvaluationService) *ExamAgentEvaluationHandler {
	return &ExamAgentEvaluationHandler{service: service}
}

func (h *ExamAgentEvaluationHandler) CreateRun(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.RunExamAgentEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	run, err := h.service.CreateRun(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("bank_id"),
		&req,
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to create exam Agent evaluation run: %v", err)
		writeExamError(c, err, "Failed to create exam Agent evaluation run")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ExamAgentEvaluationHandler) ListRuns(c *gin.Context) {
	ctx := c.Request.Context()
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	runs, err := h.service.ListRuns(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("bank_id"),
		limit,
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam Agent evaluation runs: %v", err)
		writeExamError(c, err, "Failed to list exam Agent evaluation runs")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": runs})
}

func (h *ExamAgentEvaluationHandler) GetRun(c *gin.Context) {
	ctx := c.Request.Context()
	run, err := h.service.GetRun(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("bank_id"),
		c.Param("run_id"),
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to get exam Agent evaluation run: %v", err)
		writeExamError(c, err, "Failed to get exam Agent evaluation run")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

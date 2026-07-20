package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamEvaluationCenterHandler struct {
	service interfaces.ExamEvaluationCenterService
}

func NewExamEvaluationCenterHandler(service interfaces.ExamEvaluationCenterService) *ExamEvaluationCenterHandler {
	return &ExamEvaluationCenterHandler{service: service}
}

func (h *ExamEvaluationCenterHandler) GetCenter(c *gin.Context) {
	filter := evaluationCenterFilterFromQuery(c)
	result, err := h.service.GetCenter(
		c.Request.Context(), examTenantID(c), examUserID(c), filter,
	)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to get evaluation center")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamEvaluationCenterHandler) SetBaseline(c *gin.Context) {
	var req types.SetExamEvaluationBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	run, err := h.service.SetBaseline(c.Request.Context(), examTenantID(c), examUserID(c), req.RunID)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to set evaluation baseline")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ExamEvaluationCenterHandler) ListEvaluationSets(c *gin.Context) {
	sets, err := h.service.ListEvaluationSets(
		c.Request.Context(), examTenantID(c), examUserID(c), evaluationCenterFilterFromQuery(c),
	)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to list evaluation sets")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sets})
}

func (h *ExamEvaluationCenterHandler) CreateEvaluationSet(c *gin.Context) {
	var req types.CreateExamEvaluationSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	set, err := h.service.CreateEvaluationSet(c.Request.Context(), examTenantID(c), examUserID(c), &req)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to create evaluation set")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": set})
}

func (h *ExamEvaluationCenterHandler) CreateEvaluationSetVersion(c *gin.Context) {
	var req types.CreateExamEvaluationSetVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	version, err := h.service.CreateEvaluationSetVersion(
		c.Request.Context(), examTenantID(c), examUserID(c), c.Param("set_id"), &req,
	)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to create evaluation set version")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": version})
}

func (h *ExamEvaluationCenterHandler) RunEvaluationSet(c *gin.Context) {
	var req types.RunExamEvaluationSetRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	run, err := h.service.RunEvaluationSet(
		c.Request.Context(), examTenantID(c), examUserID(c), c.Param("set_id"), &req,
	)
	if err != nil {
		writeEvaluationCenterError(c, err, "Failed to run evaluation set")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func evaluationCenterFilterFromQuery(c *gin.Context) types.ExamEvaluationCenterFilter {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	return types.ExamEvaluationCenterFilter{
		Kind: types.ExamEvaluationKind(c.Query("kind")), BankID: c.Query("bank_id"),
		AgentID: c.Query("agent_id"), Status: types.ExamRAGEvaluationRunStatus(c.Query("status")),
		Regression: types.ExamEvaluationRegressionFilter(c.Query("regression")), Limit: limit,
	}
}

func examTenantID(c *gin.Context) uint64 { return c.GetUint64(types.TenantIDContextKey.String()) }
func examUserID(c *gin.Context) string   { return c.GetString(types.UserIDContextKey.String()) }

func writeEvaluationCenterError(c *gin.Context, err error, fallback string) {
	logger.Errorf(c.Request.Context(), "%s: %v", fallback, err)
	writeExamError(c, err, fallback)
}

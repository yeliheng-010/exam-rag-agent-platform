package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamQuestionHandler struct {
	questionService interfaces.ExamQuestionService
}

func NewExamQuestionHandler(questionService interfaces.ExamQuestionService) *ExamQuestionHandler {
	return &ExamQuestionHandler{questionService: questionService}
}

func (h *ExamQuestionHandler) ListQuestionBanks(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	spaceID := c.Query("space_id")

	banks, err := h.questionService.ListQuestionBanks(ctx, tenantID, userID, spaceID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list question banks: %v", err)
		writeExamError(c, err, "Failed to list question banks")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": banks})
}

func (h *ExamQuestionHandler) CreateQuestionBank(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.CreateQuestionBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	bank, err := h.questionService.CreateQuestionBank(ctx, tenantID, userID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to create question bank: %v", err)
		writeExamError(c, err, "Failed to create question bank")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": bank})
}

func (h *ExamQuestionHandler) GetQuestionBank(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	bankID := c.Param("bank_id")

	bank, err := h.questionService.GetQuestionBank(ctx, tenantID, userID, bankID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get question bank: %v", err)
		writeExamError(c, err, "Failed to get question bank")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bank})
}

func (h *ExamQuestionHandler) ListQuestionDetails(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	bankID := c.Param("bank_id")

	items, err := h.questionService.ListQuestionDetails(ctx, tenantID, userID, bankID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list questions: %v", err)
		writeExamError(c, err, "Failed to list questions")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *ExamQuestionHandler) ListQuestionGroupDetails(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	bankID := c.Param("bank_id")

	groups, err := h.questionService.ListQuestionGroupDetails(ctx, tenantID, userID, bankID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list question groups: %v", err)
		writeExamError(c, err, "Failed to list question groups")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups})
}

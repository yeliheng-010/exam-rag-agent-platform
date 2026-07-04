package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamQuestionDraftHandler struct {
	service interfaces.ExamQuestionDraftService
}

func NewExamQuestionDraftHandler(service interfaces.ExamQuestionDraftService) *ExamQuestionDraftHandler {
	return &ExamQuestionDraftHandler{service: service}
}

func (h *ExamQuestionDraftHandler) ExtractDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.ExtractExamQuestionDraftsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.service.ExtractDrafts(ctx, tenantID, userID, c.Param("task_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract exam question drafts: %v", err)
		writeExamError(c, err, "Failed to extract exam question drafts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionDraftHandler) ListDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	result, err := h.service.ListDrafts(ctx, tenantID, userID, c.Param("task_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam question drafts: %v", err)
		writeExamError(c, err, "Failed to list exam question drafts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionDraftHandler) UpdateDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.UpdateExamQuestionDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.UpdateDraft(ctx, tenantID, userID, c.Param("draft_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to update exam question draft: %v", err)
		writeExamError(c, err, "Failed to update exam question draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ExamQuestionDraftHandler) ApproveDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	result, err := h.service.ApproveDraft(ctx, tenantID, userID, c.Param("draft_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to approve exam question draft: %v", err)
		writeExamError(c, err, "Failed to approve exam question draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionDraftHandler) RejectDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	draft, err := h.service.RejectDraft(ctx, tenantID, userID, c.Param("draft_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to reject exam question draft: %v", err)
		writeExamError(c, err, "Failed to reject exam question draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

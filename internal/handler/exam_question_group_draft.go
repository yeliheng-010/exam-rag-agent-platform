package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamQuestionGroupDraftHandler struct {
	service interfaces.ExamQuestionGroupDraftService
}

func NewExamQuestionGroupDraftHandler(service interfaces.ExamQuestionGroupDraftService) *ExamQuestionGroupDraftHandler {
	return &ExamQuestionGroupDraftHandler{service: service}
}

func (h *ExamQuestionGroupDraftHandler) ExtractDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.ExtractExamQuestionGroupDraftsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.service.StartExtraction(ctx, tenantID, userID, c.Param("task_id"), req.Force)
	if err != nil {
		logger.Errorf(ctx, "Failed to extract exam question group drafts: %v", err)
		writeExamError(c, err, "Failed to extract exam question group drafts")
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionGroupDraftHandler) ListDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	result, err := h.service.ListDrafts(ctx, tenantID, userID, c.Param("task_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam question group drafts: %v", err)
		writeExamError(c, err, "Failed to list exam question group drafts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionGroupDraftHandler) UpdateDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	var req types.UpdateExamQuestionGroupDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.UpdateDraft(ctx, tenantID, userID, c.Param("draft_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to update exam question group draft: %v", err)
		writeExamError(c, err, "Failed to update exam question group draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ExamQuestionGroupDraftHandler) ApproveDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	result, err := h.service.ApproveDraft(ctx, tenantID, userID, c.Param("draft_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to approve exam question group draft: %v", err)
		writeExamError(c, err, "Failed to approve exam question group draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamQuestionGroupDraftHandler) RejectDraft(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	draft, err := h.service.RejectDraft(ctx, tenantID, userID, c.Param("draft_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to reject exam question group draft: %v", err)
		writeExamError(c, err, "Failed to reject exam question group draft")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

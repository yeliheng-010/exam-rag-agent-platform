package handler

import (
	"net/http"
	"strconv"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func (h *ExamAssignmentHandler) SendAssignmentReminders(c *gin.Context) {
	ctx := c.Request.Context()
	var req types.SendExamAssignmentReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.assignmentService.SendAssignmentReminders(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("class_id"),
		c.Param("assignment_id"),
		&req,
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to send exam assignment reminders: %v", err)
		writeExamError(c, err, "Failed to send exam assignment reminders")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamAssignmentHandler) ListAssignmentNotifications(c *gin.Context) {
	ctx := c.Request.Context()
	limit, _ := strconv.Atoi(c.Query("limit"))
	result, err := h.assignmentService.ListAssignmentNotifications(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		limit,
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam assignment notifications: %v", err)
		writeExamError(c, err, "Failed to list exam assignment notifications")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamAssignmentHandler) MarkAssignmentNotificationRead(c *gin.Context) {
	ctx := c.Request.Context()
	err := h.assignmentService.MarkAssignmentNotificationRead(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("notification_id"),
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to mark exam assignment notification read: %v", err)
		writeExamError(c, err, "Failed to mark exam assignment notification read")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ExamAssignmentHandler) MarkAllAssignmentNotificationsRead(c *gin.Context) {
	ctx := c.Request.Context()
	err := h.assignmentService.MarkAllAssignmentNotificationsRead(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
	)
	if err != nil {
		logger.Errorf(ctx, "Failed to mark all exam assignment notifications read: %v", err)
		writeExamError(c, err, "Failed to mark all exam assignment notifications read")
		return
	}
	c.Status(http.StatusNoContent)
}

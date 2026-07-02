package handler

import (
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
)

type ExamTeacherApplicationHandler struct {
	service interfaces.ExamTeacherApplicationService
}

func NewExamTeacherApplicationHandler(service interfaces.ExamTeacherApplicationService) *ExamTeacherApplicationHandler {
	return &ExamTeacherApplicationHandler{service: service}
}

func (h *ExamTeacherApplicationHandler) GetMine(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	app, err := h.service.GetMine(ctx, tenantID, userID)
	if err != nil {
		writeExamError(c, err, "Failed to get teacher application")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": app})
}

func (h *ExamTeacherApplicationHandler) Apply(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.ApplyExamTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	app, err := h.service.Apply(ctx, tenantID, userID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to apply exam teacher: %v", err)
		writeExamError(c, err, "Failed to apply exam teacher")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": app})
}

func (h *ExamTeacherApplicationHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var status *types.ExamTeacherApplicationStatus
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		parsed := types.ExamTeacherApplicationStatus(raw)
		status = &parsed
	}
	apps, err := h.service.List(ctx, tenantID, status)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam teacher applications: %v", err)
		writeExamError(c, err, "Failed to list exam teacher applications")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": apps})
}

func (h *ExamTeacherApplicationHandler) Approve(c *gin.Context) {
	h.review(c, true)
}

func (h *ExamTeacherApplicationHandler) Reject(c *gin.Context) {
	h.review(c, false)
}

func (h *ExamTeacherApplicationHandler) review(c *gin.Context, approve bool) {
	ctx := c.Request.Context()
	reviewerID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	applicationID := c.Param("application_id")

	var req types.ReviewExamTeacherApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	var (
		app *types.ExamTeacherApplication
		err error
	)
	if approve {
		app, err = h.service.Approve(ctx, tenantID, reviewerID, applicationID, &req)
	} else {
		app, err = h.service.Reject(ctx, tenantID, reviewerID, applicationID, &req)
	}
	if err != nil {
		logger.Errorf(ctx, "Failed to review exam teacher application: %v", err)
		writeExamError(c, err, "Failed to review exam teacher application")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": app})
}

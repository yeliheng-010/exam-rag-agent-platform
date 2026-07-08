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

type ExamAssignmentHandler struct {
	assignmentService interfaces.ExamAssignmentService
}

func NewExamAssignmentHandler(assignmentService interfaces.ExamAssignmentService) *ExamAssignmentHandler {
	return &ExamAssignmentHandler{assignmentService: assignmentService}
}

func (h *ExamAssignmentHandler) CreateAssignment(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.CreateExamAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	item, err := h.assignmentService.CreateAssignment(ctx, tenantID, userID, c.Param("class_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to create exam assignment: %v", err)
		writeExamError(c, err, "Failed to create exam assignment")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": item})
}

func (h *ExamAssignmentHandler) ListClassAssignments(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	limit, _ := strconv.Atoi(c.Query("limit"))

	items, err := h.assignmentService.ListClassAssignments(ctx, tenantID, userID, c.Param("class_id"), types.ListExamAssignmentsFilter{Limit: limit})
	if err != nil {
		logger.Errorf(ctx, "Failed to list class assignments: %v", err)
		writeExamError(c, err, "Failed to list class assignments")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *ExamAssignmentHandler) ListMyAssignments(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	limit, _ := strconv.Atoi(c.Query("limit"))

	items, err := h.assignmentService.ListMyAssignments(ctx, tenantID, userID, types.ListExamAssignmentsFilter{Limit: limit})
	if err != nil {
		logger.Errorf(ctx, "Failed to list my exam assignments: %v", err)
		writeExamError(c, err, "Failed to list my exam assignments")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *ExamAssignmentHandler) CreateAssignmentAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	result, err := h.assignmentService.CreateAssignmentAttempt(ctx, tenantID, userID, c.Param("assignment_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to create assignment practice attempt: %v", err)
		writeExamError(c, err, "Failed to create assignment practice attempt")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
}

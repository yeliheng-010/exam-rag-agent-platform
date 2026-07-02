package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
)

type ExamClassHandler struct {
	classService interfaces.ExamClassService
}

func NewExamClassHandler(classService interfaces.ExamClassService) *ExamClassHandler {
	return &ExamClassHandler{classService: classService}
}

func (h *ExamClassHandler) ListClasses(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	classes, err := h.classService.ListClasses(ctx, tenantID, userID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam classes: %v", err)
		writeExamError(c, err, "Failed to list exam classes")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": classes})
}

func (h *ExamClassHandler) CreateClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.CreateExamClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	class, err := h.classService.CreateClass(ctx, tenantID, userID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to create exam class: %v", err)
		writeExamError(c, err, "Failed to create exam class")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": class})
}

func (h *ExamClassHandler) RequestJoinClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.JoinExamClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	member, err := h.classService.RequestJoinClass(ctx, tenantID, userID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to request joining exam class: %v", err)
		writeExamError(c, err, "Failed to request joining exam class")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": member})
}

func (h *ExamClassHandler) GetClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	classID := c.Param("class_id")

	class, err := h.classService.GetClass(ctx, tenantID, userID, classID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get exam class: %v", err)
		writeExamError(c, err, "Failed to get exam class")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": class})
}

func (h *ExamClassHandler) ListClassMembers(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	classID := c.Param("class_id")

	members, err := h.classService.ListClassMembers(ctx, tenantID, userID, classID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam class members: %v", err)
		writeExamError(c, err, "Failed to list exam class members")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": members})
}

func (h *ExamClassHandler) ApproveClassMember(c *gin.Context) {
	ctx := c.Request.Context()
	reviewerID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	classID := c.Param("class_id")
	targetUserID := c.Param("user_id")

	member, err := h.classService.ApproveClassMember(ctx, tenantID, reviewerID, classID, targetUserID)
	if err != nil {
		logger.Errorf(ctx, "Failed to approve exam class member: %v", err)
		writeExamError(c, err, "Failed to approve exam class member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": member})
}

func (h *ExamClassHandler) RejectClassMember(c *gin.Context) {
	ctx := c.Request.Context()
	reviewerID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	classID := c.Param("class_id")
	targetUserID := c.Param("user_id")

	member, err := h.classService.RejectClassMember(ctx, tenantID, reviewerID, classID, targetUserID)
	if err != nil {
		logger.Errorf(ctx, "Failed to reject exam class member: %v", err)
		writeExamError(c, err, "Failed to reject exam class member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": member})
}

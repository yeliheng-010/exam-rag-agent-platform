package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func (h *ExamClassHandler) UpdateClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	classID := c.Param("class_id")
	var req types.UpdateExamClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	class, err := h.classService.UpdateClass(ctx, tenantID, userID, classID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to update exam class: %v", err)
		writeExamError(c, err, "Failed to update exam class")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": class})
}

func (h *ExamClassHandler) ArchiveClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	class, err := h.classService.ArchiveClass(ctx, tenantID, userID, c.Param("class_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to archive exam class: %v", err)
		writeExamError(c, err, "Failed to archive exam class")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": class})
}

func (h *ExamClassHandler) RestoreClass(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	class, err := h.classService.RestoreClass(ctx, tenantID, userID, c.Param("class_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to restore exam class: %v", err)
		writeExamError(c, err, "Failed to restore exam class")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": class})
}

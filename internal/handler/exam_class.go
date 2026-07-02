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

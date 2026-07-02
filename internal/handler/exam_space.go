package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamSpaceHandler struct {
	spaceService interfaces.ExamSpaceService
}

func NewExamSpaceHandler(spaceService interfaces.ExamSpaceService) *ExamSpaceHandler {
	return &ExamSpaceHandler{spaceService: spaceService}
}

func (h *ExamSpaceHandler) ListSpaces(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	spaces, err := h.spaceService.ListSpaces(ctx, tenantID, userID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam spaces: %v", err)
		writeExamError(c, err, "Failed to list exam spaces")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": spaces})
}

func (h *ExamSpaceHandler) EnsurePersonalSpace(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	space, err := h.spaceService.EnsurePersonalSpace(ctx, tenantID, userID)
	if err != nil {
		logger.Errorf(ctx, "Failed to ensure personal exam space: %v", err)
		writeExamError(c, err, "Failed to ensure personal exam space")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": space})
}

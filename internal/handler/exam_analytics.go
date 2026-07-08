package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamAnalyticsHandler struct {
	analyticsService interfaces.ExamAnalyticsService
}

func NewExamAnalyticsHandler(analyticsService interfaces.ExamAnalyticsService) *ExamAnalyticsHandler {
	return &ExamAnalyticsHandler{analyticsService: analyticsService}
}

func (h *ExamAnalyticsHandler) GetClassAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	result, err := h.analyticsService.GetClassAnalytics(ctx, tenantID, userID, c.Param("class_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to get exam class analytics: %v", err)
		writeExamError(c, err, "Failed to get exam class analytics")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

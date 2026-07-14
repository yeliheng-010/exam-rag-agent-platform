package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamInterventionHandler struct {
	interventionService interfaces.ExamInterventionService
}

func NewExamInterventionHandler(
	interventionService interfaces.ExamInterventionService,
) *ExamInterventionHandler {
	return &ExamInterventionHandler{interventionService: interventionService}
}

func (h *ExamInterventionHandler) GetClassRecommendations(c *gin.Context) {
	req, ok := bindExamRecommendationQuery(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	result, err := h.interventionService.RecommendClassPractice(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		c.Param("class_id"),
		req,
	)
	writeExamRecommendationResponse(c, result, err)
}

func (h *ExamInterventionHandler) GetStudentRecommendations(c *gin.Context) {
	req, ok := bindExamRecommendationQuery(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	result, err := h.interventionService.RecommendStudentPractice(
		ctx,
		c.GetUint64(types.TenantIDContextKey.String()),
		c.GetString(types.UserIDContextKey.String()),
		req,
	)
	writeExamRecommendationResponse(c, result, err)
}

func bindExamRecommendationQuery(c *gin.Context) (types.ExamPracticeRecommendationRequest, bool) {
	var req types.ExamPracticeRecommendationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		writeExamError(c, service.ErrExamInvalidRequest, "Invalid recommendation query")
		return req, false
	}
	return req, true
}

func writeExamRecommendationResponse(
	c *gin.Context,
	result *types.ExamPracticeRecommendationResult,
	err error,
) {
	if err != nil {
		logger.Errorf(c.Request.Context(), "Failed to get exam practice recommendations: %v", err)
		writeExamError(c, err, "Failed to get exam practice recommendations")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamResourceHandler struct {
	resourceService interfaces.ExamResourceService
}

func NewExamResourceHandler(resourceService interfaces.ExamResourceService) *ExamResourceHandler {
	return &ExamResourceHandler{resourceService: resourceService}
}

func (h *ExamResourceHandler) BindKnowledgeBase(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	kbID := c.Param("kb_id")

	var req types.BindKnowledgeBaseResourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	resource, err := h.resourceService.BindKnowledgeBase(ctx, tenantID, userID, kbID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to bind knowledge base to exam space: %v", err)
		writeExamError(c, err, "Failed to bind knowledge base")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resource})
}

func (h *ExamResourceHandler) GetKnowledgeBaseBinding(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	kbID := c.Param("kb_id")

	resource, err := h.resourceService.GetKnowledgeBaseBinding(ctx, tenantID, userID, kbID)
	if err != nil {
		logger.Errorf(ctx, "Failed to get knowledge base exam binding: %v", err)
		writeExamError(c, err, "Failed to get knowledge base exam binding")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resource})
}

func (h *ExamResourceHandler) ListResources(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	filter := types.ListExamResourcesFilter{
		ResourceType: types.ExamResourceType(c.Query("resource_type")),
		SpaceID:      c.Query("space_id"),
		DomainID:     c.Query("domain_id"),
		SubjectID:    c.Query("subject_id"),
		MaterialType: types.ExamMaterialType(c.Query("material_type")),
	}
	resources, err := h.resourceService.ListResources(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam resources: %v", err)
		writeExamError(c, err, "Failed to list exam resources")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": resources})
}

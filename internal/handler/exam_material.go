package handler

import (
	"net/http"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamMaterialHandler struct {
	materialService interfaces.ExamMaterialService
}

func NewExamMaterialHandler(materialService interfaces.ExamMaterialService) *ExamMaterialHandler {
	return &ExamMaterialHandler{materialService: materialService}
}

func (h *ExamMaterialHandler) RegisterMaterial(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.RegisterExamMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.materialService.RegisterMaterial(ctx, tenantID, userID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to register exam material: %v", err)
		writeExamError(c, err, "Failed to register exam material")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
}

func (h *ExamMaterialHandler) ListMaterials(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	filter := types.ListExamMaterialsFilter{
		SpaceID:      c.Query("space_id"),
		DomainID:     c.Query("domain_id"),
		SubjectID:    c.Query("subject_id"),
		MaterialType: types.ExamMaterialType(c.Query("material_type")),
	}
	materials, err := h.materialService.ListMaterials(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam materials: %v", err)
		writeExamError(c, err, "Failed to list exam materials")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": materials})
}

func (h *ExamMaterialHandler) CreateStructuringTask(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	materialID := c.Param("material_id")

	var req types.CreateExamStructuringTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	task, err := h.materialService.CreateStructuringTask(ctx, tenantID, userID, materialID, &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to create exam structuring task: %v", err)
		writeExamError(c, err, "Failed to create exam structuring task")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": task})
}

func (h *ExamMaterialHandler) ListStructuringTasks(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	filter := types.ListExamStructuringTasksFilter{
		SpaceID:    c.Query("space_id"),
		MaterialID: c.Query("material_id"),
		Status:     types.ExamStructuringTaskStatus(c.Query("status")),
	}
	tasks, err := h.materialService.ListStructuringTasks(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam structuring tasks: %v", err)
		writeExamError(c, err, "Failed to list exam structuring tasks")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tasks})
}

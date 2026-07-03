package handler

import (
	stderrors "errors"
	"net/http"
	"strconv"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type BillingHandler struct {
	service interfaces.BillingService
}

func NewBillingHandler(svc interfaces.BillingService) *BillingHandler {
	return &BillingHandler{service: svc}
}

func (h *BillingHandler) GetMine(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("tenant ID not found"))
		return
	}
	status, err := h.service.GetTenantBilling(ctx, tenantID)
	if err != nil {
		writeBillingError(c, err, "Failed to get billing status")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": status})
}

func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.service.ListPlans(c.Request.Context(), c.Query("include_archived") == "true")
	if err != nil {
		writeBillingError(c, err, "Failed to list billing plans")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

func (h *BillingHandler) CreatePlan(c *gin.Context) {
	ctx := c.Request.Context()
	actorID := c.GetString(types.UserIDContextKey.String())
	var req types.CreateBillingPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	plan, err := h.service.CreatePlan(ctx, actorID, &req)
	if err != nil {
		writeBillingError(c, err, "Failed to create billing plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plan})
}

func (h *BillingHandler) ListSubscriptions(c *gin.Context) {
	ctx := c.Request.Context()
	page, pageSize := parseBillingPage(c)
	filter := types.ListTenantSubscriptionsFilter{
		TenantID: parseUintQuery(c, "tenant_id"),
		Status:   types.SubscriptionStatus(c.Query("status")),
	}
	subs, total, err := h.service.ListTenantSubscriptions(ctx, filter, page, pageSize)
	if err != nil {
		writeBillingError(c, err, "Failed to list tenant subscriptions")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      subs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *BillingHandler) UpsertTenantSubscription(c *gin.Context) {
	ctx := c.Request.Context()
	actorID := c.GetString(types.UserIDContextKey.String())
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 64)
	if err != nil || tenantID == 0 {
		c.Error(apperrors.NewBadRequestError("invalid tenant_id"))
		return
	}
	var req types.UpsertTenantSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	sub, err := h.service.UpsertTenantSubscription(ctx, actorID, tenantID, &req)
	if err != nil {
		writeBillingError(c, err, "Failed to update tenant subscription")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sub})
}

func (h *BillingHandler) ListOrders(c *gin.Context) {
	ctx := c.Request.Context()
	page, pageSize := parseBillingPage(c)
	filter := types.ListBillingOrdersFilter{
		TenantID: parseUintQuery(c, "tenant_id"),
		Status:   types.BillingOrderStatus(c.Query("status")),
	}
	orders, total, err := h.service.ListOrders(ctx, filter, page, pageSize)
	if err != nil {
		writeBillingError(c, err, "Failed to list billing orders")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      orders,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *BillingHandler) CreateOrder(c *gin.Context) {
	ctx := c.Request.Context()
	actorID := c.GetString(types.UserIDContextKey.String())
	var req types.CreateBillingOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	order, err := h.service.CreateOrder(ctx, actorID, &req)
	if err != nil {
		writeBillingError(c, err, "Failed to create billing order")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": order})
}

func writeBillingError(c *gin.Context, err error, fallback string) {
	switch {
	case stderrors.Is(err, service.ErrBillingInvalidRequest):
		c.Error(apperrors.NewValidationError("Invalid request parameters"))
	case stderrors.Is(err, service.ErrBillingPlanNotFound):
		c.Error(apperrors.NewNotFoundError("Billing plan not found"))
	case stderrors.Is(err, service.ErrBillingPlanAlreadyExists):
		c.Error(apperrors.NewConflictError("Billing plan already exists"))
	default:
		logger.Errorf(c.Request.Context(), "%s: %v", fallback, err)
		c.Error(apperrors.NewInternalServerError(fallback).WithDetails(err.Error()))
	}
}

func parseBillingPage(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func parseUintQuery(c *gin.Context, key string) uint64 {
	raw := c.Query(key)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

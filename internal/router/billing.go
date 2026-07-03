package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/gin-gonic/gin"
)

func RegisterBillingRoutes(r *gin.RouterGroup, billingHandler *handler.BillingHandler, g *rbacGuards) {
	if billingHandler == nil {
		return
	}

	r.GET("/billing/me", g.Viewer(), billingHandler.GetMine)

	admin := r.Group("/system/admin/billing", g.SystemAdmin())
	{
		admin.GET("/plans", billingHandler.ListPlans)
		admin.POST("/plans", billingHandler.CreatePlan)
		admin.GET("/subscriptions", billingHandler.ListSubscriptions)
		admin.PUT("/subscriptions/:tenant_id", billingHandler.UpsertTenantSubscription)
		admin.GET("/orders", billingHandler.ListOrders)
		admin.POST("/orders", billingHandler.CreateOrder)
	}
}

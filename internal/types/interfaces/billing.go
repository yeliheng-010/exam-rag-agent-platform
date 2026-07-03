package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type BillingRepository interface {
	CreatePlan(ctx context.Context, plan *types.BillingPlan) error
	GetPlanByID(ctx context.Context, id string) (*types.BillingPlan, error)
	GetPlanByCode(ctx context.Context, code string) (*types.BillingPlan, error)
	ListPlans(ctx context.Context, includeArchived bool) ([]*types.BillingPlan, error)
	UpsertSubscription(ctx context.Context, subscription *types.TenantSubscription) error
	GetSubscriptionByTenant(ctx context.Context, tenantID uint64) (*types.TenantSubscription, error)
	ListSubscriptions(ctx context.Context, filter types.ListTenantSubscriptionsFilter, page, pageSize int) ([]*types.TenantSubscription, int64, error)
	CreateOrder(ctx context.Context, order *types.BillingOrder) error
	ListOrders(ctx context.Context, filter types.ListBillingOrdersFilter, page, pageSize int) ([]*types.BillingOrder, int64, error)
}

type BillingService interface {
	GetTenantBilling(ctx context.Context, tenantID uint64) (*types.TenantBillingStatus, error)
	CreatePlan(ctx context.Context, actorUserID string, req *types.CreateBillingPlanRequest) (*types.BillingPlan, error)
	ListPlans(ctx context.Context, includeArchived bool) ([]*types.BillingPlan, error)
	UpsertTenantSubscription(ctx context.Context, actorUserID string, tenantID uint64, req *types.UpsertTenantSubscriptionRequest) (*types.TenantSubscription, error)
	ListTenantSubscriptions(ctx context.Context, filter types.ListTenantSubscriptionsFilter, page, pageSize int) ([]*types.TenantSubscription, int64, error)
	CreateOrder(ctx context.Context, actorUserID string, req *types.CreateBillingOrderRequest) (*types.BillingOrder, error)
	ListOrders(ctx context.Context, filter types.ListBillingOrdersFilter, page, pageSize int) ([]*types.BillingOrder, int64, error)
}

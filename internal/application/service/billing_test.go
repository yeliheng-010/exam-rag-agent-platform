package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
)

type fakeBillingRepo struct {
	plans         map[string]*types.BillingPlan
	plansByCode   map[string]string
	subscriptions map[uint64]*types.TenantSubscription
	orders        map[string]*types.BillingOrder
}

func newFakeBillingRepo() *fakeBillingRepo {
	return &fakeBillingRepo{
		plans:         map[string]*types.BillingPlan{},
		plansByCode:   map[string]string{},
		subscriptions: map[uint64]*types.TenantSubscription{},
		orders:        map[string]*types.BillingOrder{},
	}
}

func cloneBillingPlan(plan *types.BillingPlan) *types.BillingPlan {
	if plan == nil {
		return nil
	}
	cp := *plan
	if plan.Entitlements != nil {
		cp.Entitlements = types.JSONMap{}
		for k, v := range plan.Entitlements {
			cp.Entitlements[k] = v
		}
	}
	return &cp
}

func cloneTenantSubscription(sub *types.TenantSubscription) *types.TenantSubscription {
	if sub == nil {
		return nil
	}
	cp := *sub
	cp.Plan = cloneBillingPlan(sub.Plan)
	return &cp
}

func cloneBillingOrder(order *types.BillingOrder) *types.BillingOrder {
	if order == nil {
		return nil
	}
	cp := *order
	if order.Entitlements != nil {
		cp.Entitlements = types.JSONMap{}
		for k, v := range order.Entitlements {
			cp.Entitlements[k] = v
		}
	}
	cp.Plan = cloneBillingPlan(order.Plan)
	return &cp
}

func (r *fakeBillingRepo) CreatePlan(_ context.Context, plan *types.BillingPlan) error {
	if _, exists := r.plansByCode[plan.Code]; exists {
		return repository.ErrBillingPlanAlreadyExists
	}
	r.plans[plan.ID] = cloneBillingPlan(plan)
	r.plansByCode[plan.Code] = plan.ID
	return nil
}

func (r *fakeBillingRepo) GetPlanByID(_ context.Context, id string) (*types.BillingPlan, error) {
	plan := r.plans[id]
	if plan == nil {
		return nil, repository.ErrBillingPlanNotFound
	}
	return cloneBillingPlan(plan), nil
}

func (r *fakeBillingRepo) GetPlanByCode(_ context.Context, code string) (*types.BillingPlan, error) {
	id := r.plansByCode[code]
	if id == "" {
		return nil, repository.ErrBillingPlanNotFound
	}
	return cloneBillingPlan(r.plans[id]), nil
}

func (r *fakeBillingRepo) ListPlans(context.Context, bool) ([]*types.BillingPlan, error) {
	out := make([]*types.BillingPlan, 0, len(r.plans))
	for _, plan := range r.plans {
		out = append(out, cloneBillingPlan(plan))
	}
	return out, nil
}

func (r *fakeBillingRepo) UpsertSubscription(_ context.Context, sub *types.TenantSubscription) error {
	cp := cloneTenantSubscription(sub)
	if cp.Plan == nil {
		cp.Plan = cloneBillingPlan(r.plans[cp.PlanID])
	}
	r.subscriptions[sub.TenantID] = cp
	return nil
}

func (r *fakeBillingRepo) GetSubscriptionByTenant(_ context.Context, tenantID uint64) (*types.TenantSubscription, error) {
	sub := r.subscriptions[tenantID]
	if sub == nil {
		return nil, repository.ErrTenantSubscriptionNotFound
	}
	cp := cloneTenantSubscription(sub)
	cp.Plan = cloneBillingPlan(r.plans[cp.PlanID])
	return cp, nil
}

func (r *fakeBillingRepo) ListSubscriptions(context.Context, types.ListTenantSubscriptionsFilter, int, int) ([]*types.TenantSubscription, int64, error) {
	out := make([]*types.TenantSubscription, 0, len(r.subscriptions))
	for _, sub := range r.subscriptions {
		out = append(out, cloneTenantSubscription(sub))
	}
	return out, int64(len(out)), nil
}

func (r *fakeBillingRepo) CreateOrder(_ context.Context, order *types.BillingOrder) error {
	r.orders[order.ID] = cloneBillingOrder(order)
	return nil
}

func (r *fakeBillingRepo) ListOrders(context.Context, types.ListBillingOrdersFilter, int, int) ([]*types.BillingOrder, int64, error) {
	out := make([]*types.BillingOrder, 0, len(r.orders))
	for _, order := range r.orders {
		out = append(out, cloneBillingOrder(order))
	}
	return out, int64(len(out)), nil
}

func TestBillingGetTenantBillingFallsBackToFreePlan(t *testing.T) {
	repo := newFakeBillingRepo()
	svc := NewBillingService(repo).(*billingService)

	status, err := svc.GetTenantBilling(context.Background(), 1001)
	if err != nil {
		t.Fatalf("GetTenantBilling returned error: %v", err)
	}
	if !status.IsFree {
		t.Fatalf("status.IsFree = false, want true")
	}
	if status.Subscription != nil {
		t.Fatalf("default billing status should not fabricate subscription row")
	}
	if status.Plan == nil || status.Plan.Code != "free" {
		t.Fatalf("default plan = %+v, want code=free", status.Plan)
	}
	if got := status.Plan.Entitlements["class_member_limit"]; got != float64(50) {
		t.Fatalf("free class_member_limit = %#v, want 50", got)
	}
}

func TestBillingCreatePlanNormalizesDefaults(t *testing.T) {
	repo := newFakeBillingRepo()
	svc := NewBillingService(repo).(*billingService)

	plan, err := svc.CreatePlan(context.Background(), "admin-1", &types.CreateBillingPlanRequest{
		Code:        " Pro ",
		Name:        "专业版",
		AmountCents: 19900,
		Entitlements: types.JSONMap{
			"class_member_limit": 200,
		},
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}
	if plan.Code != "pro" {
		t.Fatalf("plan code = %q, want pro", plan.Code)
	}
	if plan.Currency != "CNY" {
		t.Fatalf("currency = %q, want CNY", plan.Currency)
	}
	if plan.Interval != types.BillingIntervalMonth {
		t.Fatalf("interval = %q, want month", plan.Interval)
	}
	if plan.Status != types.BillingPlanStatusActive {
		t.Fatalf("status = %q, want active", plan.Status)
	}
	if plan.CreatedByUserID != "admin-1" {
		t.Fatalf("created_by = %q, want admin-1", plan.CreatedByUserID)
	}
}

func TestBillingCreateOrderSnapshotsPlanPrice(t *testing.T) {
	repo := newFakeBillingRepo()
	svc := NewBillingService(repo).(*billingService)
	plan, err := svc.CreatePlan(context.Background(), "admin-1", &types.CreateBillingPlanRequest{
		Code:        "ielts-pro",
		Name:        "雅思专业版",
		Currency:    "cny",
		AmountCents: 29900,
		Interval:    types.BillingIntervalYear,
		Entitlements: types.JSONMap{
			"agent_calls_monthly": 5000,
		},
	})
	if err != nil {
		t.Fatalf("CreatePlan returned error: %v", err)
	}

	order, err := svc.CreateOrder(context.Background(), "admin-1", &types.CreateBillingOrderRequest{
		TenantID: 1001,
		PlanID:   plan.ID,
		Provider: "manual",
		Status:   types.BillingOrderStatusPaid,
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if order.AmountCents != 29900 || order.Currency != "CNY" {
		t.Fatalf("order price snapshot = %d %s, want 29900 CNY", order.AmountCents, order.Currency)
	}
	if got := order.Entitlements["agent_calls_monthly"]; got != 5000 {
		t.Fatalf("order entitlements snapshot = %#v, want 5000", got)
	}

	repo.plans[plan.ID].AmountCents = 9900
	if order.AmountCents != 29900 {
		t.Fatalf("mutating plan changed order snapshot to %d", order.AmountCents)
	}
}

func TestBillingUpsertSubscriptionRejectsMissingPlan(t *testing.T) {
	repo := newFakeBillingRepo()
	svc := NewBillingService(repo).(*billingService)

	_, err := svc.UpsertTenantSubscription(context.Background(), "admin-1", 1001, &types.UpsertTenantSubscriptionRequest{
		PlanID: "missing",
	})
	if !errors.Is(err, ErrBillingPlanNotFound) {
		t.Fatalf("UpsertTenantSubscription error = %v, want ErrBillingPlanNotFound", err)
	}
}

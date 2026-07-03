package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

var (
	ErrBillingInvalidRequest     = errors.New("invalid billing request")
	ErrBillingPlanNotFound       = repository.ErrBillingPlanNotFound
	ErrBillingPlanAlreadyExists  = repository.ErrBillingPlanAlreadyExists
	ErrTenantSubscriptionMissing = repository.ErrTenantSubscriptionNotFound
)

type billingService struct {
	repo interfaces.BillingRepository
}

func NewBillingService(repo interfaces.BillingRepository) interfaces.BillingService {
	return &billingService{repo: repo}
}

func (s *billingService) GetTenantBilling(ctx context.Context, tenantID uint64) (*types.TenantBillingStatus, error) {
	if tenantID == 0 {
		return nil, ErrBillingInvalidRequest
	}
	sub, err := s.repo.GetSubscriptionByTenant(ctx, tenantID)
	if err == nil {
		plan := sub.Plan
		if plan == nil || plan.ID == "" {
			plan, err = s.repo.GetPlanByID(ctx, sub.PlanID)
			if err != nil {
				return nil, mapPlanErr(err)
			}
		}
		return &types.TenantBillingStatus{
			TenantID:     tenantID,
			Plan:         copyBillingPlan(plan),
			Subscription: sub,
			IsFree:       plan != nil && plan.Code == "free",
		}, nil
	}
	if !errors.Is(err, repository.ErrTenantSubscriptionNotFound) {
		return nil, err
	}

	freePlan, err := s.repo.GetPlanByCode(ctx, "free")
	if err != nil {
		if !errors.Is(err, repository.ErrBillingPlanNotFound) {
			return nil, err
		}
		freePlan = defaultFreeBillingPlan()
	}
	return &types.TenantBillingStatus{
		TenantID: tenantID,
		Plan:     copyBillingPlan(freePlan),
		IsFree:   true,
	}, nil
}

func (s *billingService) CreatePlan(ctx context.Context, actorUserID string, req *types.CreateBillingPlanRequest) (*types.BillingPlan, error) {
	if req == nil {
		return nil, ErrBillingInvalidRequest
	}
	code := normalizePlanCode(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" || req.AmountCents < 0 {
		return nil, ErrBillingInvalidRequest
	}
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "CNY"
	}
	interval := req.Interval
	if interval == "" {
		interval = types.BillingIntervalMonth
	}
	if !isBillingIntervalValid(interval) {
		return nil, ErrBillingInvalidRequest
	}
	now := time.Now()
	plan := &types.BillingPlan{
		ID:              uuid.New().String(),
		Code:            code,
		Name:            name,
		Description:     strings.TrimSpace(req.Description),
		Currency:        currency,
		AmountCents:     req.AmountCents,
		Interval:        interval,
		Entitlements:    copyJSONMap(req.Entitlements),
		Status:          types.BillingPlanStatusActive,
		CreatedByUserID: strings.TrimSpace(actorUserID),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if plan.Entitlements == nil {
		plan.Entitlements = types.JSONMap{}
	}
	if err := s.repo.CreatePlan(ctx, plan); err != nil {
		if errors.Is(err, repository.ErrBillingPlanAlreadyExists) {
			return nil, ErrBillingPlanAlreadyExists
		}
		return nil, err
	}
	return copyBillingPlan(plan), nil
}

func (s *billingService) ListPlans(ctx context.Context, includeArchived bool) ([]*types.BillingPlan, error) {
	return s.repo.ListPlans(ctx, includeArchived)
}

func (s *billingService) UpsertTenantSubscription(
	ctx context.Context,
	actorUserID string,
	tenantID uint64,
	req *types.UpsertTenantSubscriptionRequest,
) (*types.TenantSubscription, error) {
	if tenantID == 0 || req == nil || strings.TrimSpace(req.PlanID) == "" {
		return nil, ErrBillingInvalidRequest
	}
	plan, err := s.repo.GetPlanByID(ctx, strings.TrimSpace(req.PlanID))
	if err != nil {
		return nil, mapPlanErr(err)
	}
	status := req.Status
	if status == "" {
		status = types.SubscriptionStatusActive
	}
	if !isSubscriptionStatusValid(status) {
		return nil, ErrBillingInvalidRequest
	}
	now := time.Now()
	sub := &types.TenantSubscription{
		ID:                 uuid.New().String(),
		TenantID:           tenantID,
		PlanID:             plan.ID,
		Status:             status,
		CurrentPeriodStart: req.CurrentPeriodStart,
		CurrentPeriodEnd:   req.CurrentPeriodEnd,
		SourceOrderID:      req.SourceOrderID,
		CreatedByUserID:    strings.TrimSpace(actorUserID),
		CreatedAt:          now,
		UpdatedAt:          now,
		Plan:               copyBillingPlan(plan),
	}
	if err := s.repo.UpsertSubscription(ctx, sub); err != nil {
		return nil, err
	}
	return s.repo.GetSubscriptionByTenant(ctx, tenantID)
}

func (s *billingService) ListTenantSubscriptions(
	ctx context.Context,
	filter types.ListTenantSubscriptionsFilter,
	page int,
	pageSize int,
) ([]*types.TenantSubscription, int64, error) {
	return s.repo.ListSubscriptions(ctx, filter, page, pageSize)
}

func (s *billingService) CreateOrder(ctx context.Context, actorUserID string, req *types.CreateBillingOrderRequest) (*types.BillingOrder, error) {
	if req == nil || req.TenantID == 0 || strings.TrimSpace(req.PlanID) == "" {
		return nil, ErrBillingInvalidRequest
	}
	plan, err := s.repo.GetPlanByID(ctx, strings.TrimSpace(req.PlanID))
	if err != nil {
		return nil, mapPlanErr(err)
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "manual"
	}
	status := req.Status
	if status == "" {
		status = types.BillingOrderStatusPending
	}
	if !isBillingOrderStatusValid(status) {
		return nil, ErrBillingInvalidRequest
	}
	paidAt := req.PaidAt
	if status == types.BillingOrderStatusPaid && paidAt == nil {
		nowPaid := time.Now()
		paidAt = &nowPaid
	}
	now := time.Now()
	order := &types.BillingOrder{
		ID:              uuid.New().String(),
		TenantID:        req.TenantID,
		PlanID:          plan.ID,
		Provider:        provider,
		ProviderOrderID: strings.TrimSpace(req.ProviderOrderID),
		Currency:        plan.Currency,
		AmountCents:     plan.AmountCents,
		Status:          status,
		Entitlements:    copyJSONMap(plan.Entitlements),
		CreatedByUserID: strings.TrimSpace(actorUserID),
		PaidAt:          paidAt,
		CreatedAt:       now,
		UpdatedAt:       now,
		Plan:            copyBillingPlan(plan),
	}
	if order.Entitlements == nil {
		order.Entitlements = types.JSONMap{}
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	return copyBillingOrder(order), nil
}

func (s *billingService) ListOrders(
	ctx context.Context,
	filter types.ListBillingOrdersFilter,
	page int,
	pageSize int,
) ([]*types.BillingOrder, int64, error) {
	return s.repo.ListOrders(ctx, filter, page, pageSize)
}

func mapPlanErr(err error) error {
	if errors.Is(err, repository.ErrBillingPlanNotFound) {
		return ErrBillingPlanNotFound
	}
	return err
}

func normalizePlanCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func defaultFreeBillingPlan() *types.BillingPlan {
	return &types.BillingPlan{
		ID:          "free",
		Code:        "free",
		Name:        "免费版",
		Description: "默认免费权益",
		Currency:    "CNY",
		Interval:    types.BillingIntervalMonth,
		Status:      types.BillingPlanStatusActive,
		Entitlements: types.JSONMap{
			"class_count_limit":           float64(1),
			"class_member_limit":          float64(50),
			"document_parse_monthly":      float64(100),
			"exam_structuring_monthly":    float64(20),
			"agent_calls_monthly":         float64(1000),
			"question_generation_monthly": float64(100),
		},
	}
}

func copyBillingPlan(plan *types.BillingPlan) *types.BillingPlan {
	if plan == nil {
		return nil
	}
	cp := *plan
	cp.Entitlements = copyJSONMap(plan.Entitlements)
	return &cp
}

func copyBillingOrder(order *types.BillingOrder) *types.BillingOrder {
	if order == nil {
		return nil
	}
	cp := *order
	cp.Entitlements = copyJSONMap(order.Entitlements)
	cp.Plan = copyBillingPlan(order.Plan)
	return &cp
}

func copyJSONMap(in types.JSONMap) types.JSONMap {
	if in == nil {
		return nil
	}
	out := make(types.JSONMap, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isBillingIntervalValid(interval types.BillingInterval) bool {
	switch interval {
	case types.BillingIntervalMonth, types.BillingIntervalYear, types.BillingIntervalOneTime:
		return true
	default:
		return false
	}
}

func isSubscriptionStatusValid(status types.SubscriptionStatus) bool {
	switch status {
	case types.SubscriptionStatusActive,
		types.SubscriptionStatusPastDue,
		types.SubscriptionStatusCanceled,
		types.SubscriptionStatusExpired:
		return true
	default:
		return false
	}
}

func isBillingOrderStatusValid(status types.BillingOrderStatus) bool {
	switch status {
	case types.BillingOrderStatusPending,
		types.BillingOrderStatusPaid,
		types.BillingOrderStatusCanceled,
		types.BillingOrderStatusFailed:
		return true
	default:
		return false
	}
}

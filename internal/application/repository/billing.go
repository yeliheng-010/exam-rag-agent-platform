package repository

import (
	"context"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBillingPlanNotFound        = errors.New("billing plan not found")
	ErrBillingPlanAlreadyExists   = errors.New("billing plan already exists")
	ErrTenantSubscriptionNotFound = errors.New("tenant subscription not found")
)

type billingRepository struct {
	db *gorm.DB
}

func NewBillingRepository(db *gorm.DB) interfaces.BillingRepository {
	return &billingRepository{db: db}
}

func (r *billingRepository) CreatePlan(ctx context.Context, plan *types.BillingPlan) error {
	if existing, err := r.GetPlanByCode(ctx, plan.Code); err == nil && existing != nil {
		return ErrBillingPlanAlreadyExists
	} else if err != nil && !errors.Is(err, ErrBillingPlanNotFound) {
		return err
	}
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *billingRepository) GetPlanByID(ctx context.Context, id string) (*types.BillingPlan, error) {
	var plan types.BillingPlan
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBillingPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *billingRepository) GetPlanByCode(ctx context.Context, code string) (*types.BillingPlan, error) {
	var plan types.BillingPlan
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBillingPlanNotFound
		}
		return nil, err
	}
	return &plan, nil
}

func (r *billingRepository) ListPlans(ctx context.Context, includeArchived bool) ([]*types.BillingPlan, error) {
	query := r.db.WithContext(ctx).Model(&types.BillingPlan{})
	if !includeArchived {
		query = query.Where("status = ?", types.BillingPlanStatusActive)
	}
	var plans []*types.BillingPlan
	err := query.Order("amount_cents ASC, created_at ASC").Find(&plans).Error
	return plans, err
}

func (r *billingRepository) UpsertSubscription(ctx context.Context, subscription *types.TenantSubscription) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"plan_id",
			"status",
			"current_period_start",
			"current_period_end",
			"source_order_id",
			"created_by_user_id",
			"updated_at",
		}),
	}).Create(subscription).Error
}

func (r *billingRepository) GetSubscriptionByTenant(ctx context.Context, tenantID uint64) (*types.TenantSubscription, error) {
	var sub types.TenantSubscription
	err := r.db.WithContext(ctx).
		Preload("Plan").
		Where("tenant_id = ?", tenantID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantSubscriptionNotFound
		}
		return nil, err
	}
	return &sub, nil
}

func (r *billingRepository) ListSubscriptions(
	ctx context.Context,
	filter types.ListTenantSubscriptionsFilter,
	page int,
	pageSize int,
) ([]*types.TenantSubscription, int64, error) {
	query := r.db.WithContext(ctx).Model(&types.TenantSubscription{})
	if filter.TenantID > 0 {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageOffsetLimit(page, pageSize)
	var subs []*types.TenantSubscription
	err := query.Preload("Plan").
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&subs).Error
	return subs, total, err
}

func (r *billingRepository) CreateOrder(ctx context.Context, order *types.BillingOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *billingRepository) ListOrders(
	ctx context.Context,
	filter types.ListBillingOrdersFilter,
	page int,
	pageSize int,
) ([]*types.BillingOrder, int64, error) {
	query := r.db.WithContext(ctx).Model(&types.BillingOrder{})
	if filter.TenantID > 0 {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset, limit := pageOffsetLimit(page, pageSize)
	var orders []*types.BillingOrder
	err := query.Preload("Plan").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&orders).Error
	return orders, total, err
}

func pageOffsetLimit(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return (page - 1) * pageSize, pageSize
}

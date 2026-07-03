package types

import "time"

type BillingPlanStatus string

const (
	BillingPlanStatusActive   BillingPlanStatus = "active"
	BillingPlanStatusArchived BillingPlanStatus = "archived"
)

type BillingInterval string

const (
	BillingIntervalMonth   BillingInterval = "month"
	BillingIntervalYear    BillingInterval = "year"
	BillingIntervalOneTime BillingInterval = "one_time"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
)

type BillingOrderStatus string

const (
	BillingOrderStatusPending  BillingOrderStatus = "pending"
	BillingOrderStatusPaid     BillingOrderStatus = "paid"
	BillingOrderStatusCanceled BillingOrderStatus = "canceled"
	BillingOrderStatusFailed   BillingOrderStatus = "failed"
)

type BillingPlan struct {
	ID              string            `json:"id" gorm:"type:varchar(36);primaryKey"`
	Code            string            `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Name            string            `json:"name" gorm:"type:varchar(128);not null"`
	Description     string            `json:"description" gorm:"type:text;not null;default:''"`
	Currency        string            `json:"currency" gorm:"type:varchar(16);not null;default:'CNY'"`
	AmountCents     int64             `json:"amount_cents" gorm:"not null;default:0"`
	Interval        BillingInterval   `json:"interval" gorm:"type:varchar(32);not null;default:'month'"`
	Entitlements    JSONMap           `json:"entitlements" gorm:"type:jsonb"`
	Status          BillingPlanStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedByUserID string            `json:"created_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func (BillingPlan) TableName() string {
	return "billing_plans"
}

type TenantSubscription struct {
	ID                 string             `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64             `json:"tenant_id" gorm:"not null;uniqueIndex"`
	PlanID             string             `json:"plan_id" gorm:"type:varchar(36);not null;index"`
	Status             SubscriptionStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CurrentPeriodStart *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time         `json:"current_period_end,omitempty"`
	SourceOrderID      *string            `json:"source_order_id,omitempty" gorm:"type:varchar(36);index"`
	CreatedByUserID    string             `json:"created_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`

	Plan *BillingPlan `json:"plan,omitempty" gorm:"foreignKey:PlanID;references:ID"`
}

func (TenantSubscription) TableName() string {
	return "tenant_subscriptions"
}

type BillingOrder struct {
	ID              string             `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID        uint64             `json:"tenant_id" gorm:"not null;index"`
	PlanID          string             `json:"plan_id" gorm:"type:varchar(36);not null;index"`
	Provider        string             `json:"provider" gorm:"type:varchar(32);not null;default:'manual'"`
	ProviderOrderID string             `json:"provider_order_id" gorm:"type:varchar(128);not null;default:'';index"`
	Currency        string             `json:"currency" gorm:"type:varchar(16);not null;default:'CNY'"`
	AmountCents     int64              `json:"amount_cents" gorm:"not null;default:0"`
	Status          BillingOrderStatus `json:"status" gorm:"type:varchar(32);not null;default:'pending'"`
	Entitlements    JSONMap            `json:"entitlements" gorm:"type:jsonb"`
	CreatedByUserID string             `json:"created_by_user_id" gorm:"type:varchar(36);not null;default:''"`
	PaidAt          *time.Time         `json:"paid_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`

	Plan *BillingPlan `json:"plan,omitempty" gorm:"foreignKey:PlanID;references:ID"`
}

func (BillingOrder) TableName() string {
	return "billing_orders"
}

type CreateBillingPlanRequest struct {
	Code         string          `json:"code" binding:"required"`
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	Currency     string          `json:"currency"`
	AmountCents  int64           `json:"amount_cents"`
	Interval     BillingInterval `json:"interval"`
	Entitlements JSONMap         `json:"entitlements"`
}

type UpsertTenantSubscriptionRequest struct {
	PlanID             string             `json:"plan_id" binding:"required"`
	Status             SubscriptionStatus `json:"status"`
	CurrentPeriodStart *time.Time         `json:"current_period_start"`
	CurrentPeriodEnd   *time.Time         `json:"current_period_end"`
	SourceOrderID      *string            `json:"source_order_id"`
}

type CreateBillingOrderRequest struct {
	TenantID        uint64             `json:"tenant_id" binding:"required"`
	PlanID          string             `json:"plan_id" binding:"required"`
	Provider        string             `json:"provider"`
	ProviderOrderID string             `json:"provider_order_id"`
	Status          BillingOrderStatus `json:"status"`
	PaidAt          *time.Time         `json:"paid_at"`
}

type ListBillingOrdersFilter struct {
	TenantID uint64
	Status   BillingOrderStatus
}

type ListTenantSubscriptionsFilter struct {
	TenantID uint64
	Status   SubscriptionStatus
}

type TenantBillingStatus struct {
	TenantID     uint64              `json:"tenant_id"`
	Plan         *BillingPlan        `json:"plan"`
	Subscription *TenantSubscription `json:"subscription,omitempty"`
	IsFree       bool                `json:"is_free"`
}

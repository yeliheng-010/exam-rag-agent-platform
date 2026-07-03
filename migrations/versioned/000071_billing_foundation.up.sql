-- Migration: 000071_billing_foundation
-- Description: Add provider-neutral billing plans, subscriptions, and orders.

DO $$ BEGIN RAISE NOTICE '[Migration 000071] Creating billing foundation tables...'; END $$;

CREATE TABLE IF NOT EXISTS billing_plans (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
    amount_cents BIGINT NOT NULL DEFAULT 0,
    interval VARCHAR(32) NOT NULL DEFAULT 'month',
    entitlements JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_plans_status ON billing_plans(status);

CREATE TABLE IF NOT EXISTS tenant_subscriptions (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id VARCHAR(36) NOT NULL REFERENCES billing_plans(id),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    source_order_id VARCHAR(36),
    created_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_plan_id ON tenant_subscriptions(plan_id);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_status ON tenant_subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_source_order_id ON tenant_subscriptions(source_order_id);

CREATE TABLE IF NOT EXISTS billing_orders (
    id VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id BIGINT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id VARCHAR(36) NOT NULL REFERENCES billing_plans(id),
    provider VARCHAR(32) NOT NULL DEFAULT 'manual',
    provider_order_id VARCHAR(128) NOT NULL DEFAULT '',
    currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
    amount_cents BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    entitlements JSONB NOT NULL DEFAULT '{}',
    created_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    paid_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_billing_orders_tenant_id ON billing_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_billing_orders_plan_id ON billing_orders(plan_id);
CREATE INDEX IF NOT EXISTS idx_billing_orders_status ON billing_orders(status);
CREATE INDEX IF NOT EXISTS idx_billing_orders_provider_order_id ON billing_orders(provider_order_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_constraint
         WHERE conname = 'fk_tenant_subscriptions_source_order'
    ) THEN
        ALTER TABLE tenant_subscriptions
            ADD CONSTRAINT fk_tenant_subscriptions_source_order
            FOREIGN KEY (source_order_id) REFERENCES billing_orders(id)
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

INSERT INTO billing_plans (
    id,
    code,
    name,
    description,
    currency,
    amount_cents,
    interval,
    entitlements,
    status,
    created_by_user_id,
    created_at,
    updated_at
) VALUES (
    'free',
    'free',
    '免费版',
    '默认免费权益',
    'CNY',
    0,
    'month',
    '{
      "class_count_limit": 1,
      "class_member_limit": 50,
      "document_parse_monthly": 100,
      "exam_structuring_monthly": 20,
      "agent_calls_monthly": 1000,
      "question_generation_monthly": 100
    }'::jsonb,
    'active',
    'system',
    NOW(),
    NOW()
) ON CONFLICT (code) DO NOTHING;

DO $$ BEGIN RAISE NOTICE '[Migration 000071] Billing foundation ready'; END $$;

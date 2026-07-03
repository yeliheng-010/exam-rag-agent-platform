-- Migration: 000071_billing_foundation (down)

DO $$ BEGIN RAISE NOTICE '[Migration 000071 DOWN] Dropping billing foundation tables...'; END $$;

ALTER TABLE IF EXISTS tenant_subscriptions
    DROP CONSTRAINT IF EXISTS fk_tenant_subscriptions_source_order;

DROP TABLE IF EXISTS billing_orders;
DROP TABLE IF EXISTS tenant_subscriptions;
DROP TABLE IF EXISTS billing_plans;

DO $$ BEGIN RAISE NOTICE '[Migration 000071 DOWN] Billing foundation tables dropped'; END $$;

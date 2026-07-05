-- Migration: 000074_exam_question_groups
-- Description: Store structured exam question groups for passage, math, and legacy-compatible questions.

CREATE TABLE IF NOT EXISTS question_groups (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    group_type VARCHAR(64) NOT NULL DEFAULT 'single_question',
    title VARCHAR(255) NOT NULL DEFAULT '',
    material_text TEXT NOT NULL DEFAULT '',
    material_format VARCHAR(32) NOT NULL DEFAULT 'plain_text',
    asset_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_year INT,
    source_region VARCHAR(128) NOT NULL DEFAULT '',
    paper_type VARCHAR(128) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_question_groups_tenant_bank
    ON question_groups(tenant_id, question_bank_id, status);

CREATE INDEX IF NOT EXISTS idx_question_groups_space
    ON question_groups(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_question_groups_sort
    ON question_groups(question_bank_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS question_group_assets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    group_id VARCHAR(36) NOT NULL REFERENCES question_groups(id) ON DELETE CASCADE,
    asset_type VARCHAR(64) NOT NULL,
    storage_uri TEXT NOT NULL DEFAULT '',
    alt_text TEXT NOT NULL DEFAULT '',
    source_chunk_id VARCHAR(36) NOT NULL DEFAULT '',
    bbox JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_question_group_assets_group
    ON question_group_assets(tenant_id, group_id, sort_order);

CREATE TABLE IF NOT EXISTS exam_question_group_drafts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    task_id VARCHAR(36) NOT NULL REFERENCES exam_structuring_tasks(id) ON DELETE CASCADE,
    material_id VARCHAR(36) NOT NULL REFERENCES exam_materials(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id),
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    group_type VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    material_text TEXT NOT NULL DEFAULT '',
    material_format VARCHAR(32) NOT NULL DEFAULT 'plain_text',
    questions_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    assets_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    strategy_code VARCHAR(64) NOT NULL DEFAULT '',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    raw_model_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    approved_group_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_question_group_drafts_task
    ON exam_question_group_drafts(tenant_id, task_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_group_drafts_space
    ON exam_question_group_drafts(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_group_drafts_bank
    ON exam_question_group_drafts(question_bank_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_group_drafts_approved_group
    ON exam_question_group_drafts(approved_group_id)
    WHERE approved_group_id <> '';

ALTER TABLE questions
    ADD COLUMN IF NOT EXISTS group_id VARCHAR(36) REFERENCES question_groups(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS question_no VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS order_in_group INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS question_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_questions_group_order
    ON questions(tenant_id, group_id, order_in_group);

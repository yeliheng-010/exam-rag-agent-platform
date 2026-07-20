-- Migration: 000082_exam_evaluation_center
-- Description: Add platform evaluation sets, immutable versions, and run baselines.

ALTER TABLE exam_rag_evaluation_runs
    ADD COLUMN IF NOT EXISTS is_baseline BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS evaluation_set_id VARCHAR(36),
    ADD COLUMN IF NOT EXISTS evaluation_set_version INTEGER;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_exam_evaluation_run_baseline
    ON exam_rag_evaluation_runs(
        tenant_id,
        question_bank_id,
        evaluation_kind,
        COALESCE(agent_id, '')
    )
    WHERE is_baseline = TRUE;

CREATE INDEX IF NOT EXISTS idx_exam_evaluation_runs_set
    ON exam_rag_evaluation_runs(tenant_id, evaluation_set_id, evaluation_set_version)
    WHERE evaluation_set_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS exam_evaluation_sets (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    evaluation_kind VARCHAR(16) NOT NULL,
    agent_id VARCHAR(36),
    name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    current_version INTEGER NOT NULL DEFAULT 1,
    created_by VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_evaluation_sets_scope
    ON exam_evaluation_sets(tenant_id, question_bank_id, evaluation_kind, updated_at DESC);

CREATE TABLE IF NOT EXISTS exam_evaluation_set_versions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    evaluation_set_id VARCHAR(36) NOT NULL REFERENCES exam_evaluation_sets(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    source_run_id VARCHAR(36) NOT NULL,
    definition_snapshot JSONB NOT NULL,
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_exam_evaluation_set_version UNIQUE(evaluation_set_id, version)
);

CREATE INDEX IF NOT EXISTS idx_exam_evaluation_set_versions_created
    ON exam_evaluation_set_versions(tenant_id, evaluation_set_id, version DESC);

ALTER TABLE exam_rag_evaluation_runs
    ADD CONSTRAINT fk_exam_evaluation_runs_set
    FOREIGN KEY (evaluation_set_id) REFERENCES exam_evaluation_sets(id) ON DELETE SET NULL;

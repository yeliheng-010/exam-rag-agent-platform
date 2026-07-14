-- Migration: 000080_exam_rag_evaluation_runs
-- Description: Persist immutable question-bank RAG evaluation run snapshots.

CREATE TABLE IF NOT EXISTS exam_rag_evaluation_runs (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    created_by VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    progress JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_snapshot JSONB NOT NULL,
    result_snapshot JSONB,
    error_message TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_rag_evaluation_runs_bank_created
    ON exam_rag_evaluation_runs(tenant_id, question_bank_id, created_at DESC);

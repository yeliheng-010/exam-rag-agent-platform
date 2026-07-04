-- Migration: 000073_exam_question_drafts
-- Description: Store LLM extracted exam question drafts before teacher approval.

CREATE TABLE IF NOT EXISTS exam_question_drafts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    task_id VARCHAR(36) NOT NULL REFERENCES exam_structuring_tasks(id) ON DELETE CASCADE,
    material_id VARCHAR(36) NOT NULL REFERENCES exam_materials(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id),
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    source_chunk_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    question_no VARCHAR(64) NOT NULL DEFAULT '',
    question_type_code VARCHAR(64) NOT NULL DEFAULT '',
    stem TEXT NOT NULL,
    options_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    answer_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    explanation TEXT NOT NULL DEFAULT '',
    difficulty VARCHAR(32) NOT NULL DEFAULT 'unknown',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    raw_model_output TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    approved_question_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_by_user_id VARCHAR(36) NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_task
    ON exam_question_drafts(tenant_id, task_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_space
    ON exam_question_drafts(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_bank
    ON exam_question_drafts(question_bank_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_question_drafts_approved_question
    ON exam_question_drafts(approved_question_id)
    WHERE approved_question_id <> '';

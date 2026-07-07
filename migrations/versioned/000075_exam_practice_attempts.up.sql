-- Migration: 000075_exam_practice_attempts
-- Description: Store student practice attempts and per-question answers for official exam question groups.

CREATE TABLE IF NOT EXISTS exam_practice_attempts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    group_id VARCHAR(36) NOT NULL REFERENCES question_groups(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'in_progress',
    question_count INT NOT NULL DEFAULT 0,
    answered_count INT NOT NULL DEFAULT 0,
    correct_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_practice_attempts_user_group
    ON exam_practice_attempts(tenant_id, user_id, group_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_practice_attempts_space
    ON exam_practice_attempts(tenant_id, space_id, created_at DESC);

CREATE TABLE IF NOT EXISTS exam_practice_answers (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    attempt_id VARCHAR(36) NOT NULL REFERENCES exam_practice_attempts(id) ON DELETE CASCADE,
    question_id VARCHAR(36) NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    question_no VARCHAR(64) NOT NULL DEFAULT '',
    answer_text TEXT NOT NULL DEFAULT '',
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    correct_answer TEXT NOT NULL DEFAULT '',
    question_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    answer_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    explanation_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_exam_practice_answer_question UNIQUE (attempt_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_practice_answers_attempt
    ON exam_practice_answers(tenant_id, attempt_id, answered_at);

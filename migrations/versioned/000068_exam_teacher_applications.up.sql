-- Migration: 000068_exam_teacher_applications
-- Description: Add teacher application approval flow for the exam platform.

CREATE TABLE IF NOT EXISTS exam_teacher_applications (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    reason TEXT NOT NULL DEFAULT '',
    reviewer_id VARCHAR(36),
    review_note TEXT NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_teacher_applications_tenant_status
    ON exam_teacher_applications(tenant_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_teacher_applications_reviewer
    ON exam_teacher_applications(reviewer_id);

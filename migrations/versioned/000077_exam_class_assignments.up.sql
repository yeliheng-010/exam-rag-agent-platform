-- Migration: 000077_exam_class_assignments
-- Description: Store class-level practice assignments and link practice attempts to assignments.

CREATE TABLE IF NOT EXISTS exam_class_assignments (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    class_id VARCHAR(36) NOT NULL REFERENCES exam_classes(id) ON DELETE CASCADE,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id) ON DELETE CASCADE,
    group_id VARCHAR(36) NOT NULL REFERENCES question_groups(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    instructions TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'published',
    due_at TIMESTAMPTZ,
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_class_assignments_class
    ON exam_class_assignments(tenant_id, class_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_class_assignments_group
    ON exam_class_assignments(tenant_id, group_id, created_at DESC);

ALTER TABLE exam_practice_attempts
    ADD COLUMN IF NOT EXISTS assignment_id VARCHAR(36) REFERENCES exam_class_assignments(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_exam_practice_attempts_assignment
    ON exam_practice_attempts(tenant_id, user_id, assignment_id, created_at DESC);

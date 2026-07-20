-- Migration: 000083_exam_assignment_notifications
-- Description: Persist assignment lifecycle notifications and teacher reminders.

CREATE TABLE IF NOT EXISTS exam_assignment_notifications (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    class_id VARCHAR(36) NOT NULL REFERENCES exam_classes(id) ON DELETE CASCADE,
    assignment_id VARCHAR(36) NOT NULL REFERENCES exam_class_assignments(id) ON DELETE CASCADE,
    group_id VARCHAR(36) NOT NULL,
    recipient_user_id VARCHAR(36) NOT NULL,
    actor_user_id VARCHAR(36) NOT NULL,
    kind VARCHAR(32) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_assignment_notifications_recipient
    ON exam_assignment_notifications(tenant_id, recipient_user_id, read_at, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_assignment_notifications_reminder
    ON exam_assignment_notifications(tenant_id, assignment_id, recipient_user_id, kind, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_assignment_notifications_class
    ON exam_assignment_notifications(tenant_id, class_id, created_at DESC);

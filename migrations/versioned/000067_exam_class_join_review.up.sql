-- Migration: 000067_exam_class_join_review
-- Description: Add class join-review indexes for pending member approvals.

CREATE INDEX IF NOT EXISTS idx_exam_class_members_class_status_role
    ON exam_class_members(class_id, status, role);

-- Migration: 000067_exam_class_join_review (down)

DROP INDEX IF EXISTS idx_exam_class_members_class_status_role;

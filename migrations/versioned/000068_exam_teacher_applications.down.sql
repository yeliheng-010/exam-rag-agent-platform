-- Migration: 000068_exam_teacher_applications (down)

DROP INDEX IF EXISTS idx_exam_teacher_applications_reviewer;
DROP INDEX IF EXISTS idx_exam_teacher_applications_tenant_status;
DROP TABLE IF EXISTS exam_teacher_applications;

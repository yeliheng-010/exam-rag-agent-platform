-- Migration: 000077_exam_class_assignments (down)

DROP INDEX IF EXISTS idx_exam_practice_attempts_assignment;

ALTER TABLE exam_practice_attempts
    DROP COLUMN IF EXISTS assignment_id;

DROP INDEX IF EXISTS idx_exam_class_assignments_group;
DROP INDEX IF EXISTS idx_exam_class_assignments_class;
DROP TABLE IF EXISTS exam_class_assignments;

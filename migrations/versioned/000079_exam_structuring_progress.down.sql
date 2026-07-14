-- Rollback: 000079_exam_structuring_progress

ALTER TABLE exam_structuring_tasks
    DROP COLUMN IF EXISTS progress;

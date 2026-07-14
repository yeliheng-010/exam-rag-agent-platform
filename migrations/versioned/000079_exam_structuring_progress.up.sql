-- Migration: 000079_exam_structuring_progress
-- Description: Persist batch progress and quality summaries for background exam structuring.

ALTER TABLE exam_structuring_tasks
    ADD COLUMN IF NOT EXISTS progress JSONB NOT NULL DEFAULT '{}'::jsonb;

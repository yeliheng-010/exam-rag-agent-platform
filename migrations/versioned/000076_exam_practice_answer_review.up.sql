-- Migration: 000076_exam_practice_answer_review
-- Description: Add student review state for practice answers.

ALTER TABLE exam_practice_answers
    ADD COLUMN IF NOT EXISTS review_status VARCHAR(32) NOT NULL DEFAULT 'unreviewed',
    ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_exam_practice_answers_review_status
    ON exam_practice_answers(tenant_id, review_status, reviewed_at DESC);

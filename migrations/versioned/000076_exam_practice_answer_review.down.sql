-- Migration: 000076_exam_practice_answer_review (down)

DROP INDEX IF EXISTS idx_exam_practice_answers_review_status;

ALTER TABLE exam_practice_answers
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS review_note,
    DROP COLUMN IF EXISTS review_status;

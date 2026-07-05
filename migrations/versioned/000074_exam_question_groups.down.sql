-- Migration: 000074_exam_question_groups (down)

DROP INDEX IF EXISTS idx_questions_group_order;

ALTER TABLE questions
    DROP COLUMN IF EXISTS question_metadata,
    DROP COLUMN IF EXISTS order_in_group,
    DROP COLUMN IF EXISTS question_no,
    DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS exam_question_group_drafts;
DROP TABLE IF EXISTS question_group_assets;
DROP TABLE IF EXISTS question_groups;

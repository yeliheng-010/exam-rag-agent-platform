-- Migration: 000075_exam_practice_attempts (down)

DROP INDEX IF EXISTS idx_exam_practice_answers_attempt;
DROP TABLE IF EXISTS exam_practice_answers;

DROP INDEX IF EXISTS idx_exam_practice_attempts_space;
DROP INDEX IF EXISTS idx_exam_practice_attempts_user_group;
DROP TABLE IF EXISTS exam_practice_attempts;

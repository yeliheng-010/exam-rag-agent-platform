-- Rollback: 000082_exam_evaluation_center

ALTER TABLE exam_rag_evaluation_runs
    DROP CONSTRAINT IF EXISTS fk_exam_evaluation_runs_set;

DROP TABLE IF EXISTS exam_evaluation_set_versions;
DROP TABLE IF EXISTS exam_evaluation_sets;

DROP INDEX IF EXISTS idx_exam_evaluation_runs_set;
DROP INDEX IF EXISTS uniq_exam_evaluation_run_baseline;

ALTER TABLE exam_rag_evaluation_runs
    DROP COLUMN IF EXISTS evaluation_set_version,
    DROP COLUMN IF EXISTS evaluation_set_id,
    DROP COLUMN IF EXISTS is_baseline;

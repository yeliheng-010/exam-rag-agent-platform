-- Rollback: 000081_exam_agent_evaluation_runs

DROP INDEX IF EXISTS idx_exam_evaluation_runs_agent;
DROP INDEX IF EXISTS idx_exam_evaluation_runs_kind_created;

ALTER TABLE exam_rag_evaluation_runs
    DROP COLUMN IF EXISTS agent_id,
    DROP COLUMN IF EXISTS evaluation_kind;

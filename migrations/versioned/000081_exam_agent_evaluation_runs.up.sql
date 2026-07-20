-- Migration: 000081_exam_agent_evaluation_runs
-- Description: Reuse question-bank evaluation runs for isolated Agent behavior evaluation.

ALTER TABLE exam_rag_evaluation_runs
    ADD COLUMN IF NOT EXISTS evaluation_kind VARCHAR(16) NOT NULL DEFAULT 'rag',
    ADD COLUMN IF NOT EXISTS agent_id VARCHAR(36);

CREATE INDEX IF NOT EXISTS idx_exam_evaluation_runs_kind_created
    ON exam_rag_evaluation_runs(tenant_id, question_bank_id, evaluation_kind, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_evaluation_runs_agent
    ON exam_rag_evaluation_runs(tenant_id, agent_id, created_at DESC)
    WHERE agent_id IS NOT NULL;

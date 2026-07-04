-- Migration: 000072_exam_material_structuring
-- Description: Register exam materials from WeKnora knowledge documents and track paper structuring tasks.

CREATE TABLE IF NOT EXISTS exam_materials (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    knowledge_base_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    material_type VARCHAR(64) NOT NULL DEFAULT 'learning_material',
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    source_year INTEGER,
    source_region VARCHAR(128) NOT NULL DEFAULT '',
    paper_type VARCHAR(128) NOT NULL DEFAULT '',
    ingest_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, knowledge_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_materials_space
    ON exam_materials(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_materials_knowledge_base
    ON exam_materials(tenant_id, knowledge_base_id);

CREATE INDEX IF NOT EXISTS idx_exam_materials_domain_subject
    ON exam_materials(tenant_id, domain_id, subject_id, material_type);

CREATE INDEX IF NOT EXISTS idx_exam_materials_ingest
    ON exam_materials(tenant_id, ingest_status, status);

CREATE TABLE IF NOT EXISTS exam_structuring_tasks (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    material_id VARCHAR(36) NOT NULL REFERENCES exam_materials(id),
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    question_bank_id VARCHAR(36) NOT NULL REFERENCES question_banks(id),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    strategy VARCHAR(64) NOT NULL DEFAULT 'manual_review',
    source_chunk_count INTEGER NOT NULL DEFAULT 0,
    structured_question_count INTEGER NOT NULL DEFAULT 0,
    review_required BOOLEAN NOT NULL DEFAULT TRUE,
    error_message TEXT NOT NULL DEFAULT '',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exam_structuring_tasks_space
    ON exam_structuring_tasks(tenant_id, space_id, status);

CREATE INDEX IF NOT EXISTS idx_exam_structuring_tasks_material
    ON exam_structuring_tasks(tenant_id, material_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exam_structuring_tasks_bank
    ON exam_structuring_tasks(question_bank_id);

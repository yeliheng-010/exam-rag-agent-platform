-- Migration: 000066_exam_space_resources
-- Description: Bind WeKnora resources to exam platform spaces and exam metadata.

CREATE TABLE IF NOT EXISTS exam_space_resources (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    space_id VARCHAR(36) NOT NULL REFERENCES exam_spaces(id),
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(36) NOT NULL,
    domain_id VARCHAR(36) NOT NULL REFERENCES exam_domains(id),
    subject_id VARCHAR(36) REFERENCES exam_subjects(id),
    material_type VARCHAR(64) NOT NULL DEFAULT 'learning_material',
    review_status VARCHAR(32) NOT NULL DEFAULT 'private',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_by_user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, resource_type, resource_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_space_resources_space
    ON exam_space_resources(tenant_id, space_id, resource_type, status);

CREATE INDEX IF NOT EXISTS idx_exam_space_resources_domain_subject
    ON exam_space_resources(tenant_id, domain_id, subject_id, material_type);

CREATE INDEX IF NOT EXISTS idx_exam_space_resources_review
    ON exam_space_resources(tenant_id, review_status, status);

-- Rollback: 000078_exam_space_resource_scope

DROP INDEX IF EXISTS idx_exam_space_resources_unique_space_resource;

ALTER TABLE exam_space_resources
    ADD CONSTRAINT exam_space_resources_tenant_id_resource_type_resource_id_key
    UNIQUE(tenant_id, resource_type, resource_id);

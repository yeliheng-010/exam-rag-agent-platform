-- Migration: 000078_exam_space_resource_scope
-- Description: Scope exam resource bindings by space so one knowledge base can be bound to multiple classes.

ALTER TABLE exam_space_resources
    DROP CONSTRAINT IF EXISTS exam_space_resources_tenant_id_resource_type_resource_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_space_resources_unique_space_resource
    ON exam_space_resources(tenant_id, space_id, resource_type, resource_id);

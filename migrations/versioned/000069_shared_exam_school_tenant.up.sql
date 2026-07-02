-- Migration: 000069_shared_exam_school_tenant
-- Description: Align public self-service users with the shared school tenant model.

DO $$ BEGIN RAISE NOTICE '[Migration 000069] Aligning viewer signups to shared exam school tenant...'; END $$;

WITH school AS (
    SELECT id AS tenant_id
      FROM tenants
     WHERE status = 'active'
       AND deleted_at IS NULL
     ORDER BY created_at ASC, id ASC
     LIMIT 1
),
candidates AS (
    SELECT u.id AS user_id,
           u.tenant_id AS old_tenant_id,
           s.tenant_id AS school_tenant_id,
           u.created_at AS joined_at
      FROM users u
      JOIN school s ON TRUE
      JOIN tenant_members tm
        ON tm.user_id = u.id
       AND tm.tenant_id = u.tenant_id
       AND tm.status = 'active'
       AND tm.deleted_at IS NULL
     WHERE u.deleted_at IS NULL
       AND u.is_active = TRUE
       AND u.tenant_id IS NOT NULL
       AND u.tenant_id <> s.tenant_id
       AND tm.role = 'viewer'
)
INSERT INTO tenant_members (user_id, tenant_id, role, status, joined_at, created_at, updated_at)
SELECT c.user_id,
       c.school_tenant_id,
       'viewer',
       'active',
       COALESCE(c.joined_at, NOW()),
       NOW(),
       NOW()
  FROM candidates c
 WHERE NOT EXISTS (
       SELECT 1
         FROM tenant_members existing
        WHERE existing.user_id = c.user_id
          AND existing.tenant_id = c.school_tenant_id
          AND existing.deleted_at IS NULL
 )
ON CONFLICT DO NOTHING;

WITH school AS (
    SELECT id AS tenant_id
      FROM tenants
     WHERE status = 'active'
       AND deleted_at IS NULL
     ORDER BY created_at ASC, id ASC
     LIMIT 1
),
candidates AS (
    SELECT u.id AS user_id,
           u.tenant_id AS old_tenant_id,
           s.tenant_id AS school_tenant_id
      FROM users u
      JOIN school s ON TRUE
      JOIN tenant_members tm
        ON tm.user_id = u.id
       AND tm.tenant_id = u.tenant_id
       AND tm.status = 'active'
       AND tm.deleted_at IS NULL
     WHERE u.deleted_at IS NULL
       AND u.is_active = TRUE
       AND u.tenant_id IS NOT NULL
       AND u.tenant_id <> s.tenant_id
       AND tm.role = 'viewer'
),
movable_applications AS (
    SELECT app.id AS application_id,
           c.school_tenant_id
      FROM exam_teacher_applications app
      JOIN candidates c
        ON c.user_id = app.user_id
       AND c.old_tenant_id = app.tenant_id
     WHERE NOT EXISTS (
           SELECT 1
             FROM exam_teacher_applications existing
            WHERE existing.tenant_id = c.school_tenant_id
              AND existing.user_id = app.user_id
     )
)
UPDATE exam_teacher_applications app
   SET tenant_id = movable_applications.school_tenant_id,
       updated_at = NOW()
  FROM movable_applications
 WHERE app.id = movable_applications.application_id;

WITH school AS (
    SELECT id AS tenant_id
      FROM tenants
     WHERE status = 'active'
       AND deleted_at IS NULL
     ORDER BY created_at ASC, id ASC
     LIMIT 1
),
candidates AS (
    SELECT u.id AS user_id,
           s.tenant_id AS school_tenant_id
      FROM users u
      JOIN school s ON TRUE
      JOIN tenant_members tm
        ON tm.user_id = u.id
       AND tm.tenant_id = u.tenant_id
       AND tm.status = 'active'
       AND tm.deleted_at IS NULL
     WHERE u.deleted_at IS NULL
       AND u.is_active = TRUE
       AND u.tenant_id IS NOT NULL
       AND u.tenant_id <> s.tenant_id
       AND tm.role = 'viewer'
)
UPDATE users u
   SET tenant_id = candidates.school_tenant_id,
       updated_at = NOW()
  FROM candidates
 WHERE u.id = candidates.user_id;

DO $$ BEGIN RAISE NOTICE '[Migration 000069] Shared exam school tenant alignment ready'; END $$;

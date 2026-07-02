-- Migration: 000070_revoke_legacy_school_alignment_tokens
-- Description: Force viewer users aligned to the shared school tenant to re-login.

DO $$ BEGIN RAISE NOTICE '[Migration 000070] Revoking legacy tokens after shared school tenant alignment...'; END $$;

WITH school AS (
    SELECT id AS tenant_id
      FROM tenants
     WHERE status = 'active'
       AND deleted_at IS NULL
     ORDER BY created_at ASC, id ASC
     LIMIT 1
),
aligned_viewers AS (
    SELECT DISTINCT u.id AS user_id,
           s.tenant_id AS school_tenant_id
      FROM users u
      JOIN school s ON TRUE
      JOIN tenant_members school_member
        ON school_member.user_id = u.id
       AND school_member.tenant_id = s.tenant_id
       AND school_member.role = 'viewer'
       AND school_member.status = 'active'
       AND school_member.deleted_at IS NULL
      JOIN tenant_members legacy_member
        ON legacy_member.user_id = u.id
       AND legacy_member.tenant_id <> s.tenant_id
       AND legacy_member.role = 'viewer'
       AND legacy_member.status = 'active'
       AND legacy_member.deleted_at IS NULL
     WHERE u.deleted_at IS NULL
       AND u.is_active = TRUE
       AND u.tenant_id = s.tenant_id
)
UPDATE users u
   SET preferences = COALESCE(u.preferences, '{}'::jsonb) - 'last_active_tenant_id',
       updated_at = NOW()
  FROM aligned_viewers av
 WHERE u.id = av.user_id
   AND COALESCE(u.preferences, '{}'::jsonb) ? 'last_active_tenant_id'
   AND (u.preferences->>'last_active_tenant_id')::BIGINT <> av.school_tenant_id;

WITH school AS (
    SELECT id AS tenant_id
      FROM tenants
     WHERE status = 'active'
       AND deleted_at IS NULL
     ORDER BY created_at ASC, id ASC
     LIMIT 1
),
aligned_viewers AS (
    SELECT DISTINCT u.id AS user_id
      FROM users u
      JOIN school s ON TRUE
      JOIN tenant_members school_member
        ON school_member.user_id = u.id
       AND school_member.tenant_id = s.tenant_id
       AND school_member.role = 'viewer'
       AND school_member.status = 'active'
       AND school_member.deleted_at IS NULL
      JOIN tenant_members legacy_member
        ON legacy_member.user_id = u.id
       AND legacy_member.tenant_id <> s.tenant_id
       AND legacy_member.role = 'viewer'
       AND legacy_member.status = 'active'
       AND legacy_member.deleted_at IS NULL
     WHERE u.deleted_at IS NULL
       AND u.is_active = TRUE
       AND u.tenant_id = s.tenant_id
)
UPDATE auth_tokens token
   SET is_revoked = TRUE,
       updated_at = NOW()
  FROM aligned_viewers av
 WHERE token.user_id = av.user_id
   AND token.is_revoked = FALSE
   AND token.token_type IN ('access_token', 'refresh_token');

DO $$ BEGIN RAISE NOTICE '[Migration 000070] Legacy tokens revoked after shared school tenant alignment'; END $$;

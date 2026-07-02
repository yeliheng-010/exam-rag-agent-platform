-- Migration: 000070_revoke_legacy_school_alignment_tokens (down)
--
-- Token revocation is intentionally irreversible.

DO $$ BEGIN RAISE NOTICE '[Migration 000070 DOWN] No-op: revoked legacy tokens are not restored'; END $$;

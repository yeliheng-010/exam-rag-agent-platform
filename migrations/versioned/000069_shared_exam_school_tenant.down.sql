-- Migration: 000069_shared_exam_school_tenant (down)
--
-- This migration intentionally keeps the aligned school memberships and home
-- tenant pointers. Reversing them safely would require knowing which rows were
-- legacy personal signups versus intentional cross-tenant memberships.

DO $$ BEGIN RAISE NOTICE '[Migration 000069 DOWN] No-op: shared school tenant alignment is not automatically reversible'; END $$;

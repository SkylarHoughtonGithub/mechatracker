-- ============================================================
-- mechatracker — Migration 001: Rollback
-- ============================================================

DROP TABLE IF EXISTS deleted_items    CASCADE;
DROP TABLE IF EXISTS repair_history   CASCADE;
DROP TABLE IF EXISTS resources        CASCADE;
DROP TABLE IF EXISTS photos           CASCADE;
DROP TABLE IF EXISTS repair_notes     CASCADE;
DROP TABLE IF EXISTS parts            CASCADE;
DROP TABLE IF EXISTS repair_steps     CASCADE;
DROP TABLE IF EXISTS repairs          CASCADE;
DROP TABLE IF EXISTS repair_templates CASCADE;
DROP TABLE IF EXISTS vehicles         CASCADE;
DROP TABLE IF EXISTS invites          CASCADE;
DROP TABLE IF EXISTS users            CASCADE;
DROP TABLE IF EXISTS organizations    CASCADE;

DROP FUNCTION IF EXISTS set_updated_at CASCADE;

DROP TYPE IF EXISTS difficulty    CASCADE;
DROP TYPE IF EXISTS source_type   CASCADE;
DROP TYPE IF EXISTS part_status   CASCADE;
DROP TYPE IF EXISTS repair_status CASCADE;
DROP TYPE IF EXISTS user_role     CASCADE;
DROP TYPE IF EXISTS org_plan      CASCADE;

DROP EXTENSION IF EXISTS "pg_trgm"   CASCADE;
DROP EXTENSION IF EXISTS "uuid-ossp" CASCADE;
-- ============================================================
-- mechatracker — Migration 001: Core Schema
-- Source of truth: docs/architecture/data_model.md
-- Run: task db:migrate
-- ============================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
-- pgvector added in a later migration when PDF indexing is built (Phase 4)
-- CREATE EXTENSION IF NOT EXISTS "vector";

-- ============================================================
-- ENUMS
-- ============================================================

CREATE TYPE org_plan      AS ENUM ('personal', 'shop', 'fleet');
CREATE TYPE user_role     AS ENUM ('owner', 'manager', 'technician');
CREATE TYPE repair_status AS ENUM ('draft', 'in_progress', 'pending_parts', 'completed');
CREATE TYPE part_status   AS ENUM ('needed', 'ordered', 'in_stock', 'installed');
CREATE TYPE source_type   AS ENUM ('youtube', 'forum', 'pdf', 'parts', 'manual', 'other');
CREATE TYPE difficulty    AS ENUM ('easy', 'medium', 'hard');

-- ============================================================
-- ORGANIZATIONS
-- Every user belongs to exactly one org. Solo users get a
-- personal org (personal=true) created automatically on signup.
-- ============================================================

CREATE TABLE organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    plan       org_plan NOT NULL DEFAULT 'personal',
    personal   BOOLEAN NOT NULL DEFAULT false,  -- true = auto-created for solo user
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- USERS
-- auth_id = Clerk user ID. Used only for JWT lookup.
-- All internal FK relationships use our UUID id.
-- ============================================================

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_id   TEXT NOT NULL UNIQUE,
    org_id     UUID NOT NULL REFERENCES organizations(id),
    role       user_role NOT NULL DEFAULT 'owner',
    email      TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_clerk_id ON users(clerk_id);
CREATE INDEX idx_users_org_id   ON users(org_id);

-- ============================================================
-- VEHICLES
-- Scoped to an org, not a user — a shop owns its vehicles.
-- unit_label is for fleet use ("Unit #07").
-- ============================================================

CREATE TABLE vehicles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES organizations(id),
    year       SMALLINT NOT NULL,
    make       TEXT NOT NULL,
    model      TEXT NOT NULL,
    trim       TEXT,
    vin        TEXT,
    mileage    INT,
    unit_label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vehicles_org_id ON vehicles(org_id);

-- ============================================================
-- REPAIR TEMPLATES
-- System sentinel org: 00000000-0000-0000-0000-000000000001
-- Global templates use this org_id. Merged at query time:
--   WHERE org_id = $org OR org_id = '00000000-0000-0000-0000-000000000001'
-- ============================================================

CREATE TABLE repair_templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id),
    system         TEXT NOT NULL,
    title          TEXT NOT NULL,
    difficulty     difficulty,
    est_minutes    INT,
    default_steps  JSONB NOT NULL DEFAULT '[]',
    default_parts  JSONB NOT NULL DEFAULT '[]',
    default_tools  JSONB NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_templates_org_id ON repair_templates(org_id);
CREATE INDEX idx_repair_templates_system ON repair_templates(system);

-- full-text search on template titles
CREATE INDEX idx_repair_templates_title_trgm
    ON repair_templates USING gin(title gin_trgm_ops);

-- ============================================================
-- REPAIRS
-- org_id is denormalized here for fast permission scoping
-- without a join through vehicles.
-- ============================================================

CREATE TABLE repairs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vehicle_id       UUID NOT NULL REFERENCES vehicles(id),
    org_id           UUID NOT NULL REFERENCES organizations(id),
    created_by       UUID NOT NULL REFERENCES users(id),
    assigned_to      UUID REFERENCES users(id),
    template_id      UUID REFERENCES repair_templates(id),

    title            TEXT NOT NULL,
    system           TEXT NOT NULL,
    status           repair_status NOT NULL DEFAULT 'draft',
    progress_pct     SMALLINT NOT NULL DEFAULT 0
                     CHECK (progress_pct BETWEEN 0 AND 100),

    total_parts_cost NUMERIC(10,2) NOT NULL DEFAULT 0,

    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repairs_vehicle_id  ON repairs(vehicle_id);
CREATE INDEX idx_repairs_org_id      ON repairs(org_id);
CREATE INDEX idx_repairs_assigned_to ON repairs(assigned_to);
CREATE INDEX idx_repairs_status      ON repairs(status);
CREATE INDEX idx_repairs_title_trgm  ON repairs USING gin(title gin_trgm_ops);

-- ============================================================
-- REPAIR STEPS
-- ============================================================

CREATE TABLE repair_steps (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id    UUID NOT NULL REFERENCES repairs(id),
    position     SMALLINT NOT NULL,
    title        TEXT NOT NULL,
    notes        TEXT,
    torque_spec  TEXT,
    completed    BOOLEAN NOT NULL DEFAULT false,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (repair_id, position)
);

CREATE INDEX idx_repair_steps_repair_id ON repair_steps(repair_id);

-- ============================================================
-- PARTS
-- ============================================================

CREATE TABLE parts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id   UUID NOT NULL REFERENCES repairs(id),
    name        TEXT NOT NULL,
    part_number TEXT,
    supplier    TEXT,
    status      part_status NOT NULL DEFAULT 'needed',
    cost        NUMERIC(10,2),
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_parts_repair_id ON parts(repair_id);

-- ============================================================
-- REPAIR NOTES
-- ============================================================

CREATE TABLE repair_notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id  UUID NOT NULL REFERENCES repairs(id),
    user_id    UUID NOT NULL REFERENCES users(id),
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_notes_repair_id ON repair_notes(repair_id);

-- ============================================================
-- PHOTOS
-- step_id is nullable — photo can be repair-level or step-level.
-- storage_key is the DO Spaces object key.
-- ============================================================

CREATE TABLE photos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id   UUID NOT NULL REFERENCES repairs(id),
    step_id     UUID REFERENCES repair_steps(id),
    user_id     UUID NOT NULL REFERENCES users(id),
    storage_key TEXT NOT NULL,
    caption     TEXT,
    size_bytes  INT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_photos_repair_id ON photos(repair_id);
CREATE INDEX idx_photos_step_id   ON photos(step_id);

-- ============================================================
-- RESOURCES
-- metadata is JSONB — shape varies by source_type:
--   youtube: {duration_seconds, channel, view_count}
--   forum:   {reply_count, view_count, domain}
--   pdf:     {page_count, file_size_bytes, storage_key}
--   parts:   {price, supplier, in_stock, part_number}
-- ============================================================

CREATE TABLE resources (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id     UUID NOT NULL REFERENCES repairs(id),
    source_type   source_type NOT NULL,
    title         TEXT NOT NULL,
    url           TEXT NOT NULL,
    thumbnail_url TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    saved_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_resources_repair_id ON resources(repair_id);

-- ============================================================
-- REPAIR HISTORY
-- Written once when a repair is completed. Never updated.
-- repair_id, vehicle_id, org_id, created_by, assigned_to are
-- plain columns (not FKs) so this row survives deletion of
-- the source records. Snapshot columns are JSONB so they
-- remain readable regardless of future schema changes.
-- ============================================================

CREATE TABLE repair_history (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repair_id          UUID NOT NULL,
    vehicle_id         UUID NOT NULL,
    org_id             UUID NOT NULL,
    created_by         UUID NOT NULL,
    assigned_to        UUID,

    -- Human-readable snapshots frozen at completion time
    vehicle_snapshot   TEXT NOT NULL,   -- "2018 Ford F-150 XLT 4WD — Unit #07"
    assigned_snapshot  TEXT,            -- "Jane Smith"

    title              TEXT NOT NULL,
    system             TEXT NOT NULL,
    total_parts_cost   NUMERIC(10,2) NOT NULL,
    total_steps        SMALLINT NOT NULL,

    steps_snapshot     JSONB NOT NULL,
    parts_snapshot     JSONB NOT NULL,
    notes_snapshot     JSONB NOT NULL,
    resources_snapshot JSONB NOT NULL,

    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ NOT NULL,
    archived_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_history_org_id     ON repair_history(org_id);
CREATE INDEX idx_repair_history_vehicle_id ON repair_history(vehicle_id);
CREATE INDEX idx_repair_history_repair_id  ON repair_history(repair_id);

-- ============================================================
-- DELETED ITEMS
-- Soft-delete buffer. Background worker purges rows where
-- purge_at < now(), runs nightly.
-- ============================================================

CREATE TABLE deleted_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    org_id      UUID NOT NULL,
    deleted_by  UUID NOT NULL,
    snapshot    JSONB NOT NULL,
    purge_at    TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '30 days',
    deleted_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deleted_items_org_id   ON deleted_items(org_id);
CREATE INDEX idx_deleted_items_purge_at ON deleted_items(purge_at);

-- ============================================================
-- UPDATED_AT TRIGGER
-- Auto-maintain updated_at on tables that have it.
-- ============================================================

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_repairs_updated_at
    BEFORE UPDATE ON repairs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- SYSTEM SENTINEL ORG
-- Global repair templates use this org_id.
-- Inserted here so FK constraints on repair_templates hold.
-- ============================================================

INSERT INTO organizations (id, name, slug, plan, personal)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'System',
    'system',
    'personal',
    false
);
---
layout: default
title: Data Model Architecture
parent: Architecture
---

## Design decisions

| Decision | Choice | Rationale |
|---|---|---|
| Primary keys | UUID everywhere | Auth-provider-agnostic, safe to expose in URLs |
| Auth ID | `clerk_id` stored as a plain column on `users` | Internal UUIDs own all FK relationships — Clerk is swappable |
| Orgs always required | Solo users get a personal org on signup | Uniform data model — no branching logic for solo vs team |
| Assigned technician | Nullable `assigned_to` on repairs | Solo users self-assign, managers assign to techs |
| Org ID on repairs | Denormalized directly on `repairs` | Fast permission scoping without joins |
| Deletes | Hard delete everywhere | Confirmation dialog on client — no soft-delete complexity in queries |
| Recently deleted | Separate `deleted_items` buffer table | Snapshot on delete, auto-purge after 30 days via scheduled worker |
| Repair history | Separate `repair_history` archive table | Denormalized snapshot — survives vehicle/user deletion, survives schema changes |
| Templates | Single table, scoped by `org_id` | Global templates use a system sentinel org UUID — merged at read time |
| `resources.metadata` | `jsonb` | YouTube, forums, PDFs, parts all have different shapes |
| `repair_history` snapshots | `jsonb` columns | Frozen state — readable forever regardless of future schema changes |

---

## Enums

```sql
CREATE TYPE org_plan       AS ENUM ('personal', 'shop', 'fleet');
CREATE TYPE user_role      AS ENUM ('owner', 'manager', 'technician');
CREATE TYPE repair_status  AS ENUM ('draft', 'in_progress', 'pending_parts', 'completed');
CREATE TYPE part_status    AS ENUM ('needed', 'ordered', 'in_stock', 'installed');
CREATE TYPE source_type    AS ENUM ('youtube', 'forum', 'pdf', 'parts', 'manual', 'other');
CREATE TYPE difficulty     AS ENUM ('easy', 'medium', 'hard');
```

---

## Tables

### organizations
```sql
CREATE TABLE organizations (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name        TEXT NOT NULL,
  slug        TEXT NOT NULL UNIQUE,
  plan        org_plan NOT NULL DEFAULT 'personal',
  personal    BOOLEAN NOT NULL DEFAULT false,   -- true = auto-created for solo user
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### users
```sql
CREATE TABLE users (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  clerk_id    TEXT NOT NULL UNIQUE,             -- Clerk user ID, lookup only
  org_id      UUID NOT NULL REFERENCES organizations(id),
  role        user_role NOT NULL DEFAULT 'owner',
  email       TEXT NOT NULL,
  name        TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_clerk_id ON users(clerk_id);
CREATE INDEX idx_users_org_id   ON users(org_id);
```

### vehicles
```sql
CREATE TABLE vehicles (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id      UUID NOT NULL REFERENCES organizations(id),
  year        SMALLINT NOT NULL,
  make        TEXT NOT NULL,
  model       TEXT NOT NULL,
  trim        TEXT,
  vin         TEXT,
  mileage     INT,
  unit_label  TEXT,                             -- Fleet unit number e.g. "Unit #07"
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vehicles_org_id ON vehicles(org_id);
```

### repair_templates
```sql
-- System sentinel org: 00000000-0000-0000-0000-000000000001
-- Global templates use this org_id. Org templates use the real org UUID.
-- Query merges both: WHERE org_id = $org OR org_id = '00000000-0000-0000-0000-000000000001'

CREATE TABLE repair_templates (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id         UUID NOT NULL REFERENCES organizations(id),
  system         TEXT NOT NULL,                 -- 'suspension', 'engine', 'brakes', etc.
  title          TEXT NOT NULL,
  difficulty     difficulty,
  est_minutes    INT,
  default_steps  JSONB NOT NULL DEFAULT '[]',   -- [{position, title, notes, torque_spec}]
  default_parts  JSONB NOT NULL DEFAULT '[]',   -- [{name, part_number, notes}]
  default_tools  JSONB NOT NULL DEFAULT '[]',   -- [{name, notes}]
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_templates_org_id ON repair_templates(org_id);
CREATE INDEX idx_repair_templates_system ON repair_templates(system);
```

### repairs
```sql
CREATE TABLE repairs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vehicle_id        UUID NOT NULL REFERENCES vehicles(id),
  org_id            UUID NOT NULL REFERENCES organizations(id), -- denormalized for fast scoping
  created_by        UUID NOT NULL REFERENCES users(id),
  assigned_to       UUID REFERENCES users(id),                  -- nullable
  template_id       UUID REFERENCES repair_templates(id),       -- nullable, seed reference only
  title             TEXT NOT NULL,
  system            TEXT NOT NULL,
  status            repair_status NOT NULL DEFAULT 'draft',
  progress_pct      SMALLINT NOT NULL DEFAULT 0
                    CHECK (progress_pct BETWEEN 0 AND 100),
  total_parts_cost  NUMERIC(10,2) NOT NULL DEFAULT 0,
  started_at        TIMESTAMPTZ,
  completed_at      TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repairs_vehicle_id  ON repairs(vehicle_id);
CREATE INDEX idx_repairs_org_id      ON repairs(org_id);
CREATE INDEX idx_repairs_assigned_to ON repairs(assigned_to);
CREATE INDEX idx_repairs_status      ON repairs(status);
```

### repair_steps
```sql
CREATE TABLE repair_steps (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id     UUID NOT NULL REFERENCES repairs(id),
  position      SMALLINT NOT NULL,
  title         TEXT NOT NULL,
  notes         TEXT,
  torque_spec   TEXT,
  completed     BOOLEAN NOT NULL DEFAULT false,
  completed_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  UNIQUE (repair_id, position)
);

CREATE INDEX idx_repair_steps_repair_id ON repair_steps(repair_id);
```

### parts
```sql
CREATE TABLE parts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id    UUID NOT NULL REFERENCES repairs(id),
  name         TEXT NOT NULL,
  part_number  TEXT,
  supplier     TEXT,
  status       part_status NOT NULL DEFAULT 'needed',
  cost         NUMERIC(10,2),
  notes        TEXT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_parts_repair_id ON parts(repair_id);
```

### repair_notes
```sql
CREATE TABLE repair_notes (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id  UUID NOT NULL REFERENCES repairs(id),
  user_id    UUID NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_notes_repair_id ON repair_notes(repair_id);
```

### photos
```sql
CREATE TABLE photos (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id    UUID NOT NULL REFERENCES repairs(id),
  step_id      UUID REFERENCES repair_steps(id),   -- nullable — repair-level or step-level
  user_id      UUID NOT NULL REFERENCES users(id),
  storage_key  TEXT NOT NULL,                       -- DO Spaces object key
  caption      TEXT,
  size_bytes   INT,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_photos_repair_id ON photos(repair_id);
CREATE INDEX idx_photos_step_id   ON photos(step_id);
```

### resources
```sql
CREATE TABLE resources (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id      UUID NOT NULL REFERENCES repairs(id),
  source_type    source_type NOT NULL,
  title          TEXT NOT NULL,
  url            TEXT NOT NULL,
  thumbnail_url  TEXT,
  metadata       JSONB NOT NULL DEFAULT '{}',
  -- YouTube: {duration_seconds, channel, view_count}
  -- Forum:   {reply_count, view_count, domain}
  -- PDF:     {page_count, file_size_bytes, storage_key}
  -- Parts:   {price, supplier, in_stock, part_number}
  saved_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_resources_repair_id ON resources(repair_id);
```

### repair_history
```sql
-- Written once when a repair is completed. Never updated.
-- Retains repair_id as a plain column (not FK) so it survives repair deletion.

CREATE TABLE repair_history (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  repair_id         UUID NOT NULL,               -- plain column, not FK
  vehicle_id        UUID NOT NULL,               -- plain column, not FK
  org_id            UUID NOT NULL,               -- plain column, not FK
  created_by        UUID NOT NULL,               -- plain column, not FK
  assigned_to       UUID,                        -- plain column, not FK
  vehicle_snapshot  TEXT NOT NULL,               -- "2018 Ford F-150 XLT 4WD — Unit #07"
  assigned_snapshot TEXT,                        -- "Jane Smith" at time of completion
  title             TEXT NOT NULL,
  system            TEXT NOT NULL,
  total_parts_cost  NUMERIC(10,2) NOT NULL,
  total_steps       SMALLINT NOT NULL,
  steps_snapshot    JSONB NOT NULL,              -- full step list at completion
  parts_snapshot    JSONB NOT NULL,              -- full parts list at completion
  notes_snapshot    JSONB NOT NULL,              -- all notes at completion
  resources_snapshot JSONB NOT NULL,             -- all linked resources at completion
  started_at        TIMESTAMPTZ,
  completed_at      TIMESTAMPTZ NOT NULL,
  archived_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_repair_history_org_id     ON repair_history(org_id);
CREATE INDEX idx_repair_history_vehicle_id ON repair_history(vehicle_id);
CREATE INDEX idx_repair_history_repair_id  ON repair_history(repair_id);
```

### deleted_items
```sql
-- Temporary holding buffer for recently deleted records.
-- Background worker hard-purges rows where purge_at < now() nightly.

CREATE TABLE deleted_items (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entity_type  TEXT NOT NULL,                   -- 'repair', 'vehicle', 'photo', etc.
  entity_id    UUID NOT NULL,
  org_id       UUID NOT NULL,
  deleted_by   UUID NOT NULL,
  snapshot     JSONB NOT NULL,                  -- full row at time of deletion
  purge_at     TIMESTAMPTZ NOT NULL             -- DEFAULT now() + INTERVAL '30 days'
                 DEFAULT now() + INTERVAL '30 days',
  deleted_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_deleted_items_org_id   ON deleted_items(org_id);
CREATE INDEX idx_deleted_items_purge_at ON deleted_items(purge_at);
```

---

## Template query

Global and org templates merged at read time. Org templates sort first — an org can shadow a global template by creating one with the same system + title.

```sql
SELECT *
FROM repair_templates
WHERE org_id = $1                                          -- caller's org
   OR org_id = '00000000-0000-0000-0000-000000000001'     -- system sentinel org
ORDER BY
  (org_id = $1) DESC,                                     -- org templates first
  system,
  title;
```

---

## Signup flow

When Clerk fires a new user webhook, the Go API runs this in a single transaction:

```
1. INSERT into organizations (personal = true, plan = 'personal')
2. INSERT into users (org_id = new org, role = 'owner')
3. INSERT into repair_templates for each global template... (skip — merged at read time)
```

No template copying needed. Global templates are always available via the merge query above.

---

## Key relationships

```
organizations
  └── users (many)
  └── vehicles (many)
  └── repairs (many, denormalized org_id)
      └── repair_steps (many, ordered by position)
      └── parts (many)
      └── repair_notes (many)
      └── photos (many, optionally pinned to a step)
      └── resources (many)
      └── repair_history (one, written on completion)

repair_templates
  └── org_id = system sentinel → global templates
  └── org_id = real org UUID  → org-specific templates
```

---

## Open questions

| Question | Notes |
|---|---|
| pgvector for PDF search | Add `vector` column to a `document_chunks` table when PDF indexing is built in Phase 4 |
| Mileage tracking | Log mileage per repair for service interval tracking — future feature |
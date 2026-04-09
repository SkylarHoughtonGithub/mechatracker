-- ============================================================
-- mechatracker — Repair Queries
-- ============================================================

-- name: GetRepair :one
SELECT r.* FROM repairs r
WHERE r.id = $1
  AND r.org_id = $2
LIMIT 1;

-- name: ListRepairsByVehicle :many
SELECT * FROM repairs
WHERE vehicle_id = $1
  AND org_id = $2
ORDER BY created_at DESC;

-- name: ListActiveRepairs :many
-- Active repairs assigned to (or created by) the requesting user.
SELECT r.*,
       v.year, v.make, v.model, v.trim, v.unit_label
FROM repairs r
JOIN vehicles v ON v.id = r.vehicle_id
WHERE r.org_id = $1
  AND r.status = 'in_progress'
  AND (r.assigned_to = $2 OR r.created_by = $2)
ORDER BY r.started_at DESC NULLS LAST;

-- name: CreateRepair :one
INSERT INTO repairs (
    vehicle_id, org_id, created_by, assigned_to, template_id,
    title, system, status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, 'draft'
)
RETURNING *;

-- name: UpdateRepairStatus :one
UPDATE repairs SET
    status       = @status::repair_status,
    started_at   = CASE WHEN @status::repair_status = 'in_progress' AND started_at IS NULL
                        THEN now() ELSE started_at END,
    completed_at = CASE WHEN @status::repair_status = 'completed' THEN now() ELSE completed_at END
WHERE id = @id
  AND org_id = @org_id
RETURNING *;

-- name: UpdateRepair :one
UPDATE repairs SET
    title       = COALESCE(sqlc.narg('title'),       title),
    system      = COALESCE(sqlc.narg('system'),      system),
    assigned_to = COALESCE(sqlc.narg('assigned_to'), assigned_to)
WHERE id = $1
  AND org_id = $2
RETURNING *;

-- name: UpdateRepairProgress :one
-- Recalculates progress_pct from completed steps. Called after
-- any step complete/uncomplete. total_parts_cost is also refreshed.
UPDATE repairs SET
    progress_pct     = $2,
    total_parts_cost = $3
WHERE id = $1
RETURNING *;

-- name: SearchRepairs :many
SELECT r.*,
       v.year, v.make, v.model, v.unit_label
FROM repairs r
JOIN vehicles v ON v.id = r.vehicle_id
WHERE r.org_id = $1
  AND r.title % $2
ORDER BY similarity(r.title, $2) DESC
LIMIT 20;

-- name: WriteRepairHistory :one
-- Called once when a repair reaches 'completed'. Caller builds
-- the snapshot values in Go — the vehicle_snapshot and
-- assigned_snapshot strings are computed before this call.
INSERT INTO repair_history (
    repair_id, vehicle_id, org_id, created_by, assigned_to,
    vehicle_snapshot, assigned_snapshot,
    title, system, total_parts_cost, total_steps,
    steps_snapshot, parts_snapshot, notes_snapshot, resources_snapshot,
    started_at, completed_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7,
    $8, $9, $10, $11,
    $12, $13, $14, $15,
    $16, $17
)
RETURNING *;

-- name: ListRepairHistory :many
SELECT * FROM repair_history
WHERE org_id = $1
  AND vehicle_id = $2
ORDER BY completed_at DESC;
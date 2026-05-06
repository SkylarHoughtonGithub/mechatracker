-- ============================================================
-- mechatracker — Vehicle Queries
-- All queries scope to org_id — never expose cross-org data.
-- ============================================================

-- name: GetVehicle :one
SELECT * FROM vehicles
WHERE id = $1
  AND org_id = $2
LIMIT 1;

-- name: ListVehicles :many
SELECT * FROM vehicles
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: CreateVehicle :one
INSERT INTO vehicles (org_id, year, make, model, trim, vin, mileage, unit_label)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateVehicle :one
UPDATE vehicles SET
    year       = COALESCE(sqlc.narg('year'),       year),
    make       = COALESCE(sqlc.narg('make'),       make),
    model      = COALESCE(sqlc.narg('model'),      model),
    trim       = COALESCE(sqlc.narg('trim'),       trim),
    vin        = COALESCE(sqlc.narg('vin'),        vin),
    mileage    = COALESCE(sqlc.narg('mileage'),    mileage),
    unit_label = COALESCE(sqlc.narg('unit_label'), unit_label)
WHERE id = $1
  AND org_id = $2
RETURNING *;

-- name: DeleteVehicle :one
-- Returns the deleted row so the caller can snapshot it into deleted_items.
DELETE FROM vehicles
WHERE id = $1
  AND org_id = $2
RETURNING *;

-- name: DeleteVehicleSystem :exec
-- Deletes all repairs (and cascaded children) for a vehicle+system combination.
DELETE FROM repairs
WHERE vehicle_id = $1
  AND org_id = $2
  AND system = $3;
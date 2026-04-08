-- name: GetGarageVehicles :many
-- Returns one row per vehicle+system combination for the org.
-- The handler collapses these into nested vehicle → []systems structs.
-- Vehicles with no repairs appear once with NULL system columns.
SELECT
    v.id                                                              AS vehicle_id,
    v.year,
    v.make,
    v.model,
    v.trim,
    v.vin,
    v.mileage,
    v.unit_label,
    v.created_at                                                      AS vehicle_created_at,

    (
        SELECT MAX(r2.completed_at)
        FROM repairs r2
        WHERE r2.vehicle_id = v.id
          AND r2.status = 'completed'
    )                                                                 AS last_repair_at,

    (
        SELECT COUNT(*)
        FROM repairs r3
        WHERE r3.vehicle_id = v.id
          AND r3.status = 'in_progress'
    )                                                                 AS open_repair_count,

    r.system                                                          AS system_key,
    COALESCE(COUNT(r.id), 0)                                         AS system_repair_count,
    COALESCE(BOOL_OR(r.status = 'in_progress'), false)               AS system_has_open

FROM vehicles v
LEFT JOIN repairs r ON r.vehicle_id = v.id
WHERE v.org_id = $1
GROUP BY
    v.id, v.year, v.make, v.model, v.trim, v.vin,
    v.mileage, v.unit_label, v.created_at, r.system
ORDER BY v.created_at ASC, r.system ASC NULLS LAST;


-- name: GetGarageStats :one
-- Org-level aggregate stats for the summary header.
-- Single row — safe to cache at 60s TTL.
SELECT
    COUNT(DISTINCT v.id)                                              AS total_vehicles,

    COUNT(DISTINCT r.id) FILTER (WHERE r.status = 'in_progress')    AS open_repairs,

    COUNT(DISTINCT r.id) FILTER (
        WHERE r.completed_at >= NOW() - INTERVAL '30 days'
    )                                                                 AS repairs_30d,

    COALESCE(
        SUM(p.cost) FILTER (
            WHERE r.completed_at >= NOW() - INTERVAL '30 days'
              AND p.cost IS NOT NULL
        ),
        0
    )::NUMERIC(10,2)                                                  AS parts_spend_30d

FROM vehicles v
LEFT JOIN repairs r ON r.vehicle_id = v.id
LEFT JOIN parts   p ON p.repair_id  = r.id
WHERE v.org_id = $1;
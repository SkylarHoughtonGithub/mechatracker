-- ============================================================
-- mechatracker — Steps Queries
-- ============================================================

-- name: ListSteps :many
SELECT * FROM repair_steps
WHERE repair_id = $1
ORDER BY position ASC;

-- name: CreateStep :one
INSERT INTO repair_steps (repair_id, position, title, notes, torque_spec)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CompleteStep :one
UPDATE repair_steps SET
    completed    = true,
    completed_at = now()
WHERE id = $1
  AND repair_id = $2
RETURNING *;

-- name: UncompleteStep :one
UPDATE repair_steps SET
    completed    = false,
    completed_at = NULL
WHERE id = $1
  AND repair_id = $2
RETURNING *;

-- name: UpdateStep :one
UPDATE repair_steps SET
    title      = COALESCE(sqlc.narg('title'),      title),
    notes      = COALESCE(sqlc.narg('notes'),      notes),
    torque_spec = COALESCE(sqlc.narg('torque_spec'), torque_spec),
    position   = COALESCE(sqlc.narg('position'),   position)
WHERE id = $1
  AND repair_id = $2
RETURNING *;

-- name: DeleteStep :exec
DELETE FROM repair_steps WHERE id = $1 AND repair_id = $2;

-- name: CountSteps :one
-- Used to recalculate progress_pct after a complete/uncomplete.
SELECT
    COUNT(*)                            AS total,
    COUNT(*) FILTER (WHERE completed)   AS completed
FROM repair_steps
WHERE repair_id = $1;

-- ============================================================
-- Parts Queries
-- ============================================================

-- name: ListParts :many
SELECT * FROM parts
WHERE repair_id = $1
ORDER BY created_at ASC;

-- name: CreatePart :one
INSERT INTO parts (repair_id, name, part_number, supplier, status, cost, notes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdatePartStatus :one
UPDATE parts SET status = @status::part_status
WHERE id = @id
  AND repair_id = @repair_id
RETURNING *;

-- name: UpdatePart :one
UPDATE parts SET
    name        = COALESCE(sqlc.narg('name'),        name),
    part_number = COALESCE(sqlc.narg('part_number'), part_number),
    supplier    = COALESCE(sqlc.narg('supplier'),    supplier),
    cost        = COALESCE(sqlc.narg('cost'),        cost),
    notes       = COALESCE(sqlc.narg('notes'),       notes)
WHERE id = $1
  AND repair_id = $2
RETURNING *;

-- name: DeletePart :exec
DELETE FROM parts WHERE id = $1 AND repair_id = $2;

-- name: SumPartsCost :one
SELECT COALESCE(SUM(cost), 0)::NUMERIC(10,2) AS total
FROM parts
WHERE repair_id = $1
  AND cost IS NOT NULL;

-- ============================================================
-- Notes Queries
-- ============================================================

-- name: ListNotes :many
SELECT n.*, u.name AS author_name
FROM repair_notes n
JOIN users u ON u.id = n.user_id
WHERE n.repair_id = $1
ORDER BY n.created_at ASC;

-- name: CreateNote :one
INSERT INTO repair_notes (repair_id, user_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeleteNote :exec
DELETE FROM repair_notes
WHERE id = $1
  AND user_id = $2;

-- ============================================================
-- User Queries
-- ============================================================

-- name: GetUserByGoogleSub :one
SELECT * FROM users
WHERE google_sub = $1
LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: CreateUser :one
-- Called from POST /orgs (new org) and POST /invites/{token}/accept.
-- The org must exist first (CreateOrg for Path A, invite.org_id for Path B).
INSERT INTO users (google_sub, org_id, role, email, email_verified, name, picture_url)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListOrgMembers :many
SELECT * FROM users
WHERE org_id = $1
ORDER BY name ASC;

-- name: UpdateMemberRole :one
UPDATE users SET role = @role::user_role
WHERE id = @id
  AND org_id = @org_id
RETURNING *;

-- ============================================================
-- Invite Queries
-- ============================================================

-- name: CreateInvite :one
-- Called from POST /orgs/invites. expires_at is now() + 24h, set by the handler.
INSERT INTO invites (org_id, email, role, token, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetInviteByToken :one
-- Returns the invite row regardless of status; handler decides whether it's
-- still valid (expired? consumed?). This keeps the "why can't I accept this?"
-- error message precise.
SELECT * FROM invites
WHERE token = $1
LIMIT 1;

-- name: ConsumeInvite :one
-- Idempotent-ish: only stamps consumed_at if it is still NULL. Returns the
-- updated row, or zero rows if the invite was already consumed (handler maps
-- that to a 409).
UPDATE invites SET consumed_at = now()
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING *;

-- name: ListPendingInvitesForOrg :many
SELECT * FROM invites
WHERE org_id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
ORDER BY created_at DESC;

-- name: DeleteInvite :exec
-- Owner/admin can rescind a pending invite before it's consumed.
DELETE FROM invites
WHERE id = $1
  AND org_id = $2
  AND consumed_at IS NULL;

-- ============================================================
-- Deleted Items Queries
-- ============================================================

-- name: InsertDeletedItem :one
INSERT INTO deleted_items (entity_type, entity_id, org_id, deleted_by, snapshot)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: PurgeExpiredDeletedItems :exec
-- Run by the nightly background worker.
DELETE FROM deleted_items WHERE purge_at < now();

-- name: ListResources :many
SELECT * FROM resources
WHERE repair_id = $1
ORDER BY saved_at ASC;

-- name: CreateOrg :one
-- Called from POST /orgs after Google sign-in. A personal org is created
-- first, then CreateUser is called with the new org's id.
INSERT INTO organizations (name, slug, plan, personal)
VALUES ($1, $2, 'personal', true)
RETURNING *;

-- name: GetOrgBySlug :one
SELECT * FROM organizations WHERE slug = $1 LIMIT 1;

-- name: GetOrgByID :one
SELECT * FROM organizations WHERE id = $1 LIMIT 1;
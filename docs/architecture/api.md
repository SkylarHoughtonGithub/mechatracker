---
layout: default
title: API Architecture
parent: Architecture
---

## Overview

The API is a Go service using the Gin HTTP framework. It follows a layered structure: router → middleware → handler → store. Handlers are thin — they parse input, call one or more store methods, and write a response. Business logic that spans multiple store calls lives in a service layer (not yet built).

---

## Structure

```
api/
├── cmd/api/main.go          # entry point — wires config, DB pool, router
├── internal/
│   ├── router/router.go     # route registration
│   ├── middleware/          # auth, logging, recovery, request ID
│   ├── handler/             # one file per domain (vehicle.go, repair.go, ...)
│   ├── store/               # sqlc-generated — do not hand-edit
│   ├── mocks/               # mockery-generated — do not hand-edit
│   ├── apierr/              # canonical error response helpers
│   └── config/              # env-driven config, loaded once at startup
├── migrations/              # golang-migrate SQL files — schema source of truth
├── queries/                 # hand-written SQL → sqlc generates store/
└── sqlc.yaml                # sqlc config
```

---

## Request lifecycle

```
client
  → RequestID middleware    (assigns X-Request-ID)
  → Recovery middleware     (catches panics, returns structured 500)
  → Logger middleware       (structured log line after handler returns)
  → Auth middleware         (verifies JWT, sets user/org context)
  → handler func            (parses input, calls store, writes response)
```

Every authenticated route goes through all five layers. The `/health` and `/ready` endpoints skip Auth.

---

## Handlers

### Conventions

Handlers receive a `*gin.Context` and return nothing. All responses go through `c.JSON` or an `apierr.*` helper — never `c.String` or raw `http.ResponseWriter`.

```go
func (h *VehicleHandler) Get(c *gin.Context) {
    orgID, ok := orgIDFromCtx(c)
    if !ok {
        return  // apierr helper already wrote the response
    }
    vehicleID, ok := parseUUIDParam(c, "vehicleID")
    if !ok {
        return
    }

    vehicle, err := h.q.GetVehicle(c.Request.Context(), store.GetVehicleParams{
        ID:    vehicleID,
        OrgID: orgID,
    })
    if err != nil {
        apierr.NotFound(c, "Vehicle not found.")
        return
    }

    c.JSON(http.StatusOK, vehicle)
}
```

### Handler construction

Each handler struct holds a `store.Querier` and a `*zap.Logger`. The Querier is the sqlc-generated interface — using the interface (not the concrete `*store.Queries` type) keeps handlers testable with `MockQuerier`.

```go
type VehicleHandler struct {
    q   store.Querier
    log *zap.Logger
}

func NewVehicleHandler(q store.Querier, log *zap.Logger) *VehicleHandler {
    return &VehicleHandler{q: q, log: log}
}
```

Handlers are constructed in `router/router.go` and passed into the route registration functions.

### Input validation

Request bodies are decoded with `c.ShouldBindJSON`. Validation rules live on the request struct using Gin's `binding` struct tags. If binding fails, respond with `apierr.BadRequest` — never return a raw binding error to the client.

```go
type createVehicleReq struct {
    Year  int16  `json:"year"  binding:"required,min=1900,max=2100"`
    Make  string `json:"make"  binding:"required,max=100"`
    Model string `json:"model" binding:"required,max=100"`
}

var req createVehicleReq
if err := c.ShouldBindJSON(&req); err != nil {
    apierr.BadRequest(c, err.Error())
    return
}
```

URL parameters are parsed with the `parseUUIDParam` helper in `handler/vehicle.go`. Add equivalent helpers to other handler files as needed, or promote them to a shared `handler/helpers.go` once there are three or more handlers using them.

---

## Error responses

All errors use the `apierr` package. Every error response has the same JSON shape:

```json
{
  "error": {
    "code":    "VEHICLE_NOT_FOUND",
    "message": "Vehicle not found.",
    "details": {}
  }
}
```

| Helper | Status | Code |
|---|---|---|
| `apierr.BadRequest` | 400 | `BAD_REQUEST` |
| `apierr.Unauthorized` | 401 | `UNAUTHORIZED` |
| `apierr.Forbidden` | 403 | `FORBIDDEN` |
| `apierr.NotFound` | 404 | `NOT_FOUND` |
| `apierr.Validation` | 422 | `VALIDATION_ERROR` |
| `apierr.Conflict` | 409 | `CONFLICT` |
| `apierr.Internal` | 500 | `INTERNAL_ERROR` |

`apierr.Internal` logs the error at `Error` level and returns a generic message — never expose internal error details to the client.

---

## Auth and context

The `Auth` middleware in `internal/middleware/auth.go` verifies the Clerk JWT and sets three context keys:

| Key | Value |
|---|---|
| `KeyUserID` | Clerk subject string (`user_2abc...`) in production; `DevUserID` UUID in dev bypass |
| `KeyOrgID` | Clerk org ID string in production; `DevOrgID` UUID in dev bypass |
| `KeyOrgRole` | `"org:admin"` \| `"org:member"` etc. |

### Reading context in handlers

Two helpers in `handler/vehicle.go` read these keys. Promote them to `handler/helpers.go` when the second handler is added:

**`orgIDFromCtx`** — parses `KeyOrgID` as a UUID. Aborts with 401 if missing or malformed. Every handler that queries the DB calls this first — it is the org-scoping guard.

**`callerIDFromCtx`** — reads `KeyUserID` and parses it as a UUID. Works today because dev bypass sets `KeyUserID` to `DevUserID` (a real UUID). **This will break when real Clerk auth is wired in** — the Clerk subject string is not a UUID. The fix is a user-lookup middleware that calls `GetUserByClerkID`, stores the internal UUID under a new key (e.g. `KeyInternalUserID`), and has `callerIDFromCtx` read that key instead.

### Dev bypass

When `CLERK_DEV_BYPASS=true`, the middleware skips JWT verification and injects:

```go
c.Set(KeyUserID, DevUserID)   // "00000000-0000-0000-0000-000000000002"
c.Set(KeyOrgID,  DevOrgID)    // "00000000-0000-0000-0000-000000000003"
c.Set(KeyOrgRole, "org:admin")
```

These match the dev seed records in `migrations/000002_dev_seeds.up.sql`. See `local_dev.md` for details.

### Clerk JWT verification status

The JWKS fetch and RSA key parsing in `auth.go` are partially implemented. The cache and HTTP fetch work correctly. The `rsaPEMFromJWK` function — which converts a raw JWK `n`/`e` to an RSA public key — is a stub that panics. It needs to be replaced with `github.com/lestrrat-go/jwx/v2/jwk` before real Clerk auth can be used.

---

## Store layer

The store layer is generated by sqlc from `queries/*.sql` against the schema in `migrations/`. Do not hand-edit files in `internal/store/` — they are overwritten on every `task generate:sqlc`.

### Querier interface

sqlc generates a `Querier` interface alongside the concrete `*Queries` type. Handlers always depend on `Querier`, not `*Queries`, so tests can inject `MockQuerier` without a real database.

### Org scoping

Every query that touches user data includes `org_id` in the `WHERE` clause. This is enforced at the query level, not the handler level — a handler cannot accidentally return another org's data by forgetting a filter. Example:

```sql
-- name: GetVehicle :one
SELECT * FROM vehicles
WHERE id = $1
  AND org_id = $2
```

If a vehicle exists but belongs to a different org, the query returns no rows and the handler returns 404 — indistinguishable from a vehicle that doesn't exist at all.

### Deleted items

Hard deletes write a JSON snapshot of the deleted row to the `deleted_items` table before returning. The background worker purges rows where `purge_at < now()` nightly (30-day retention). This is implemented in `VehicleHandler.Delete` and should be replicated in any handler that deletes a top-level entity (vehicles, repairs).

---

## Implemented endpoints

| Method | Path | Handler | Status |
|---|---|---|---|
| GET | `/health` | `handleHealth` | done |
| GET | `/ready` | `handleReady` | done |
| GET | `/v1/vehicles` | `VehicleHandler.List` | done |
| POST | `/v1/vehicles` | `VehicleHandler.Create` | done |
| GET | `/v1/vehicles/:vehicleID` | `VehicleHandler.Get` | done |
| PATCH | `/v1/vehicles/:vehicleID` | `VehicleHandler.Update` | done |
| DELETE | `/v1/vehicles/:vehicleID` | `VehicleHandler.Delete` | done |
| GET | `/v1/repairs` | — | stub |
| POST | `/v1/repairs` | — | stub |
| GET | `/v1/repairs/:repairID` | — | stub |
| PATCH | `/v1/repairs/:repairID` | — | stub |
| POST | `/v1/repairs/:repairID/steps/:stepID/complete` | — | stub |
| POST | `/v1/repairs/:repairID/notes` | — | stub |
| POST | `/v1/repairs/:repairID/photos` | — | stub |
| GET | `/v1/garage` | — | stub |
| GET | `/v1/org/members` | — | stub |
| PATCH | `/v1/org/members/:userID/role` | — | stub |

---

## Testing

Handler tests use `MockQuerier` from `internal/mocks/` to avoid needing a real database. A typical test:

```go
func TestVehicleHandler_Get(t *testing.T) {
    q := mocks.NewMockQuerier(t)

    q.EXPECT().GetVehicle(mock.Anything, store.GetVehicleParams{
        ID:    vehicleID,
        OrgID: orgID,
    }).Return(store.Vehicle{
        ID:    vehicleID,
        Make:  "Ford",
        Model: "F-150",
        Year:  2018,
    }, nil)

    h := handler.NewVehicleHandler(q, zap.NewNop())
    // fire test HTTP request, assert response
}
```

Run all tests with `task api:test`.
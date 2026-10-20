# mechatracker — Skylab golden-path alignment

Oct 20, 2026 · @Skylar

> First real workload to flow through the Skylab Go service template (template → tenants-pr → KubeVela Application → cluster). See `skylab-backstage/plan.md` for the surrounding rework.

## North star

mechatracker is the proof point for golden-path items #1 and #2: a Backstage template run opens a PR against `platform/tenants`, Argo CD syncs the resulting KubeVela Application, and the entity page shows live traces, dashboard and SLO. mechatracker is the only workload riding the golden path (chirpy was dropped — Boot.dev-tutorial fingerprint, not worth the portfolio tax). Because it already needs Postgres, mechatracker also doubles as the database-template proof once that template lands: Phase 2 swaps the hand-wired `postgresql.enabled` in the Helm chart for a Crossplane-provisioned Postgres XR declared through the Backstage database template.

The Helm chart in `infra/helm/mechatracker/` retires once the KubeVela path is proven — one deployment story, not two.

Before any of that lands, Clerk comes out. Clerk's hosted org/membership model was the thing blocking deployment — ripping it keeps the "login with Google" experience, moves orgs/memberships into mechatracker's own DB, and drops a managed-service dependency.

Only the Go API in `api/` is in scope for the Go service template. `web/` ships separately (second component in the same KubeVela Application, or a future Next.js template).

## Current state (as of 2026-10-20)

- Go API in `api/` on :8080: gin router (with `gin-contrib/cors`), pgxpool, zap logging, graceful shutdown, `apierr` envelope for error responses. Entry at `api/cmd/api/`.
- `GET /health` in `api/internal/router/router.go:44`.
- `api/Dockerfile`: alpine multi-stage, buildx cross-compile, Go 1.25, port 8080.
- Taskfile with ~40 targets (`api:image:push`, `helm:upgrade`, db migrations).
- Postgres via pgxpool; golang-migrate migrations in `api/migrations/`.
- **Clerk** is embedded across: `internal/middleware/auth.go`, `internal/middleware/user_lookup.go`, `internal/handler/webhook.go` (svix), `internal/store/models.go`, `migrations/000001_init.up.sql` + `000002_dev_seeds.up.sql`, `queries/misc.sql`, `openapi/*`, and `web/` sign-in UI (`NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`).
- Helm chart at `infra/helm/mechatracker/`.
- k3d dev config at `infra/k3d/`.
- No tests in `api/`.
- No OTel deps or wiring.
- No `.gitea/workflows/` or `.github/workflows/`.
- No root `catalog-info.yaml`, no mkdocs TechDocs (`docs/` exists but is bare Jekyll scaffolding).

---

## Phase 0 — Clerk → Google auth

Owner's own tables, Google for identity. Multi-tenancy preserved (users belong to orgs via internal memberships), but Google is just the sign-in — mechatracker owns the user/org/membership model.

### Target shape

- **Identity:** Google OAuth 2.0 + ID-token verification. Library: `google.golang.org/api/idtoken` for server-side ID-token verification; `golang.org/x/oauth2/google` for the OAuth code flow if we need offline access (probably not — ID token on sign-in is enough).
- **Sessions:** signed JWT in an httpOnly, SameSite=Lax cookie. HS256 with a secret from env (`SESSION_SIGNING_KEY`). Short expiry (e.g. 12h) with silent refresh via a re-check against Google on expiry.
- **Users and orgs:** single-org-per-user (matches existing `data_model.md`). `users` table keeps `org_id` NOT NULL and `role` as a per-user column; the only auth-related schema change is `clerk_id` → `google_sub` (unique, not null), plus `email_verified`, `name`, `picture_url` for display. `role ∈ {owner, admin, member}` (renamed from the current `owner/manager/technician` enum; the UI can still present these as "Owner / Manager / Technician" in a shop context).
- **First sign-in with no invite:** user lands on a "create an org or paste an invite" screen. No auto-created org. Users with no `users` row yet can hit `/auth/me` (returns `{user: null, needs_onboarding: true}`), but every tenant-scoped route rejects them until they create or join one.
- **Invites:** `invites(org_id, email, role, token, expires_at, consumed_at)`. Single-use (reject if `consumed_at IS NOT NULL`), 24h expiry (`expires_at = now() + 24h` at creation; reject if past). Accept flow: signed-in Google account clicks link, server checks token valid + unexpired + unconsumed + Google email matches invite email + no existing `users` row for that `google_sub` (single-org constraint), inserts the `users` row scoped to the inviting org with the invite's role, stamps `consumed_at`. An existing user with another org clicking an invite link is rejected with "This Google account is already in a workspace — sign out and use a different Google account to accept."
- **Webhooks:** gone. svix dependency removed. Nothing external pushes user state into us anymore. **ngrok goes with it** — Google OAuth only requires outbound HTTPS from the server (JWKS fetch for ID-token verification), and the browser redirect works against `http://localhost` without a tunnel.
- **Dev bypass:** keeps the same shape as today's `CLERK_DEV_BYPASS`, renamed to `DEV_AUTH_BYPASS`. When set (and `ENV=development`), middleware skips token verification and injects the dev seed user + org. See `local_dev.md`.

### Rip-out steps

- [ ] Delete `internal/middleware/auth.go` Clerk code; replace with a middleware that validates the session JWT cookie or falls back to the Google ID token in `Authorization: Bearer`.
- [ ] Delete `internal/middleware/user_lookup.go` Clerk-specific logic; new version looks up the internal `users` row by `google_sub` from the session JWT claims.
- [ ] Delete `internal/handler/webhook.go` and the Clerk route in `internal/router/router.go`.
- [ ] Add handlers: `POST /auth/google` (verify ID token, upsert `users` row *only* if invite consumed or org created in same request; otherwise just mint a session and record the Google claims in a short-lived cookie for the onboarding step), `POST /auth/logout` (clear cookie), `GET /auth/me` (returns `{user: {...}, org: {...}}` once onboarded, or `{user: null, needs_onboarding: true, google_claims: {email, name}}` for a fresh sign-in), `POST /orgs` (create org + `users` row with role=owner for the signed-in Google claims; rejects if a user already exists for this `google_sub`), `POST /orgs/invites` (owner/admin only; 24h, single-use), `POST /invites/{token}/accept` (consume invite + create `users` row scoped to the inviting org; rejects if the Google account already has a `users` row).
- [ ] Drop `svix` and Clerk Go SDK deps from `go.mod`; `go mod tidy`.
- [ ] **Hard migration** — edit `migrations/000001_init.up.sql` in place rather than ship a 000003: (a) `users.clerk_id` becomes `google_sub TEXT NOT NULL UNIQUE`, add `email_verified BOOLEAN NOT NULL DEFAULT false` + `picture_url TEXT`, (b) rename the `user_role` enum values to `('owner', 'admin', 'member')`, (c) add `invites(id, org_id, email, role, token, expires_at, consumed_at, invited_by, created_at)`. Edit `migrations/000002_dev_seeds.up.sql` to use `google_sub`. No live data to protect; devs recreate local DBs from scratch (`task db:nuke && task db:migrate`).
- [ ] Regenerate sqlc from the updated `queries/` after the schema moves.
- [ ] Update `openapi/openapi3.yaml` + regen `swagger.yaml` / `swagger.json`: remove Clerk security scheme, add bearer + cookie session schemes, replace webhook paths with the three new auth routes.
- [ ] `web/`: remove Clerk SDK and `<ClerkProvider>`; replace sign-in page with Google sign-in button (Google Identity Services JS; one-tap optional). Call `POST /auth/google` with the ID token. Replace `useUser()` / `useOrganization()` call sites with calls to `/auth/me`. Add a "create or join" onboarding page shown when `/auth/me` returns `needs_onboarding: true`, with a form to create a new org and a field to paste/accept an invite token. Env: drop `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY`, add `NEXT_PUBLIC_GOOGLE_CLIENT_ID`.
- [ ] `Taskfile.yaml`: delete `WEBHOOK_DOMAIN` var, `dev:webhook-tunnel` task, ngrok steps from `dev:tunnels`, and `ngrok` from the brew-install list in `setup`.
- [ ] `docs/development/local_dev.md`: delete the Clerk + ngrok section (lines ~150-220); dev flow is now `task dev` + a browser Google sign-in, no tunnel.
- [ ] Reseed dev data via `000002_dev_seeds.up.sql`: pick one Google account's `sub` as the dev user; comment the file to say how to swap it.
- [ ] Smoke-test locally: sign in with Google → cookie set → `/auth/me` returns user + first org → create a second org → invite a second Google account → that account accepts.

### Config

- `GOOGLE_CLIENT_ID` (required at startup; the server also uses it as the ID-token audience).
- `GOOGLE_ALLOWED_HD` (optional; restrict to a Google Workspace hosted domain if we ever want that).
- `SESSION_SIGNING_KEY` (required; 32+ bytes).
- `SESSION_COOKIE_DOMAIN`, `SESSION_COOKIE_SECURE` (default true), `SESSION_TTL` (default 12h).

### Done when

- [ ] Clerk, svix, ngrok, and `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY` are gone from the repo (grep -r).
- [ ] A fresh user signs in with Google, lands on the "create or join" screen, creates an org, and is its owner.
- [ ] That owner invites a second Google account; the invite link works once, is rejected on second use, and is rejected after 24h.
- [ ] Nothing in `api/` requires an outbound call to Clerk at startup or runtime, and `task dev` works without any tunnel.

---

## Phase 1 — Golden-path alignment

What the Go service template emits, from `skylab-backstage/app/templates/go-service/skeleton/`:

- `Dockerfile`: Go 1.23 build → `gcr.io/distroless/static:nonroot`, CGO off, port 8080. (Template still pins 1.23; mechatracker stays on 1.25 — flag the template as a follow-up rather than downgrade.)
- `main.go`: OTLP gRPC tracer via `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc`, honors `OTEL_EXPORTER_OTLP_ENDPOINT`, `/healthz`, graceful shutdown on SIGTERM/SIGINT.
- `catalog-info.yaml` with annotations: `backstage.io/techdocs-ref`, `backstage.io/kubernetes-id`, `argocd/app-selector`, `grafana/dashboard-selector`, `grafana/alert-label-selector`.
- `mkdocs.yml` + `docs/index.md`.
- `.gitea/workflows/ci.yaml`: test → buildx push to GHCR → Trivy CRITICAL exit=1 → cosign keyless sign, SBOM + `provenance: mode=max`.
- `tenants-pr/applications/<name>.yaml`: one-line KubeVela Application, `type: go-service` ComponentDefinition. The ComponentDefinition bakes in OTel env, `skylab.io/system` labels and `/healthz` probes — manifest only carries the image.

### Gaps

**Must-have before Backstage can run it:**

- [ ] Rename `/health` → `/healthz` in `api/internal/router/router.go:44`.
- [ ] Add OTel SDK: `go.opentelemetry.io/otel`, `.../exporters/otlp/otlptrace/otlptracegrpc`, `.../sdk/trace`. Wire a tracer provider in `api/cmd/api/main.go` that honors `OTEL_EXPORTER_OTLP_ENDPOINT`. Instrument the gin router (`otelgin`).
- [ ] Switch `api/Dockerfile` runtime stage to `gcr.io/distroless/static:nonroot`. Verify no runtime CA/tzdata dependency; if certs are needed outbound (Google token verification does need CA roots), use `gcr.io/distroless/base:nonroot`. Keep the buildx cross-compile build stage.
- [ ] Add `.gitea/workflows/ci.yaml` modeled on the template's — adjust image target from `ghcr.io/platform/<name>` to the real GHCR owner, keep Trivy CRITICAL/exit=1 and cosign.
- [ ] Add at least one `*_test.go` so `task api:test` and the CI test job have real signal. Health handler + Google ID-token verification path are the natural starters.
- [ ] Add root `catalog-info.yaml` (Component kind, `skylab.io/system: mechatracker` label, the five annotations).
- [ ] Hand-write the first tenants-pr KubeVela Application in `platform/tenants`: `namespace: mechatracker`, `type: go-service`, `image: ghcr.io/<owner>/mechatracker:<sha>`.

**Should-have:**

- [ ] `mkdocs.yml` + minimal `docs/index.md` for TechDocs.
- [ ] Retire `infra/helm/mechatracker/` once the KubeVela path is proven.
- [ ] Add a second component to the KubeVela Application for `web/` once there's a frontend ComponentDefinition, or ship it via a future Next.js template.

**Out of scope for the golden-path proof:**

- Broad test coverage.
- Google OAuth consent-screen polish (brand, logo, verification).

---

## Phase 2 — Database-template proof

Lands after the Backstage database template exists (skylab-backstage plan.md, golden-path item #2) and after skylab-crossplane ships its `postgres` XRD (skylab-crossplane plan.md).

- [ ] Re-run mechatracker through the database template to add a declared `postgres` dependency to its KubeVela Application. The template emits the `postgres` component block and wires `DATABASE_URL` from the Crossplane-provisioned connection secret.
- [ ] Replace the in-cluster `postgres:17` StatefulSet in `infra/helm/mechatracker/` (and the k3d dev loop) with the same XR path. In dev, point the XR composition at a local Postgres instead of RDS.
- [ ] Decide how migrations run against a Crossplane-provisioned DB: init container, dedicated migrate Job, or app-startup. Current `golang-migrate` works either way; the Job pattern is cleanest and matches skylab-gitops Job conventions.
- [ ] Verify the Backstage entity page's resources tab shows both the deployment and the Postgres XR.

---

## Build order

Phase 0 (Clerk out) must finish before Phase 1 ships, because the image we build and sign has to be the Google-auth version.

1. **Phase 0:** schema migration + sqlc regen + new auth routes + middleware swap.
2. **Phase 0:** `web/` sign-in replacement, local smoke test.
3. **Phase 1:** OTel wiring in `api/cmd/api/main.go` + gin middleware (`otelgin`).
4. **Phase 1:** `/health` → `/healthz`.
5. **Phase 1:** Dockerfile runtime stage → distroless; verify image boots, hits Postgres, verifies a Google token.
6. **Phase 1:** First handler test(s).
7. **Phase 1:** `.gitea/workflows/ci.yaml`.
8. **Phase 1:** `catalog-info.yaml` + `mkdocs.yml` + `docs/index.md`.
9. **Phase 1:** Hand-write tenants-pr KubeVela Application; Argo CD syncs it; verify in cluster.
10. Once Backstage + Gitea are live, re-run via the template and confirm the end-to-end path.
11. Retire `infra/helm/mechatracker/`.

## Done when

- [ ] Clerk is gone; Google sign-in works end-to-end locally and in cluster.
- [ ] mechatracker image builds in CI, is cosign-signed with SBOM, pushed to GHCR.
- [ ] A KubeVela Application in `platform/tenants` (ultimately from a template-emitted PR) deploys mechatracker to the homelab cluster.
- [ ] The Backstage entity page shows live Argo CD, KubeVela and Grafana status.
- [ ] Grafana shows traces, a dashboard and an SLO for mechatracker.
- [ ] A database-template run adds a Crossplane-provisioned Postgres XR to mechatracker's Application; the StatefulSet is removed.
- [ ] `infra/helm/mechatracker/` is deleted.

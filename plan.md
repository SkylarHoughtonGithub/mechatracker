# mechatracker — Skylab golden-path alignment

Oct 21, 2026 · @Skylar

> First real workload to flow through the Skylab Go service template (template → tenants-pr → KubeVela Application → cluster). See `skylab-backstage/plan.md` for the surrounding rework.

## North star

mechatracker is the proof point for golden-path items #1 and #2: a Backstage template run opens a PR against `platform/tenants`, Argo CD syncs the resulting KubeVela Application, and the entity page shows live traces, dashboard and SLO. mechatracker is the only workload riding the golden path (chirpy was dropped — Boot.dev-tutorial fingerprint, not worth the portfolio tax). Because it already needs Postgres, mechatracker also doubles as the database-template proof once that template lands: Phase 2 swaps the in-cluster Postgres (currently a hand-wired StatefulSet in `skylab-gitops/workloads/mechatracker/`) for a Crossplane-provisioned Postgres XR declared through the Backstage database template.

The Helm chart in `infra/helm/mechatracker/` retires once the KubeVela path is proven on cluster — one deployment story, not two.

## Status (as of 2026-10-21)

- [x] **Phase 0 — Clerk → Google OAuth (backend, #79).** svix webhook + Clerk SDK + ngrok all gone. Google ID-token verification via `google.golang.org/api/idtoken`; HS256 session JWT in an httpOnly `mt_session` cookie; 6 new auth/onboarding routes (`POST /auth/google`, `POST /auth/logout`, `GET /auth/me`, `POST /orgs`, `POST /orgs/invites`, `POST /invites/:token/accept`); single-org-per-user shape preserved; `DEV_AUTH_BYPASS` for local dev. Schema hard-migrated in place (no 000003 — no live data): `users.clerk_id` → `google_sub` + `email_verified` + `picture_url`; `user_role` enum values renamed `owner`/`admin`/`member`; `invites` table added.
- [x] **Phase 1 — Golden-path alignment (API, #80).** OTel SDK + `otelgin` middleware (OTLP/gRPC, honors `OTEL_EXPORTER_OTLP_ENDPOINT`, non-fatal if unset). `/health` dropped; `/healthz` is the sole probe. Runtime stage → `gcr.io/distroless/base:nonroot`. Binary stamped via `-X main.version=${VERSION}` from a buildx ARG. `.github/workflows/api-ci.yaml` modeled on the go-service template (test → buildx push → Trivy CRITICAL exit=1 → cosign + SBOM + provenance=max). First test suite (session + middleware packages). Root `catalog-info.yaml` + `mkdocs.yml`.
- [x] **Phase 1 — Web Clerk → Google (frontend, #85).** `@clerk/nextjs` out, `@react-oauth/google` in. New `AuthProvider` owns `/auth/me` state. `/sign-in` with Google button; `/onboarding` for create workspace / accept invite; `/accept-invite/[token]` convenience redirect. `AppShell` gates the signed-in chrome by auth status. `middleware.ts` (replacing `proxy.ts`) is a cookie-presence check. API client uses `credentials: 'include'` + a Next.js rewrite at `/api/backend/*` for same-origin cookie flow.
- [x] **Deployed on dev cluster (`skylab-gitops` #53).** `workloads/mechatracker/application.yaml` is a KubeVela Application with the `go-service` component plus an interim `k8s-objects` component carrying a Postgres `17-alpine` StatefulSet + Service + Secret (static creds, 10Gi PVC). `DEV_AUTH_BYPASS=true` + `ENV=development` on the api so the pod comes up without real Google credentials. `dependsOn: [postgres]` on the api so the first `pool.Ping()` has a listener.

## Phase 2 — Database-template proof

Prerequisites all shipped:
- `skylab-crossplane` v0.4.0 published the Postgres XRD (`database.skylab.io/v1alpha1 XPostgres`, AWS RDS + GCP CloudSQL).
- `skylab-gitops` v0.2.0 of `kubevela-defs` shipped the `postgres` ComponentDefinition — emits the XR + a compose Job that derives `DATABASE_URL` into `db-url`.
- `skylab-backstage` v1.1.0 shipped the `database` golden-path template that emits a sibling `applications/<service>-db.yaml` via `type: postgres`, auto-merged against platform/tenants.

Cutover done hand-wired on the gitops side (`skylab-gitops` #59), ahead of a template run against a live Gitea:

- [x] Sibling `workloads/mechatracker-db/application.yaml` using `type: postgres`.
- [x] Delete the `postgres` k8s-objects block and its Secret from `workloads/mechatracker/application.yaml`.
- [x] `api.envFrom` points at `db-url` (the Secret the compose Job writes).

Still open:

- [ ] Fill in the two `REPLACE-ME` fields in `workloads/mechatracker-db/application.yaml` (AWS DB subnet group + security group IDs) once Terraform's network stack outputs them.
- [ ] Decide how migrations run against the Crossplane-provisioned DB: init container, dedicated migrate Job, or app-startup. A migrate Job tied to the `postgres` ComponentDefinition is the cleanest path and matches skylab-gitops Job conventions; defer until Phase 2 actually provisions on a real cluster.
- [ ] On an EC2/EKS or GKE cluster with wave 1 (Crossplane + providers) and real cloud credentials: verify the compose Job completes and the api pod comes up healthy against the provisioned DB.
- [ ] Verify the Backstage entity page's resources tab shows both the Deployment and the Postgres XR.
- [ ] Delete `infra/helm/mechatracker/` — the KubeVela path is now the only deployment story. Keep until a real cluster proves mechatracker-db end-to-end.
- [ ] Replace the dev-only inline password in `workloads/mechatracker-db/application.yaml` with an ESO-backed Secret once ClusterSecretStores are wired.

## Release strategy

Fleet-wide decision: release-please + Conventional Commits (see `skylab-gitops/plan.md`). To wire here:

- [ ] Add `release-please-config.json` with `release-type: go` for the root package (`.`), `package-name: mechatracker-api`.
- [ ] Add `.release-please-manifest.json` seeded with the current version.
- [ ] Add `.github/workflows/release-please.yaml` running on push to `main` with `contents: write` + `pull-requests: write`.
- [ ] Thread the release tag into the image build: the Dockerfile already accepts a `VERSION` ARG via `-X main.version=${VERSION}`; the CI workflow picks up `${{ github.ref_name }}` on tag pushes.
- [ ] PR titles use `feat(api):` / `feat(web):` / `fix:` / `chore:` etc. The `api/` scope keeps the door open for a second package once `web/` wires in its own release cadence.

## Follow-ups (not blocking)

- `@Summary` + response-type swag annotations on the 6 new auth/onboarding handlers so the OpenAPI spec documents them. The handlers work; the spec just doesn't describe them yet.
- Role-based UI gating in `web/` — the `AuthProvider` exposes `user.role`; destructive operations should check it rather than relying on 403s from the API.
- Member management UI (`web/`): list members, send/revoke invites. Backend has `GET /v1/org/members` + `POST /v1/org/invites` + `DELETE /v1/org/invites/{id}`.
- First test suite for `web/` (Playwright or Vitest + Testing Library).
- Expand `api/` test coverage past the current session + middleware pair.

## Done when

- [x] Clerk is gone; Google sign-in works end-to-end locally with `DEV_AUTH_BYPASS=true`.
- [x] mechatracker image builds in CI, is cosign-signed with SBOM, pushed to GHCR.
- [x] A KubeVela Application in `skylab-gitops/workloads/mechatracker/` (hand-written, matches the shape the template will emit) deploys mechatracker to the homelab cluster.
- [ ] On an EC2 cluster with a real `GOOGLE_CLIENT_ID` + `SESSION_SIGNING_KEY`: sign-in → onboarding → in-app flow works end-to-end against the deployed API.
- [ ] The Backstage entity page shows live Argo CD, KubeVela and Grafana status (requires Backstage pointed at the cluster + Grafana datasources provisioned).
- [ ] Grafana shows traces, a dashboard and an SLO for mechatracker (traces land once `OTEL_EXPORTER_OTLP_ENDPOINT` points at a reachable collector; dashboard + SLO are manual provisioning until a trait definition ships).
- [ ] A database-template run adds a Crossplane-provisioned Postgres XR to mechatracker's Application; the hand-wired Postgres block and `infra/helm/mechatracker/` both disappear in the same PR.

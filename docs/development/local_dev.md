---
layout: default
title: Local Development
parent: Development
---

## Overview

The local dev stack runs the full mechatracker API and its dependencies on a developer's machine using k3d (k3s in Docker) and Helm. The goal is a single command (`task dev`) that builds, deploys, and verifies the stack after first-time setup.

---

## Setup

### Prerequisites

Run once on a fresh machine:

```sh
./bootstrap.sh   # installs Homebrew and task
task setup       # installs all tools, creates cluster, installs Helm chart
```

`bootstrap.sh` installs only Homebrew and `task`. Everything else — Go, Node, k3d, kubectl, helm, sqlc, golang-migrate, docker-buildx, mockery, swag — is installed by `task setup:tools`.

After `task setup:tools`, ensure Go binaries are on your PATH. If `mockery` or `swag` are not found after installing:

```sh
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

### Inner loop

```sh
task dev          # build image → deploy → wait for postgres
task db:forward   # port-forward postgres to localhost:5432 (separate terminal)
task db:shell     # open a psql shell on the postgres instance
task db:migrate   # run pending migrations
task dev:logs     # tail API logs
```

### Code generation

```sh
task generate           # run all generators (sqlc + mockery)
task generate:sqlc      # regenerate internal/store/ from queries/*.sql
task generate:mocks     # regenerate internal/mocks/ from store.Querier interface
```

`generate:openapi` and `generate:ts` are commented out until Phase 3 (web app). They require `swag` annotations on handlers and a Next.js `shared/` package to exist.

Run `task generate` after any change to `queries/*.sql` or `migrations/*.up.sql`. Commit the generated files — CI validates they are up to date on every PR.

---

## Stack

| Component | Local | Production |
|---|---|---|
| Kubernetes | k3d (k3s in Docker) | EKS or GKE, provisioned by [skylab-crossplane](https://github.com/SkylarHoughtonGithub/skylab-crossplane) `KubernetesCluster` XRD |
| Container registry | Local k3d registry | GitHub Container Registry (GHCR) |
| Database | Official `postgres:17` StatefulSet | Crossplane-managed Postgres (`Postgres` XRD → RDS or CloudSQL) |
| Auth | Dev bypass (`DEV_AUTH_BYPASS=true`) | Google ID-token verification + session cookie |
| File storage | Not running | Crossplane-managed bucket (`Bucket` XRD → S3 or GCS) |

---

## Decisions and known limitations

### macOS only

`bootstrap.sh` assumes Homebrew and macOS. Linux workstations are not currently supported. When this becomes a need, the right fix is a Devcontainer or Nix flake that works identically on both platforms rather than extending the shell script.

### k3d registry networking

The local image registry involves two different addresses for the same container depending on which network context is doing the talking:

| Context | Address | Why |
|---|---|---|
| buildx pushing | `localhost:5050` | buildx runs on the host network; `localhost` resolves correctly |
| k3d nodes pulling | `mechatracker-registry:5050` | nodes are Docker containers; `localhost` would mean the node itself |

k3d wires up the pull address automatically via containerd mirror config when the registry is declared in `infra/k3d/config.yaml` using the `registries.create` block. Creating the registry separately with `k3d registry create` and then `--registry-use` does not wire this up reliably.

The registry name does **not** get a `k3d-` prefix when created via the config file's `registries.create` block, unlike CLI-created registries. The name in the config is used as-is.

### buildx and QEMU

Multi-platform builds (`linux/amd64,linux/arm64`) are unstable locally on Apple Silicon because QEMU — which emulates amd64 — crashes under the load of compiling Go. Local dev builds target `linux/arm64` only.

The full multi-platform build happens in CI on amd64 runners, where arm64 cross-compilation is handled natively by Go (`GOARCH=arm64`) without QEMU. See the Dockerfile — `--platform=$BUILDPLATFORM` on the builder stage pins the Go compiler to the host architecture; `GOOS`/`GOARCH` args handle cross-compilation for the output binary.

Production images pushed to GHCR are multi-arch manifest lists covering both platforms.

### Bitnami PostgreSQL

The Bitnami Helm chart for PostgreSQL was the original dependency. In August 2025, Broadcom (which acquired Bitnami) removed all free images from `docker.io/bitnami`. The chart was replaced with a plain Kubernetes StatefulSet using the official `postgres:17` image from Docker Hub, which is maintained by the PostgreSQL project and has no licensing concerns.

The production database is Crossplane-managed (via the `Postgres` XRD) and does not use this StatefulSet at all — `postgresql.enabled` defaults to `false` in `values.yaml`.

### Migrations in local dev

`task db:migrate` runs `golang-migrate` against `DATABASE_URL` from `api/.env`. It requires a direct connection to Postgres on `localhost:5432`, which means `task db:forward` must be running in a separate terminal first.

Migrations are not run automatically as part of `task dev` because the migrate binary runs on the host and cannot reach the in-cluster database without the port-forward. This is a known friction point. Options for the future:

- Run migrate as an init container in the API pod
- Add a dedicated migrate Job to the Helm chart
- Accept the manual port-forward step (current approach)

Migration files live in `api/migrations/` and use the naming convention `000N_description.up.sql` / `000N_description.down.sql`. Create new migrations with:

```sh
task db:new NAME=add_something
```

### Dev seeds

`000002_dev_seeds.up.sql` inserts a dev org and dev user with stable sentinel UUIDs:

| Entity | UUID |
|---|---|
| Dev org | `00000000-0000-0000-0000-000000000003` |
| Dev user | `00000000-0000-0000-0000-000000000002` |

These match the `DevOrgID` and `DevUserID` constants in `internal/middleware/auth.go` and are injected into every request when `DEV_AUTH_BYPASS=true`. This means `GET /v1/vehicles` returns vehicles scoped to the dev org without any Google OAuth setup.

### Auth bypass

`DEV_AUTH_BYPASS=true` in `values.dev.yaml` skips Google ID-token verification entirely. All requests are treated as `DevUserID` / `DevOrgID` with role `owner`. The config validator prevents this flag from being set outside `ENV=development`.

### Signing in with real Google (optional)

If you want to exercise the real auth flow locally — Google button, ID-token verification, session cookie, invite flow — unset `DEV_AUTH_BYPASS` and provide a Google OAuth 2.0 client ID. Google OAuth accepts `http://localhost:3000` as a redirect URI; no tunnel is required.

1. In Google Cloud Console, create an OAuth 2.0 Client ID of type "Web application" with `http://localhost:3000` as both the authorized origin and the authorized redirect URI (also add your dev subdomain if you use one).
2. Set in `values.dev.local.yaml`: `GOOGLE_CLIENT_ID`, `SESSION_SIGNING_KEY` (any 32-byte random string), and `NEXT_PUBLIC_GOOGLE_CLIENT_ID` for the web pod.
3. Browse to `http://localhost:3000`, sign in with Google; a brand-new account lands on the "create org or paste invite" screen.

### sqlc and mockery

`sqlc.yaml` lives in `api/` and is configured to read schema from `migrations/001_init.up` and queries from `queries/`. Output goes to `internal/store/`.

`.mockery.yaml` lives in `api/` and generates `MockQuerier` into `internal/mocks/mock_Querier.go` from the `store.Querier` interface. Use `MockQuerier` in handler unit tests to avoid needing a real database connection.

---

## Daily dev workflow

Port assignments:
- `localhost:8080` — API (matches `NEXT_PUBLIC_API_URL`)
- `localhost:3000` — Web (browse here)

Start the port-forwards:

```sh
task dev:tunnels
```

This runs in parallel:
- `kubectl port-forward svc/mechatracker-api 8080:8080`
- `kubectl port-forward svc/mechatracker-web 3000:3000`

Browse to `http://localhost:3000`. With `DEV_AUTH_BYPASS=true` (default in `values.dev.yaml`) you are automatically signed in as the dev user. With real Google OAuth configured (see "Signing in with real Google" above) the sign-in button mints a session cookie against your Google account.

There is no tunnel step — removing Clerk removed the only reason a public URL was needed in dev. The API's only outbound call for auth is Google's JWKS endpoint, which doesn't need inbound connectivity.

---

### Troubleshooting

**CORS errors in browser** — The web port-forward is conflicting with the API on 8080. Make sure web is forwarded to 3000 and API to 8080, and browse to `localhost:3000`.

**401 on tenant-scoped routes after Google sign-in** — The signed-in Google account has no `users` row yet. Hit `GET /auth/me` and confirm it returns `needs_onboarding: true`; complete the create-or-join flow before accessing other routes.

**`idtoken: audience mismatch`** — `GOOGLE_CLIENT_ID` in the pod doesn't match the OAuth client the web app is using. Ensure both the API and the web pod are pointing at the same client ID.
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
| Kubernetes | k3d (k3s in Docker) | DigitalOcean Droplets |
| Container registry | Local k3d registry | DigitalOcean Container Registry |
| Database | Official `postgres:17` StatefulSet | DigitalOcean Managed PostgreSQL |
| Auth | Dev bypass (`CLERK_DEV_BYPASS=true`) | Clerk JWT verification |
| File storage | Not running | DigitalOcean Spaces |

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

Production images pushed to DOCR are multi-arch manifest lists covering both platforms.

### Bitnami PostgreSQL

The Bitnami Helm chart for PostgreSQL was the original dependency. In August 2025, Broadcom (which acquired Bitnami) removed all free images from `docker.io/bitnami`. The chart was replaced with a plain Kubernetes StatefulSet using the official `postgres:17` image from Docker Hub, which is maintained by the PostgreSQL project and has no licensing concerns.

The production database is DigitalOcean Managed PostgreSQL and does not use this StatefulSet at all — `postgresql.enabled` defaults to `false` in `values.yaml`.

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

These match the `DevOrgID` and `DevUserID` constants in `internal/middleware/auth.go` and are injected into every request when `CLERK_DEV_BYPASS=true`. This means `GET /v1/vehicles` returns vehicles scoped to the dev org without any Clerk setup.

### Auth bypass

`CLERK_DEV_BYPASS=true` in `values.dev.yaml` skips JWT verification entirely. All requests are treated as `DevUserID` / `DevOrgID` with role `org:admin`. The config validator prevents this flag from being set outside `ENV=development`.

Full Clerk JWT verification (`rsaPEMFromJWK` in `middleware/auth.go`) is not yet implemented — it currently panics if called. This is intentional: the bypass covers all local dev needs and real Clerk auth will be wired in before staging is stood up. See `auth.md` for the current status.

### sqlc and mockery

`sqlc.yaml` lives in `api/` and is configured to read schema from `migrations/001_init.up` and queries from `queries/`. Output goes to `internal/store/`.

`.mockery.yaml` lives in `api/` and generates `MockQuerier` into `internal/mocks/mock_Querier.go` from the `store.Querier` interface. Use `MockQuerier` in handler unit tests to avoid needing a real database connection.

---

## Webhook tunneling (Clerk user.created)

The Clerk webhook fires on real user signups and provisions the org + user row in your database. For this to work locally, Clerk needs a public HTTPS URL that routes to your local API.

All services run as ClusterIP inside k8s, so nothing is exposed outside the cluster by default. The full chain is:

```
Clerk → ngrok → kubectl port-forward → mechatracker-api pod
```

### One-time setup

**1. Install ngrok**
```sh
brew install ngrok
```

**2. Authenticate**

Sign up at [ngrok.com](https://ngrok.com), then:
```sh
ngrok config add-authtoken <your-token>
```

**3. Set the webhook URL in Clerk — once**

Clerk Dashboard → Webhooks → Add Endpoint:
```
https://<your-ngrok-subdomain>.ngrok-free.app/webhooks/clerk
```
Subscribe to: `user.created`

Copy the **Signing Secret** (`whsec_...`) into:
- `api/.env` → `CLERK_WEBHOOK_SECRET=whsec_...`
- `infra/helm/mechatracker/values.dev.local.yaml` → `CLERK_WEBHOOK_SECRET: "whsec_..."`

The ngrok free tier generates a new subdomain each session, so you'll need to update the Clerk URL each time unless you upgrade to a paid plan with a reserved domain.

---

### Daily dev workflow

Port assignments:
- `localhost:8080` — API (matches `NEXT_PUBLIC_API_URL`, ngrok tunnels here)
- `localhost:3000` — Web (browse here)

When testing signup flows, use the tunnel task which starts everything together:

```sh
task dev:tunnels
```

This runs in parallel:
- `kubectl port-forward svc/mechatracker-api 8080:8080`
- `kubectl port-forward svc/mechatracker-web 3000:3000`
- `ngrok http 8080`

Browse to `http://localhost:3000`. After ngrok starts, copy the forwarding URL from its output and update the Clerk webhook endpoint URL.

The tunnel only needs to be running when testing real signups. For normal development with your existing seeded account, you don't need ngrok — just the port-forwards.

---

### Testing the webhook

Most realistic test: run `task dev:tunnels`, update the Clerk webhook URL with the new ngrok URL, then create a brand new account through the sign-up flow.

To test provisioning logic without a real signup, use the Clerk dashboard → Webhooks → **Send Test Event** → `user.created` while the tunnel is running.

---

### Troubleshooting

**404 from ngrok** — Check the Clerk webhook URL has the correct domain (`.ngrok-free.app` not `.ngrok-free.dev`) and includes the full path `/webhooks/clerk`.

**CORS errors in browser** — The web port-forward is conflicting with the API on 8080. Make sure web is forwarded to 3000 and API to 8080, and browse to `localhost:3000`.

**401 on /v1/garage after signup** — The webhook didn't fire or failed. Check ngrok output for the POST to `/webhooks/clerk` and look at API logs with `kubectl logs -n mechatracker-dev deploy/api`.
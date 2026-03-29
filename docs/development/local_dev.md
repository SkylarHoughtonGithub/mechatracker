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

`bootstrap.sh` installs only Homebrew and `task`. Everything else — Go, Node, k3d, kubectl, helm, sqlc, golang-migrate, docker-buildx — is installed by `task setup:tools`.

### Inner loop

```sh
task dev          # build image → deploy → wait for postgres
task db:forward   # port-forward postgres to localhost:5432 (separate terminal)
task db:shell     # Open a psql shell on the postgres instance
task db:migrate   # run pending migrations
task dev:logs     # tail API logs
```

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

### Auth bypass

`CLERK_DEV_BYPASS=true` in `values.dev.yaml` skips JWT verification entirely. All requests are treated as user `dev_user_local` / org `dev_org_local` with role `org:admin`. The config validator prevents this flag from being set outside `ENV=development`.

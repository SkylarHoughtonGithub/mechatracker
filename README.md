# mechatracker

A repair tracking platform for mechanics, auto shops, and fleet managers. Search for repair resources across multiple sources, build recipe-style repair cards with parts and tools checklists, and track repairs from first wrench turn to completion — with full history per vehicle.

---

## What it does

- **Garage** — save vehicles by year, make, model, and trim. Drill down from vehicle → system → repair type.
- **Repair cards** — recipe-style profiles listing required parts, tools, torque specs, and step-by-step procedures. Seeded from global templates, customizable per org.
- **Resource aggregation** — search YouTube tutorials, web forums, uploaded PDF manuals, and parts pricing in one place. Attach any result directly to a repair.
- **Repair tracking** — log step completions, add timestamped notes, attach photos at the repair or step level. Track parts status from needed → ordered → installed.
- **History** — completed repairs are archived as immutable snapshots per vehicle. Full cost and time history retained indefinitely.
- **Multi-user** — solo mechanics, shop teams, and fleet operations all use the same model. Users belong to an org; shops and fleets have multiple users with role-based access.

---

## Tech stack

### Frontend
| Layer | Technology |
|---|---|
| Web | Next.js (React, TypeScript) |
| Mobile | React Native + Expo |
| Shared types & API client | Generated from OpenAPI spec via `openapi-typescript` |

### Backend
| Layer | Technology |
|---|---|
| API server | Go |
| Background workers | Go (separate binary) |
| Database | PostgreSQL |
| DB query layer | sqlc (generated, type-safe) |
| Migrations | golang-migrate |
| Auth | Google OAuth 2.0 (ID-token verification in Go middleware; HS256 session JWT in `mt_session` cookie) |

### Infrastructure
| Layer | Technology |
|---|---|
| Compute | [Skylab](https://github.com/SkylarHoughtonGithub/skylab-gitops) platform — KubeVela Application in `workloads/mechatracker/` |
| Database | Crossplane-managed Postgres via [skylab-crossplane](https://github.com/SkylarHoughtonGithub/skylab-crossplane) `Postgres` XRD (RDS or CloudSQL, composition-selected) |
| File storage | Crossplane-managed bucket via `Bucket` XRD (S3 or GCS, composition-selected) |
| Container registry | GitHub Container Registry (GHCR) |
| Cloud foundations | [skylab-terraform](https://github.com/SkylarHoughtonGithub/skylab-terraform) (OIDC, Roles Anywhere, buckets, trust anchors) |
| CI/CD | GitHub Actions |
| Mobile builds | Expo EAS Build + EAS Submit |

### Observability
| Layer | Technology |
|---|---|
| Error tracking | Sentry (API, web, mobile — separate integrations) |
| Push notifications | Expo Push → APNs (iOS) + FCM (Android) |

---

## Repo structure

```
mechatracker/
├── api/          # Go API and background worker
├── web/          # Next.js web app
├── mobile/       # React Native + Expo mobile app
├── shared/       # Generated TypeScript types and API client
├── infra/        # Terraform and deploy scripts
├── docs/         # Architecture decisions
├── docker-compose.yml   # Local development stack
├── Taskfile.yml         # Developer commands
└── package.json         # npm workspaces root (web, mobile, shared)
```

See [`docs/decisions/repo.md`](docs/decisions/repo.md) for the full layout.

---

## Code generation

Several parts of the codebase are generated from source definitions and should not be hand-edited.

| Generator | Source | Output |
|---|---|---|
| sqlc | `api/queries/*.sql` | `api/internal/store/` |
| swaggo | Go handler annotations | `api/openapi/openapi.yaml` |
| openapi-typescript | `api/openapi/openapi.yaml` | `shared/types/` + `shared/client/` |
| mockery | Go interfaces | `api/mocks/` |

Run `task generate` to regenerate everything. CI validates generated files are up to date on every PR.

---

## Getting started

```sh
# Install bootstrap tools
./hack/bootstrap.sh

# Install dependencies and cluster
task setup

# Start local stack (API + Postgres + web)
task dev
```

---

## Key commands

```sh
task dev                              # start full local stack
task generate                         # regenerate all generated files
task api:test                         # run Go tests
task db:new NAME=add_something        # create a new migration and scaffolds sqlc migration files
```

---

## Decisions log

Infrastructure and architecture decisions are documented in [`docs/decisions/`](docs/decisions/).

| Decision | Status | Doc |
|---|---|---|
| Auth provider | Google OAuth 2.0 | [auth.md](docs/architecture/auth.md) |
| File storage | Crossplane `Bucket` XRD (S3 or GCS) | [storage.md](docs/architecture/storage.md) |
| Hosting | Skylab platform (KubeVela + Crossplane) | [hosting.md](docs/architecture/hosting.md) |
| Repo structure | Monorepo — Go + Next.js + Expo | [repo.md](docs/architecture/repo.md) |
| Data model | PostgreSQL schema | [data_model.md](docs/architecture/data_model.md) |
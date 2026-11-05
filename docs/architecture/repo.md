---
layout: default
title: Repo Architecture
parent: Architecture
---
## Monorepo layout

```
mechaTracker/
├── api/                        # Go API — source of truth for the data contract
│   ├── cmd/
│   │   ├── api/                # main.go — API server entrypoint
│   │   └── worker/             # main.go — background worker entrypoint
│   ├── internal/
│   │   ├── handler/            # HTTP handlers per domain
│   │   ├── service/            # Business logic layer
│   │   ├── store/              # sqlc-generated DB queries (do not hand-edit)
│   │   ├── middleware/         # Auth, logging, recovery
│   │   └── model/              # Domain types
│   ├── migrations/             # golang-migrate SQL files (schema source of truth)
│   ├── queries/                # Hand-written SQL → sqlc generates store/
│   ├── openapi/                # Generated OpenAPI spec → feeds shared/
│   ├── mocks/                  # mockery-generated mocks (do not hand-edit)
│   ├── sqlc.yaml
│   ├── Dockerfile
│   └── go.mod
│
├── web/                        # Next.js web app
│   ├── app/                    # App router pages and layouts
│   ├── components/             # UI components
│   ├── lib/                    # Hooks, utils — imports from shared/
│   ├── Dockerfile
│   ├── next.config.ts
│   └── package.json
│
├── mobile/                     # React Native + Expo
│   ├── app/                    # Expo Router screens
│   ├── components/             # Mobile UI components
│   ├── lib/                    # Hooks, offline queue — imports from shared/
│   ├── app.json
│   ├── eas.json                # EAS Build + Submit config
│   └── package.json
│
├── shared/                     # Generated from OpenAPI — do not hand-edit
│   ├── types/                  # TypeScript types
│   ├── client/                 # Typed fetch client
│   └── package.json            # Workspace package — imported by web + mobile
│
├── infra/
│   ├── helm/                   # Local-dev Helm chart (k3d only)
│   │   └── mechatracker/
│   ├── k3d/                    # k3d cluster + registry config
│   └── buildkit/               # buildx / BuildKit config
│
# Production deployment lives in skylab-gitops:
#   workloads/mechatracker/       — KubeVela Application
#   workloads/mechatracker-db/    — Crossplane-managed Postgres
│
├── .github/
│   └── workflows/
│       ├── api.yml             # test → build → push → deploy Go API
│       ├── web.yml             # test → build → push → deploy Next.js
│       ├── mobile.yml          # EAS Build on version tag
│       └── generate.yml        # Regenerate shared/ when openapi spec changes
│
├── docs/
│   └── decisions/              # auth.md, storage.md, hosting.md, repo.md
│
├── docker-compose.yml          # Local dev only — API + Postgres + web
├── Taskfile.yml                # All developer commands
├── package.json                # npm workspaces root — web, mobile, shared
└── .env.example
```

---

## Key conventions

- `shared/` is always generated — never hand-edited
- `api/internal/store/` and `api/mocks/` are always generated — never hand-edited
- `api/migrations/` is the single source of truth for the database schema
- `api/go.mod` is self-contained — Go has no dependency on the JS workspace
- Generated files are committed to the repo and validated in CI on every PR
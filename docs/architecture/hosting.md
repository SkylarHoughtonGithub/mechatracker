---
layout: default
title: Hosting Architecture
parent: Architecture
---

## Hosting Requirements

| Requirement | Notes |
|---|---|
| Single platform | Zero egress between compute and storage — same provider throughout |
| Container-based | Docker containers for all services |
| IaC | Terraform — full infrastructure as code |
| Cost-driven | Technical overhead is less important than cost |
| US-only | No global distribution needed early on |
| Managed Postgres | No self-managed database |

---

## Decision

**Platform: DigitalOcean (IaaS — Droplets + Docker)**

All services run on DigitalOcean. No cross-platform egress.

| Service | Product | Est. cost |
|---|---|---|
| Go API + workers | Droplet (2GB RAM, 1 vCPU) | $18/mo |
| Next.js web | Droplet (2GB RAM, 1 vCPU) | $18/mo |
| PostgreSQL | DO Managed Database (1GB RAM) | $15/mo |
| Photos + PDFs | DO Spaces (250GB + 1TB egress) | $25/mo |
| Container images | DO Container Registry (DOCR) | $0 (free tier) |
| Egress | Pooled with Droplet transfer allowances | $0 |
| **Total early stage** | | **~$76/mo** |

Managed Postgres scales to $50/mo at growth stage (4GB RAM). Droplets scale independently per service.

---

## Why DigitalOcean over alternatives

### vs Fly.io
Fly.io has no native object storage. Photos and PDFs would live on R2 or B2 — external to Fly, meaning egress on every photo load. That breaks the single-platform zero-egress requirement. Fly is the better option only if storage egress is not a concern.

### vs Render / Railway
Neither has object storage. Same egress problem as Fly.io. Both also have weaker IaC stories — no Terraform provider on Railway, YAML blueprints only on Render.

### vs Linode / Akamai
Comparable platform with Terraform support and pooled egress. Managed Postgres is significantly more expensive on Linode ($81.60/mo for a single-node 4GB cluster vs $50/mo on DO). Pricing has become less competitive post-Akamai acquisition.

### vs AWS / GCP
$0.09/GB egress on both. Viable only behind CloudFront/Cloud CDN which adds cost and complexity. Overkill for early stage. Migration path if needed later.

### App Platform (PaaS) vs Droplets (IaaS)
PaaS costs ~$6-8/mo more early on and the gap widens at scale — PaaS charges a premium per compute unit. Given technical comfort with infrastructure, IaaS gives more RAM and CPU per dollar. Droplet transfer allowances also pool with Spaces, giving more total zero-egress headroom than App Platform.

---

## Architecture

```
DigitalOcean account
├── VPC (private network)
│   ├── Droplet: api (Go API + background workers)
│   ├── Droplet: web (Next.js)
│   └── Managed PostgreSQL cluster
├── Spaces bucket: mechaTracker-uploads (photos, PDFs)
├── Container Registry: mechaTracker (Docker images)
└── Cloud Firewall (public ports: 80, 443 only)
```

All internal traffic (API ↔ Postgres, API ↔ workers) stays on the private VPC. Postgres is never publicly addressable. Spaces egress is pooled with Droplet transfer allowances — effectively zero for reasonable traffic volumes.

---

## Deployment model

### Containers
All services run as Docker containers on their respective Droplets. Docker Compose manages service definitions, environment injection, and restarts.

```
api Droplet
└── docker-compose.yml
    ├── mechaTracker-api      (Go binary, port 8080)
    └── mechaTracker-worker   (Go background worker)

web Droplet
└── docker-compose.yml
    └── mechaTracker-web      (Next.js, port 3000)
```

Nginx runs on each Droplet as a reverse proxy, handles SSL termination via Let's Encrypt, and forwards traffic to the container.

### CI/CD flow
```
git push → GitHub Actions
  → go test ./...
  → docker build
  → docker push → DOCR (DO Container Registry)
  → SSH to Droplet → docker compose pull → docker compose up -d
```

No downtime during deploys — Docker Compose rolling restart. Zero-downtime strategy graduates to a load balancer + multiple API Droplets when traffic warrants it.

### Terraform scope
Everything provisioned and managed in code:

```hcl
# Resources managed by Terraform
digitalocean_droplet          # api, web
digitalocean_database_cluster # postgres
digitalocean_database_db      # mechaTracker db
digitalocean_spaces_bucket    # uploads
digitalocean_container_registry
digitalocean_vpc
digitalocean_firewall
digitalocean_domain
digitalocean_record           # DNS A records
```

State stored in Terraform Cloud or DO Spaces backend. Secrets managed via environment variables injected at deploy time — never in Terraform state.

---

## External services (not hosted on DO)

These connect to the Go API over HTTPS — no platform egress cost since they are outbound calls from the API, not serving assets.

| Service | Provider | Purpose |
|---|---|---|
| Auth (identity) | Google OAuth 2.0 | ID-token issuance; mechatracker owns the user/org/invite tables in its own Postgres |
| Telemetry (API) | Sentry or Datadog | Error tracking, metrics |
| Telemetry (web) | Sentry | Frontend error tracking |
| Telemetry (mobile) | Sentry | Mobile crash reporting |
| Push notifications | Expo Push + APNs/FCM | Mobile notifications |
| Search APIs | YouTube Data API, SerpAPI | Tutorial and resource search |
| Parts pricing | TBD (Phase 4) | Parts supplier API |

---

## Telemetry detail

Telemetry is instrumented independently on each platform — they all funnel to the same provider but are separate integrations.

| Platform | SDK | What it captures |
|---|---|---|
| Go API | `sentry-go` or Datadog APM | Request errors, latency, panics, DB query times |
| Next.js | `@sentry/nextjs` | Frontend JS errors, SSR errors, Web Vitals |
| React Native | `@sentry/react-native` | Crashes, JS errors, ANRs, slow frames |

Sentry covers all three with one account and one dashboard — recommended for early stage. Datadog adds deeper APM and infrastructure metrics when needed at growth stage.

---

## Push notifications detail

The Go API sends a single request to Expo Push Notification Service. Expo handles routing to APNs (iOS) and FCM (Android) — no separate Apple or Google integration in Go code.

```
Go API → Expo Push API → APNs → iOS device
                       → FCM  → Android device
```

APNs and FCM credentials are configured once in the Expo EAS dashboard. The Go API only needs the Expo push token stored per user in Postgres.

---

## Scaling path

| Stage | Change | Est. cost |
|---|---|---|
| Early | Single api Droplet + single web Droplet | ~$76/mo |
| Growth | Upgrade Droplet sizes, scale Postgres to 4GB | ~$129/mo |
| Scale | Add load balancer + 2nd api Droplet, read replica | ~$180/mo |
| Mature | Kubernetes (DOKS) if multi-service complexity warrants it | Variable |

Kubernetes is available on DO (DOKS) and uses the same Terraform provider — migration path exists without leaving the platform.

---

## Open questions

| Question | Notes |
|---|---|
| Single Droplet vs two early on? | Could co-locate API and web on one $24/mo Droplet (4GB) to start — split when traffic justifies it |
| Terraform state backend | Terraform Cloud (free tier) or Spaces bucket as backend |
| Staging environment | Separate DO project with smaller Droplets — same Terraform, different vars |
| Log aggregation | DO has basic log forwarding — Datadog or Loki for structured log search |
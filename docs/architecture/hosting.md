---
layout: default
title: Hosting Architecture
parent: Architecture
---

## Decision

**Platform: [Skylab](https://github.com/SkylarHoughtonGithub/skylab-gitops)** — mechatracker ships as a KubeVela Application that Argo CD delivers to a Kubernetes cluster. Cloud resources (Postgres, buckets, the cluster itself) are Crossplane XRs resolved to AWS or GCP by composition, not by this app.

Cloud choice is a platform decision, not an application decision. mechatracker's manifests contain no cloud-specific fields.

---

## Deployment shape

```
platform/tenants  (Gitea)
├── applications/mechatracker.yaml      # KubeVela Application — api component
└── applications/mechatracker-db.yaml   # KubeVela Application — postgres component
       ▲
       │  watched by a git-files generator ApplicationSet in skylab-gitops
       ▼
Argo CD  (homelab hub)
       │
       ▼
Kubernetes cluster  (EKS or GKE, provisioned by skylab-crossplane KubernetesCluster XRD)
├── mechatracker-api         # go-service component — Deployment + Service
├── mechatracker-api cert    # StepClusterIssuer-signed TLS
└── XPostgres XR ─────────► RDS Instance (AWS)   or   CloudSQL Instance (GCP)
                                      │                             │
                                      ▼                             ▼
                                   db-url Secret  ◄─── compose Job derives DATABASE_URL
```

## Platform components used

| Platform piece | Role |
| --- | --- |
| **go-service ComponentDefinition** (skylab-gitops/charts/kubevela-defs) | Wraps KubeVela `webservice` with Skylab opinions: OTel env wired to the in-cluster Alloy OTLP endpoint, `skylab.io/system` labels, `/healthz` probes |
| **postgres ComponentDefinition** (skylab-gitops/charts/kubevela-defs) | Emits an `XPostgres` XR and a compose Job that derives `DATABASE_URL` into a `<name>-url` Secret the api `envFrom`'s |
| **Postgres XRD** (skylab-crossplane v0.4.0) | AWS: RDS Instance. GCP: CloudSQL DatabaseInstance + Database + User. Both write connection details via `writeConnectionSecretToRef`. |
| **Bucket XRD** (skylab-crossplane v0.1.0) | AWS: S3 bucket. GCP: GCS bucket. Used for photos + PDF manuals once the file-storage path lands. |
| **StepClusterIssuer** (skylab-gitops/charts/cert-manager) | Internal TLS from the step-ca root; public certs via ACME on the same `charts/cert-manager` wrapper |
| **Observability** (skylab-gitops wave 5) | Alloy forwards logs + traces to Loki + Tempo; kube-prometheus-stack scrapes metrics; Grafana dashboards; Sloth burn-rate rules on the PrometheusServiceLevel CR mechatracker emits |

## Image pipeline

```
git push → GitHub Actions (.github/workflows/api-ci.yaml)
  → go test ./...
  → buildx build (linux/amd64 + linux/arm64, multi-arch manifest)
  → docker push → ghcr.io/skylarhoughtongithub/mechatracker-api:sha-<sha>
  → Trivy CRITICAL (exit=1 on finding)
  → cosign sign + SBOM + provenance=max
```

The KubeVela Application's `go-service` component pins a specific image tag. Renovate opens PRs against `platform/tenants` as new tags land in GHCR.

---

## External services (connect to the Go API over HTTPS)

These are outbound calls from the API, not platform egress.

| Service | Provider | Purpose |
|---|---|---|
| Auth (identity) | Google OAuth 2.0 | ID-token issuance; mechatracker owns the user/org/invite tables in its own Postgres |
| Telemetry (API) | Sentry or Datadog | Error tracking, metrics (also emitted to Skylab Tempo/Prometheus) |
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

The Go API also emits OpenTelemetry spans over OTLP/gRPC to the in-cluster Alloy endpoint; Alloy forwards them to Tempo on the Skylab hub. Metrics are scraped by kube-prometheus-stack. Sloth generates burn-rate `PrometheusRules` from the `PrometheusServiceLevel` CR the KubeVela Application emits.

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

Horizontal scaling is a Kubernetes concern. `replicas` on the Deployment are set per environment in the KubeVela Application; HPA follows once CPU/memory baselines are established. Postgres scales by bumping the `size` parameter on the `XPostgres` XR — Crossplane reconciles the underlying RDS or CloudSQL instance.

---

## Local dev vs production

Local dev runs on k3d with an in-cluster `postgres:17` StatefulSet via the Helm chart at `infra/helm/mechatracker/`. Production uses the KubeVela Application in `skylab-gitops` with Crossplane-managed Postgres and buckets. The two share Go code and Docker images but nothing else.

See [`docs/development/local_dev.md`](../development/local_dev.md) for the local-dev stack.

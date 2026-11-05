---
layout: default
title: Storage Architecture
parent: Architecture
---

## Storage Requirements

Object storage should work for all application needs

| Type | Est. size | Access pattern |
|---|---|---|
| Repair photos | 1–6 MB each | Loaded every time a repair profile opens |
| PDF manuals | 2–20 MB each | Uploaded once, indexed, occasionally retrieved |
| Compressed thumbnails | 50–200 KB each | Loaded on list/dashboard views |

**Key constraint:** Photos are loaded frequently. Egress cost matters more than storage cost.

---

## Options

| Provider | Storage | Egress | Free tier | S3 compat | Best if... |
|---|---|---|---|---|---|
| Cloudflare R2 | $0.015/GB | Free always | 10 GB + 1M ops | Yes | Hosting anywhere |
| Backblaze B2 + Cloudflare | $0.006/GB | Free via CF | 10 GB | Yes | You want cheapest possible |
| AWS S3 | $0.023/GB | $0.09/GB | 5 GB (12mo) | Native | Hosting on AWS |
| Google Cloud Storage | $0.020/GB | $0.08/GB | 5 GB (12mo) | Yes | Hosting on GCP |
| Linode Object Storage | $0.020/GB | 1 TB free | 250 GB incl. | Yes | Hosting on Linode |
| Supabase Storage | $0.021/GB | $0.09/GB | 1 GB | Yes | All-in on Supabase |
| Azure Blob | $0.018/GB | $0.087/GB | 5 GB (12mo) | No | Skip |
| MinIO (self-hosted) | Infra only | Free | Unlimited | Yes | Skip early on |

---

## Pros & cons

### Cloudflare R2
**Pros**
- Zero egress fees — the single biggest cost driver for a photo-heavy app
- S3-compatible — identical Go code to any other option
- Can serve via Cloudflare CDN for fast global delivery at no extra cost
- Simple, predictable pricing

**Cons**
- Newer service, less community documentation than S3
- No built-in image transformation (needs separate service)
- Requires a Cloudflare account even if not hosting there

---

### Backblaze B2 + Cloudflare CDN
**Pros**
- Cheapest storage cost available ($0.006/GB)
- Zero egress via Cloudflare bandwidth alliance
- S3-compatible

**Cons**
- Two-vendor setup — more configuration than R2
- Cloudflare dependency for the free egress benefit
- Less documentation for the B2 + CF combo

---

### AWS S3
**Pros**
- Most mature, best documented object storage available
- Native Go SDK — first-class support
- Rich ecosystem (Lambda triggers, image resizing, lifecycle rules)
- Natural choice if hosting API on AWS

**Cons**
- $0.09/GB egress adds up fast when photos load frequently
- Need CloudFront in front to make egress tolerable — adds cost and setup
- Pricing complexity — request costs, transfer tiers, regional variance

---

### Google Cloud Storage
**Pros**
- Solid S3-compatible option
- Internal egress free if API is also on GCP
- Good Go client library

**Cons**
- $0.08/GB egress if not on GCP
- No advantage over R2 unless already committed to Google Cloud

---

### Linode Object Storage
**Pros**
- 1 TB egress included — very generous
- 250 GB storage included in base plan
- S3-compatible
- Natural fit if hosting API on Linode

**Cons**
- Less mature ecosystem than AWS or GCP
- CDN requires separate setup

---

### Supabase Storage
**Pros**
- Bundles auth + database + storage into one platform
- Built-in image resizing on paid plans

**Cons**
- Only 1 GB free — very limited
- $0.09/GB egress above free tier
- Only makes sense if going all-in on Supabase — conflicts with standalone Go API + Postgres

---

### Azure Blob Storage
**Pros**
- Solid enterprise option if already in Microsoft ecosystem

**Cons**
- Not S3-compatible — different SDK, different Go code
- $0.087/GB egress
- No advantage for this stack

---

### MinIO (self-hosted)
**Pros**
- Free, unlimited, full S3 compatibility
- Complete control

**Cons**
- You manage the infrastructure, backups, and uptime
- Not worth the ops overhead at this stage

---

## Key insight

Egress is the cost that matters. Storage is cheap everywhere — the difference between providers is pennies per GB. Egress is where AWS, GCP, and Azure charge $0.08–0.09/GB, and for a photo-heavy app where repair profiles load multiple images on every open, that adds up fast.

All S3-compatible options use identical Go code — swapping providers is just a config change.

---

## Decision

**Crossplane `Bucket` XRD, resolved to S3 on AWS or GCS on GCP by composition** (see [`skylab-crossplane`](https://github.com/SkylarHoughtonGithub/skylab-crossplane), XRD v0.1.0).

mechatracker's manifest declares the storage dependency via the Skylab `bucket` ComponentDefinition; which cloud fulfills it is a composition decision on the target cluster (`spec.cloud: aws | gcp`). The `writeConnectionSecretToRef` output is picked up by the Go API as `envFrom` for S3-compatible SDK config.

Trade-off accepted: egress on the chosen cloud is paid in exchange for cloud-agnostic application manifests and no static credentials — IAM comes from Roles Anywhere (AWS) or Workload Identity Federation (GCP), both rooted at the Skylab step-ca trust anchor. See [`hosting.md`](hosting.md) for the deployment context.

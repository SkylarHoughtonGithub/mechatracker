---
layout: default
title: Auth Architecture
parent: Architecture
---

## Auth Requirements

| Requirement | Notes |
|---|---|
| User accounts with roles | Individual, technician, shop owner, fleet manager |
| Multi-user workspaces | A shop has multiple techs — org/team concept needed |
| React Native (Expo) SDK | Mobile is a first-class platform |
| Next.js SDK | Web app |
| Go-compatible JWT verification | Backend must validate tokens — ideally no vendor SDK required on Go side |
| Social login | Google at minimum, Apple for iOS |
| Offline tolerance | Techs work in shops — token must survive connectivity gaps |
| MFA support | Nice to have, required eventually for fleet/enterprise tier |

---

## Options

| Provider | Free tier | Orgs/teams | React Native | Go support | Self-host | Offline |
|---|---|---|---|---|---|---|
| Clerk | 10k MAU | Built-in | Official SDK | JWT verify | No | Token TTL |
| Kinde | 10.5k MAU | Built-in | Official SDK | JWT verify | No | Token TTL |
| Auth0 | 25k MAU | Manual build | Official SDK | Official SDK | No (Okta) | Token TTL |
| Firebase Auth | Generous | Manual build | Official SDK | Official SDK | No | Best native |
| Supabase Auth | 50k MAU | No | Official SDK | JWT verify | Yes | Token TTL |
| WorkOS AuthKit | 1M MAU | Enterprise focus | No official SDK | JWT verify | No | Token TTL |
| Keycloak | Unlimited | Built-in | Manual | Manual | Yes (required) | Full control |
| SuperTokens | 5k MAU | Built-in | Manual | Official SDK | Yes | Full control |
| Better Auth | Unlimited | Built-in | Manual | N/A (JS only) | Yes (required) | Full control |
| AWS Cognito | 50k MAU (new) | Manual build | Official SDK | Official SDK | No | Decent |
| Roll your own | Free | Custom | Manual | Native | Yes | Full control |

---

## Pros & cons

### Clerk
**Pros**
- Organizations API maps directly to shop/fleet model — no custom build needed
- Best developer experience for React/Next.js — pre-built components ship in minutes
- Official React Native SDK
- Go backend verifies standard JWTs with public key — no Clerk SDK on backend at all
- Data export available at no cost — migration path exists if needed
- Active development, excellent documentation

**Cons**
- $25/mo + $0.02/MAU above 10k free tier — can get expensive at scale
- No self-hosting option — full vendor dependency
- Offline: JWT expiry (default 1hr) requires network to refresh; mitigate with extended TTL
- Proprietary component architecture creates some UI lock-in

---

### Kinde
**Pros**
- 10.5k MAU free — slightly more generous than Clerk
- Organizations built-in — comparable to Clerk for multi-tenancy
- Official React Native and Next.js SDKs
- Includes billing module — could replace Stripe for subscription management later
- Go SDK available
- Actively improving and competitive on pricing

**Cons**
- Newer and smaller ecosystem than Clerk or Auth0 — fewer community resources
- Documentation has gaps compared to more mature providers
- React Native SDK less battle-tested than Clerk's
- Smaller community = harder to find answers when things go wrong

---

### Auth0
**Pros**
- 25k MAU free — most generous hosted free tier of the major players
- Most mature and battle-tested platform — Okta backed
- Official Go Admin SDK — first-class backend support
- Official React Native SDK
- Highly extensible — custom Rules and Actions for complex auth flows
- Strong enterprise compliance story (SOC2, HIPAA, etc.)

**Cons**
- Notoriously expensive at scale — pricing doesn't offer volume discounts, reportedly increases per-user cost as you grow
- Steeper learning curve — 7-10 configuration steps vs Clerk's 3-5
- No built-in organizations/teams concept — multi-tenant shop/fleet model requires custom build
- Heavier, more complex than needed for an early-stage app
- Had two security incidents in recent years

---

### Firebase Auth
**Pros**
- Most generous free tier in practice — rarely costs anything at small-medium scale
- Official Go Admin SDK — first-class, well-documented
- Official React Native SDK (via React Native Firebase)
- Best offline handling — Firebase SDKs designed with offline-first in mind, token refresh is more graceful
- Battle-tested at massive consumer scale

**Cons**
- No organizations/teams concept — entire multi-user workspace model must be built custom in your own Postgres
- Ties you to Google Cloud ecosystem
- UI components are barebones — sign-in screens need more custom work
- Firebase platform lock-in risk if you use other Firebase services alongside it

---

### Supabase Auth
**Pros**
- 50k MAU free — most generous free tier of all options
- Open source and self-hostable
- Good if bundling with Supabase Postgres + Storage (one platform)
- Row-level security integrates auth directly with data access policies

**Cons**
- No organizations/teams — multi-tenant model must be built from scratch
- Auth UI library was archived October 2025 — now community-maintained, not officially supported
- Best value only if going all-in on Supabase platform — awkward alongside a standalone Go API
- Session lifetime configuration is confusing and poorly documented

---

### WorkOS AuthKit
**Pros**
- 1 million MAU free — by far the most generous free tier
- Purpose-built for B2B — SSO, SCIM, audit logs are primary features
- Strong enterprise story if PartTrack targets shops/fleets at scale
- Clean JWT-based verification, Go compatible

**Cons**
- No official React Native SDK — mobile integration is manual
- Designed as enterprise feature layer, not a full consumer auth solution
- Overkill for individual/DIY user tier
- Less community documentation for standard use cases

---

### Keycloak
**Pros**
- Fully open source, self-hosted — no per-user costs ever
- Full control over every auth detail
- Organizations and multi-tenancy built-in
- Standard OIDC/OAuth2 — any SDK works with it

**Cons**
- Notoriously complex to configure and operate — described as "wartime stories" by experienced ops engineers
- You manage infrastructure, updates, security patches, uptime
- Significant setup time before any app work begins
- Not appropriate as a starting point for a new project

---

### SuperTokens
**Pros**
- Open source with a managed cloud option
- Official Go SDK — one of the few with proper Go support
- Self-hostable — no vendor lock-in
- Organizations and multi-tenancy built-in
- More affordable than Clerk/Auth0 at scale

**Cons**
- React Native integration is manual — no official SDK
- Smaller ecosystem and community than the major players
- Managed cloud free tier only 5k MAU — low compared to alternatives
- Less polished developer experience

---

### Better Auth
**Pros**
- Fully open source and self-hosted — free at any scale
- TypeScript-first — integrates naturally with Next.js
- Organizations built-in
- No vendor dependency at all

**Cons**
- JavaScript/TypeScript only — no Go SDK, no React Native SDK
- Self-hosted means you own the infrastructure, security, and maintenance
- Very new — smaller community and fewer production war stories
- Not a fit for a Go backend

---

### AWS Cognito
**Pros**
- 50k MAU free (recently restructured)
- Official Go SDK
- Natural fit if hosting API on AWS
- Handles large scale well

**Cons**
- Notoriously painful developer experience — complex configuration, confusing terminology
- No organizations/teams concept — custom build required
- Recently tripled pricing for 60k+ MAU — pricing stability concerns
- React Native integration requires extra setup

---

### Roll your own (Go + JWT)
**Pros**
- No vendor cost or dependency
- Full control — offline works exactly how you design it
- No per-user pricing at any scale

**Cons**
- Weeks of work before building anything useful — password reset, MFA, social login all from scratch
- Security surface you own and maintain forever
- No mobile SDK — React Native integration is entirely manual
- Auth is not what makes PartTrack valuable — time is better spent elsewhere early on

---

## Key insight

The organizations/teams feature is the deciding factor for PartTrack. The shop/fleet multi-user model means you need a concept of "workspace" — a shop owner and their techs all belonging to the same account. Auth0, Firebase, Cognito, Supabase, and WorkOS all require building this from scratch. Clerk and Kinde have it built-in, which represents weeks of saved development time.

**Go backend note:** All JWT-based providers (Clerk, Kinde, Auth0, Firebase, Supabase, WorkOS) allow your Go middleware to verify tokens using a public key — no vendor SDK required on the backend. Your API stays provider-agnostic where it counts.

**Offline note:** All hosted providers face the same constraint — token refresh requires a network call. Firebase handles this most gracefully at the SDK level. For any other provider, mitigate with an extended JWT TTL (24h for mobile sessions) and an offline action queue in React Native that syncs on reconnect.

---

## Decision

**Status:** Leaning Clerk — open to Kinde as a strong alternative

**Rationale:** Organizations API covers the shop/fleet model without custom build. Best React/Next.js developer experience. Official React Native SDK. Go backend uses standard JWT verification — no vendor dependency on the API layer. Kinde is worth a closer look given its slightly larger free tier and built-in billing module.

**Revisit if:** MAU grows past ~5k paying users and auth costs become a meaningful line item. Migration is painful but Clerk provides full data export and documented migration paths.

**Key risk to track:** Vendor pricing changes. Document all user IDs in your own Postgres from day one so the data layer is auth-provider-agnostic regardless of which you choose.

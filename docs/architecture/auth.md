---
layout: default
title: Auth Architecture
parent: Architecture
---

## Current decision — Google OAuth + owned user model

**Status:** Decided — reverted from Clerk (Oct 2026).

mechatracker handles identity through Google's OAuth 2.0 ID-token flow and owns the user/org/invite data model in its own Postgres. Clerk was removed because its hosted org-management surface added operational cost (ngrok tunnel for webhook delivery, dashboard webhook config, dev vs prod signing-secret rotation) without buying anything the app couldn't do itself at this scale.

### Shape

| Concern | Implementation |
|---|---|
| Identity | Google OAuth 2.0, server-side ID-token verification via `google.golang.org/api/idtoken` |
| Session | Signed JWT (HS256) in an httpOnly, SameSite=Lax cookie; 12h TTL; `SESSION_SIGNING_KEY` from env |
| User model | `users` table with `google_sub` (unique), `org_id` (NOT NULL), `role` (`owner`/`admin`/`member`) |
| Tenancy | Single-org-per-user. Multi-tenancy is preserved (orgs are isolated); cross-org work requires a second Google account. |
| Onboarding | First Google sign-in with no `users` row → "create org or paste invite" screen |
| Invites | `invites` table, 24h expiry, single-use, email-matched against the invitee's verified Google email |
| Webhooks | None. Google doesn't push; the API only ever pulls (JWKS verification). |
| Local dev | `DEV_AUTH_BYPASS=true` short-circuits token verification and injects the dev seed user; rejected outside `ENV=development` |

### Why single-org-per-user

Clerk's multi-org model implied one user could belong to many orgs with different roles in each — convenient, but it drove a lot of UI (org switcher), middleware (which org is this request against?), and data (`memberships` join table) that the app didn't actually benefit from. In mechatracker's domain, a technician works at a shop; if they move to a different shop they get a new account. Keeping `org_id` on `users` as NOT NULL preserves the simpler shape and matches the existing schema.

### Why Google (and not Gitea OIDC, which the rest of Skylab uses)

mechatracker is a public-facing SaaS target, not a platform internal tool. Users signing up with their personal Gmail is the lowest-friction path. The Skylab control-plane tools (Backstage, Argo CD, Grafana) stay on Gitea OIDC — different audience, different IdP.

### Routes

| Route | Purpose |
|---|---|
| `POST /auth/google` | Verify Google ID token, mint session cookie. If `users` row exists, return it; otherwise return onboarding claims. |
| `POST /auth/logout` | Clear session cookie. |
| `GET /auth/me` | Return current user + org, or `needs_onboarding: true` with Google claims. |
| `POST /orgs` | Create a new org + `users` row with role=owner for the current Google claims. |
| `POST /orgs/invites` | Owner/admin creates an invite (24h, single-use). |
| `POST /invites/{token}/accept` | Create a `users` row scoped to the invite's org (if the Google account has no existing row). |

### Config

- `GOOGLE_CLIENT_ID` — required; also used as the ID-token audience.
- `GOOGLE_ALLOWED_HD` — optional, restricts to a Google Workspace hosted domain.
- `SESSION_SIGNING_KEY` — required, 32+ bytes.
- `SESSION_COOKIE_DOMAIN`, `SESSION_COOKIE_SECURE` (default true), `SESSION_TTL` (default 12h).
- `DEV_AUTH_BYPASS` — only honored when `ENV=development`.

**Revisit if:** auth becomes a feature (SSO for enterprise shops, SAML, SCIM provisioning). At that point WorkOS or a reintroduction of a hosted provider is the right conversation — not a Clerk-shaped product, but one chosen deliberately for the enterprise need.

---

## Historical: provider comparison (Oct 2025 — pre-revert)

The sections below were the original provider-comparison exercise that led to picking Clerk. They are preserved as context for why Clerk was chosen initially and why it didn't stick — see "Current decision" above for the live design.

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
- Strong enterprise story if mechaTracker targets shops/fleets at scale
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
- Auth is not what makes mechaTracker valuable — time is better spent elsewhere early on

---

## Key insight

The organizations/teams feature is the deciding factor for mechaTracker. The shop/fleet multi-user model means you need a concept of "workspace" — a shop owner and their techs all belonging to the same account. Auth0, Firebase, Cognito, Supabase, and WorkOS all require building this from scratch. Clerk and Kinde have it built-in, which represents weeks of saved development time.

**Go backend note:** All JWT-based providers (Clerk, Kinde, Auth0, Firebase, Supabase, WorkOS) allow your Go middleware to verify tokens using a public key — no vendor SDK required on the backend. Your API stays provider-agnostic where it counts.

**Offline note:** All hosted providers face the same constraint — token refresh requires a network call. Firebase handles this most gracefully at the SDK level. For any other provider, mitigate with an extended JWT TTL (24h for mobile sessions) and an offline action queue in React Native that syncs on reconnect.

---

## Decision (historical)

**Status:** Reverted (Oct 2026). See "Current decision — Google OAuth + owned user model" at the top of this doc.

**Original decision:** Clerk, chosen for its Organizations API, prebuilt React components, and official React Native SDK.

**Why it was reverted:**
- Clerk never got past the dev-bypass stage. Full JWT verification (`rsaPEMFromJWK`) remained stubbed; wiring it would have meant pulling in `lestrrat-go/jwx/v2` and still owning the Clerk SDK upgrade path.
- Webhook delivery required ngrok (Clerk → public URL → port-forward → pod), which introduced per-dev config, free-tier subdomain churn, and signing-secret rotation steps.
- The multi-org model that was Clerk's main selling point was more flexibility than the shop/fleet domain actually needed. See "Why single-org-per-user" above.
- Removing a managed-service dependency simplifies the operational story for a hobby/portfolio project — no vendor dashboard to manage, no per-env webhook URLs, no MAU ceiling to track.

**What was kept:** the "own user IDs in your own Postgres" discipline from the original decision — the only auth-related schema change was `clerk_id` → `google_sub`.

**Revisit (reconsider a hosted provider) if:** SSO/SAML, SCIM, or audit-log compliance becomes a product requirement. The honest trigger is "we are selling into enterprise fleets," not "auth code is annoying to maintain."

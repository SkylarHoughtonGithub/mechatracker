---
layout: default
title: Auth Architecture
parent: Architecture
---

# Auth Decision

## Requirements

- User accounts iwth roles (individual, technician, fleet manager, shop owner)
- Works with both Next.js (web) and React Native/Expo (mobile)
- Go-friendly verification path (JWT validation middleware)
- Multi-user workspaces eventually
- Offline tolerance
- Social login nice to have (Google SSO at minimum)


## Considered Solutions

### Clerk

Hosted Auth with built-in UI components

#### Pros
- Go Support
- Official React SDK
- Free Tier (10k MAU)

- Pre built sign-in UI for web/mobile
- Organizations API
- Social Login out of box
- Low  overhead

#### Cons
- Not self hostable
- Expensive at scale ($25/mo per 1k MAU over free tier)
- Vendor lock-in

### Supabase

Auth bundled with Postgres + storage

#### Pros
- Go Support
- Official React SDK
- Free Tier (50k MAU)
- Self-hostable

#### Cons
- Uses Supabase postgres - conflicts with running your own Go-managed DB
- No native organizations/team concepts
- Ecosystem lockin

### Firebase Auth

Googles auth service

#### Pros
- Go Support
- Official React SDK
- Generous Free Tier
- scales well
- google sign in feels native on android

#### Cons
- Not self hostable
- No organizations/team concept - have to built multi-user workspaces 
- UI components are barebones - more custom UI work
- Tied to google cloud

### Self Managed

#### Pros
- No vendor dependency or cost
- Full control
- No per user pricing at scale
- Self hostable

#### Cons
- Lots of overhead
- Security surface in house
- Password reset, MFA, social login from scratch
- No mobile SDK


## Decision
Clerk has the most out of box considerations. Organization API, clean Go integration. Need to consider migratrion when usage becomes high with the cost expected, but is a good problem to have at that phase

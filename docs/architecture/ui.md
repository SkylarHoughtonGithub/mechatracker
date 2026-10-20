---
layout: default
title: UI Architecture
parent: Architecture
---

## Overview

The UI follows a drill-down navigation pattern: **Garage → Vehicle systems → Repair type list → Repair card**. Vehicle context (year/make/model) persists through the full drill-down so torque specs and part numbers stay vehicle-specific at every level.

Web (Next.js) is built first — faster iteration, no app store delays. Mobile (React Native + Expo) follows after the web version is stable, sharing API client and state logic via `/shared`.

---

## Navigation pattern

```
Garage
└── Vehicle (e.g. 2019 Ford F-150)
    └── System (e.g. Engine)
        └── Repair type (e.g. Spark plug replacement)
            └── Repair card (full recipe view)
```

Every screen has a back button and a breadcrumb trail. The breadcrumb supports jump navigation — tapping any crumb jumps directly to that level without stepping back one at a time.

---

## Screen inventory

### Garage
Entry point. Shows all saved vehicles with open repair counts and a stats summary (total vehicles, open repairs, parts spend).

**API:** `GET /v1/garage` — single aggregated call, already implemented.

### Vehicle systems
Vehicle hero card (year/make/model/mileage, open repair count) followed by the system grid: Engine, Suspension, Brakes, Electrical, Transmission, Cooling, etc. Each system tile shows a badge if it has open repairs.

**API:** System data is embedded in the garage response via `systems[]` on each vehicle.

### Repair type list
All repair types for a given vehicle/system combination. Each card shows difficulty pill (easy/medium/hard), estimated time, key torque specs or fluid specs inline, and open/completed badges.

The inline specs (e.g. `26 ft-lb · 5W-30 · 6 qt`) are intentional — the list is designed to answer "what do I need to know before I start" without opening the full repair card.

**API:** `GET /v1/vehicles/:id/systems/:sys/repairs`

### Repair card
The core of the app. Five collapsible sections, all expanded by default on first open:

| Section | Contents |
|---|---|
| Specs grid | Fluid type, capacity, torque spec — surfaced at the top, always visible |
| Parts | Checklist with OEM numbers, prices, gathered checkbox, OEM fit badge |
| Tools | List with notes (e.g. "rent the spring compressor", "apply penetrating oil night before") |
| Steps | Numbered procedure with per-step torque specs and completion state |
| Tutorials & resources | YouTube, forum links, uploaded PDFs — all searchable via Phase 4 aggregation |
| Notes | Timestamped free-form notes per repair |

A "Mark repair complete" button at the bottom writes `completed_at` and flips status to `complete`.

**API:** `GET /v1/repairs/:id` (full repair card), `PATCH /v1/repairs/:id`, `POST /v1/repairs/:id/steps/:step_id/complete`, `PATCH /v1/repairs/:id/parts/:part_id`

### Repair history
Searchable, filterable log per vehicle. System filter chips at the top (All / Engine / Suspension / Brakes / etc.). Expandable row on tap shows cost breakdown, torque specs used, and any notes. An in-progress banner sits above the completed list for any repair with `status = in_progress`.

**API:** `GET /v1/vehicles/:id/history`

---

## Screens still to design

- Add / edit vehicle flow
- Active repair tracking state — `in_progress` view with photo capture and partial step completion
- Multi-source search results panel (Phase 4)
- Auth screens — sign up, login, role selection (owner / technician / fleet manager)
- Fleet / team views (Phase 6)

---

## State management

- **Server state:** React Query — all API data, cache invalidation on repair mutations
- **Local UI state:** Zustand — active repair session, optimistic step/part checkbox updates
- **Optimistic updates:** Step completions and part gathered toggles update immediately in the UI, sync in background. The repair card should never feel slow.

---

## Feature tiers

The repair card is designed with two audiences in mind: amateur DIYers who need guidance, and professionals who know what they're doing and care more about the financial side.

### Free tier (all users)
- Full recipe card: parts checklist with OEM numbers and prices, tools list, torque specs, step-by-step procedure
- Repair history per vehicle
- Photo documentation
- Multi-source search and resource saving (YouTube, forums, PDFs, parts pricing)
- Tutorial and resource panel

### Pro tier (shops & fleets — not yet scoped)

The key insight: **the data is already being collected**. `unit_cost`, `total_parts_cost`, step `completed_at` timestamps, and `mileage_at_repair` are all in the schema now. The pro tier is a display and permissions layer, not a schema change.

Planned pro features:
- **Streamlined repair mode** — collapses beginner guidance (tool notes, step descriptions), surfaces cost tracking and labor time front-and-center. Toggled by role or subscription, not a separate app.
- **Labor time logging** — start/stop timer per repair, stored against the repair record
- **Parts cost markup** — apply shop margin to part costs, exportable as a line-item invoice
- **Fleet dashboard** — multi-vehicle view across an org, open repairs by technician
- **Team accounts** — technician assignment, fleet manager overview (built on mechatracker's own `organizations` + `users` tables; one `users` row per Google account, scoped to one org, with role `owner`/`admin`/`member` presented as Owner / Manager / Technician in the UI)
- **Analytics** — cost per repair, parts spend over time, most common repairs per vehicle. Fleet managers love this.

**Implementation note:** Pro tier is gated by `user.role` and/or a subscription flag (Stripe, Phase 6). The streamlined repair mode is a view toggle on the existing repair card component — not a separate route or separate data model. No feature should be built without the gate in place first to avoid accidental free access.

---

## Mobile-specific considerations

These apply to the React Native / Expo app and don't affect the web build:

- **Camera:** Expo Camera for in-repair photo capture. Photos upload directly to storage via presigned URL from `POST /v1/uploads/presign` — the API never proxies file bytes.
- **Offline:** Step completions, part gathered toggles, and notes queue locally (Expo SQLite) and sync on reconnect. Repair card data is cached for offline reading.
- **Push notifications:** Expo Notifications for parts arrival alerts and repair reminders (Phase 5).
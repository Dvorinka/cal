# Self-Hosted Minimal Planner — Development Plan

## 1. Vision

Build a:
- self-hosted
- modern
- minimal
- PWA-first
- calendar + task planner

focused on:
- daily organization
- tasks
- notes
- links
- recurring reminders
- optional holidays

NOT:
- enterprise collaboration
- email/calendar federation
- Outlook clone
- Google Workspace replacement

---

# 2. Inspiration & Resources

## UI / Design Inspiration

- Skiff Apps Repo  
  https://github.com/skiff-org/skiff-apps

- Skiff UI Components  
  https://github.com/skiff-org/skiff-ui

- Skiff Calendar Landing  
  https://skiff.com/calendar

- Linear  
  https://linear.app

- Notion Calendar  
  https://www.notion.so/product/calendar

- Raycast  
  https://www.raycast.com

---

# 3. Tech Stack

## Frontend
- React
- Vite
- TailwindCSS
- TypeScript
- PWA support
- Zustand
- Framer Motion

## Backend
- Go
- Fiber or Echo
- PostgreSQL
- JWT auth

## Infrastructure
- Docker
- Docker Compose
- Cloudflare Tunnel
- GitHub Actions

---

# 4. MVP Scope (IMPORTANT)

## ONLY build this first:

### Calendar Views
- Month view
- Week view
- Day view

### Entries
- Tasks
- Notes
- Links

### Features
- Mark complete
- Drag/drop between days
- Search
- Tags/colors
- Dark mode
- PWA install
- Offline cache

### Holidays
- Country-based holidays
- Optional toggle
- Read-only entries

---

# 5. DO NOT BUILD YET

Avoid:
- CalDAV
- Email sync
- RSVP
- Team collaboration
- External inbox integration
- Video conferencing
- Native apps
- AI features
- Shared calendars
- E2EE

These massively increase complexity.

---

# 6. Recommended Architecture

## Frontend Structure

```txt
apps/web/
├── src/
│   ├── components/
│   ├── pages/
│   ├── layouts/
│   ├── stores/
│   ├── hooks/
│   ├── services/
│   ├── types/
│   ├── utils/
│   └── styles/
```

---

## Backend Structure

```txt
apps/api/
├── cmd/
├── internal/
│   ├── auth/
│   ├── calendar/
│   ├── tasks/
│   ├── holidays/
│   ├── db/
│   └── middleware/
├── migrations/
└── pkg/
```

---

# 7. Database Design

## Users

```sql
users
- id
- email
- password_hash
- created_at
```

## Entries

```sql
entries
- id
- user_id
- title
- content
- type (task/note/link)
- link_url
- date
- completed
- color
- created_at
```

## Settings

```sql
settings
- user_id
- country
- show_holidays
- theme
```

---

# 8. Holiday System

## Recommended APIs / Sources

### Static libraries
- https://github.com/commenthol/date-holidays

### ICS holiday feeds
- https://www.officeholidays.com/ics

Use:
- cached yearly imports
- NOT live requests every page load

---

# 9. PWA Setup

## Features
- Installable
- Offline support
- Cached assets
- Push notifications later

## Tools
- https://github.com/vite-pwa/vite-plugin-pwa

---

# 10. UI Design Direction

## Principles
- Minimal
- Fast
- Smooth
- Dark-mode-first
- Keyboard friendly

## Design Notes
- soft shadows
- rounded cards
- muted colors
- subtle animations
- low visual noise

---

# 11. Suggested Development Order

## Phase 1 — Setup
- Create monorepo
- Setup frontend
- Setup Go backend
- Docker compose
- PostgreSQL

---

## Phase 2 — Auth
- Login/register
- JWT
- Sessions

---

## Phase 3 — Core Calendar
- Month grid
- Day modal/sidebar
- Create task
- Save task

---

## Phase 4 — UX
- Drag/drop
- Animations
- Search
- Dark mode
- Mobile responsiveness

---

## Phase 5 — PWA
- Offline support
- Install prompt
- IndexedDB cache

---

## Phase 6 — Holidays
- Country selector
- Holiday imports
- Render holiday labels

---

## Phase 7 — Polish
- Keyboard shortcuts
- Quick-add
- Better transitions
- Widgets

---

# 12. Suggested Branding Direction

Possible naming direction:
- minimal
- short
- modern

Examples:
- Daygrid
- Plana
- Dailo
- Calyx
- Fokus
- Talo
- Dayflow
- Luma
- Timra

---

# 13. Hosting Recommendation

## VPS
- Hetzner
- Contabo
- OVH

## Deployment
- Docker Compose
- Coolify
- Railway (development)

---

# 14. Suggested MVP Goal

Your MVP should feel like:

> “The self-hosted daily planner people actually enjoy opening.”

Not:
> “open-source Google Calendar clone.”

That difference is the entire project direction.

---

# 15. Long-Term Features (Optional)

Later:
- recurring tasks
- markdown notes
- attachments
- public sharing
- widgets
- notifications
- ICS export/import
- shared calendars

Still avoid:
- full CalDAV stack
- enterprise mail sync
- Exchange compatibility

unless the project becomes very large.
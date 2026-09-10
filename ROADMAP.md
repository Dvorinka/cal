# Cal — Roadmap

Where this is going: the self-hosted daily planner people actually enjoy
opening. Local-first-feeling, agent-native, private. This file is the contract
for what ships and in what order. Each phase lists what "done" means.

Status legend: `[x]` shipped · `[/]` partially shipped · `[ ]` planned

## Phase 0 — Foundation (done)

- [x] Monorepo: React+Vite+TS web, Go+Gin API, Postgres, shared api-client
- [x] Cookie auth, sessions, rate limiting
- [x] Month/week/day views, drag & drop, now-line, mini-month
- [x] Tasks, notes, links, events with colors, tags, times, recurrence
- [x] Holidays engine (47 countries, rule-based)
- [x] Command palette, keyboard shortcuts, toasts with undo
- [x] PWA: manifest, icons, offline cache, offline session fallback
- [x] Docker Compose deployment
- [x] Original visual identity (warm paper, forest green, Fraunces+Inter)

## Phase 1 — Depth (done)

- [x] Multi-page app: Today / Calendar / Tasks / Notes / Links / Settings
- [x] Natural-language quick add (`dentist fri 5pm #health`)
- [x] ICS feed subscriptions + .ics import + own ICS parser (RRULE, EXDATE)
- [x] Reminders (per-entry lead time), background feed sync
- [x] MCP endpoint (10 tools: list/create/update/delete/search entries,
      append_note, today, feeds, accounts)
- [x] Read-only today widget for dashboards (`/widget/today?token=`)
- [x] Android: Capacitor shell, custom icon, debug + signed release APK
- [x] CalDAV two-way sync (collection URL, basic auth, tombstones,
      15-min background sync) — verified against Radicale
- [x] Web push for reminders when the tab is closed (VAPID, sub cleanup)
- [x] Markdown notes: write/preview editor, word count, safe custom renderer

## Phase 2 — Trust (shipped 2026-09-10)

- [x] **Backup & restore** — `POST /api/restore` merges an export file back
      (ID-conflict-safe, additive); nightly `DATA_DIR/backups/` snapshots,
      14 days kept; Restore button in Settings → Your data.
- [x] **Entry history** — `entry_revisions` snapshot on every update;
      editor "History" pane lists and restores versions (restores are
      themselves snapshotted, so revertible).
- [x] **CalDAV conflict surface** — divergence keeps both: remote wins the
      synced object, the local edit survives as a detached `conflict` copy.
- [x] **Timezone** — per-user IANA timezone; reminders evaluate in it, the
      .ics export serializes timed entries in it. (VTIMEZONE blocks are
      future work — times export as UTC, which every client accepts.)
- [x] **Real .ics export feed** — `GET /api/feed.ics?token=` (widget token,
      read-only) publishes tasks+events with RRULE + STATUS:COMPLETED.
- [ ] **Push E2E test** — needs a live push service; store-level tests
      cover the subscription lifecycle, the last hop is vendor infra.
- [x] **Security panel** — change password (revokes other sessions),
      session list with per-device revoke; sessions carry user-agent +
      last-seen.
- [x] **MCP resources & prompts** — `cal://today`, `cal://week`,
      `cal://open-tasks` resources; `daily-plan` + `weekly-review` prompts.

## Phase 3 — Delight (shipped 2026-09-10)

- [x] **Focus mode** — Today strips to a Now/Next card with minutes-left
      plus the open-task checklist.
- [x] **Weekly review** — `GET /api/review/week` computes done/slipped/
      notes/streak/busiest-day; Today card + "Save as note" seeds a
      next-week-focus template.
- [x] **Habit streaks** — `GET /api/habits` walks consecutive completed
      periods per recurring `#habit` task; flame chips on Today.
- [x] **Journal template** — one-tap Journal button seeds morning/
      evening/gratitude prompts and opens the note.
- [x] **Weather strip** — Open-Meteo (keyless, geocoded from the city
      setting) on the Today header; WMO-code icons, 30-min cache.
- [x] **Attachments** — `POST /api/files` (20 MB cap, random names,
      per-user dirs), markdown `![]()` inserts; images render in preview.
- [x] **Link unfurl** — `GET /api/unfurl?url=` title/description/favicon,
      SSRF-guarded (http(s) only, private IPs refused incl. redirects).
- [x] **Print stylesheet** — chrome hidden, agenda/panels print clean.

## Phase 4 — Mobile & on-the-go

- [ ] **Android release signing in CI** — keystore from secrets, attach APK
      to GitHub releases.
- [ ] **Android home-screen widget** — native "today" widget reading the
      widget endpoint.
- [ ] **Offline write queue** — edits made offline queue and replay
      (currently reads are cached; writes need a real queue).
- [ ] **Share target** — Android "share to Cal" creates a link/note entry.
- [ ] **iOS wrapper** — same Capacitor shell; TestFlight later.
- [ ] **Multi-account push** — per-device subscription labels, per-device
      mute.

## Phase 5 — Integrations (depth)

- [ ] **OAuth for Google Calendar** — the only mainstream two-way provider
      requiring OAuth; optional self-hosted client-id config.
- [ ] **CardDAV contacts** — birthday/anniversary events from contacts.
- [ ] **Email → task** — `inbound@` webhook (documented Mailgun/SMTP-pipe
      recipes) so forwarding a mail creates a task.
- [ ] **Webhooks out** — fire on entry create/complete for n8n/Home
      Assistant.
- [ ] **CalDAV collection discovery** — PROPFIND walk so users paste the
      account root and pick a calendar.
- [ ] **RSS/Atom feeds** — subscribe blogs as dated link entries.

## Phase 6 — Polish & hardening

- [ ] **E2E test suite** — Playwright specs for the six core flows
      (auth, create, drag, sync, offline, export).
- [ ] **Load tests** — k6 against a 10k-entry account.
- [ ] **a11y audit** — axe-core pass, full keyboard-only traversal, screen
      reader labels on every interactive control.
- [ ] **Rate-limit everything sensitive** — MCP token endpoint, feed create
      (SSRF surface), import size caps.
- [ ] **Security headers** — CSP, COOP/COEP where compatible with Capacitor.
- [ ] **Per-user storage quota** — configurable, shown in settings.

## Deliberately out of scope

- Multi-user sharing / shared calendars (it's a personal planner).
- Real-time collaboration.
- E2E-encrypted storage (self-hosted already isolates; revisit if multi-user
  ever lands).
- In-app email (Skiff tried; it killed them).

## Principles the roadmap answers to

1. **Offline is non-negotiable** — every feature degrades gracefully.
2. **Agents are first-class users** — anything a human can do, MCP can do.
3. **Your data is yours** — export always, no lock-in formats.
4. **Small dependency surface** — hand-rolled where sane (ICS, markdown,
   push plumbing); the repo should be auditable in an afternoon.

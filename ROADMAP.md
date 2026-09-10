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

## Phase 2 — Trust (next)

The product works; this phase makes it dependable.

- [ ] **Backup & restore** — JSON export exists; add one-click import that
      re-keys entries, plus a nightly auto-backup file on disk.
- [ ] **Entry history** — `entry_revisions` table; editor "History" pane to
      view/restore previous versions of a note.
- [ ] **Conflict surface for CalDAV** — when both sides changed, keep both
      (duplicate with `conflict` tag) instead of silent remote-wins.
- [ ] **Timezone support** — store events in UTC + display TZ; per-user
      timezone setting; correct `VTIMEZONE` in ICS export.
- [ ] **Real .ics export feed** — subscribable `/api/feed.ics?token=` of your
      own Cal entries (the inverse of feed import).
- [ ] **Push E2E test** — headless-browser push delivery check in CI
      (web-push needs a real service; test against Mozilla's autopush dev).
- [ ] **Password change + session list** — settings security panel: change
      password, see/revoke active sessions.
- [ ] **MCP resources & prompts** — expose entries as MCP resources (not just
      tools), plus `daily-plan` / `weekly-review` prompt templates.

## Phase 3 — Delight

The reason to open it every morning.

- [ ] **Focus mode** — Today page strips to the next block + checklist;
      ambient timer for the current event.
- [ ] **Weekly review page** — auto-digest: what got done, what slipped,
      streaks; generated as a markdown note each Sunday.
- [ ] **Habit tracker** — tasks tagged `#habit` get streak dots and a
      "don't break the chain" strip on Today.
- [ ] **Journal template** — one-tap daily note seeded with prompts
      (gratitude / top-3 / tomorrow's plan).
- [ ] **Weather strip** — optional Open-Meteo (no key) forecast on Today.
- [ ] **Attachment support** — images/files on notes, stored under
      `DATA_DIR/uploads`, rendered in preview.
- [ ] **Link unfurl** — fetch `<title>`/favicon for link entries (opt-in,
      server-side fetch with SSRF guard: http(s) only, no private ranges).
- [ ] **Print stylesheet** — week/day views print cleanly.

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

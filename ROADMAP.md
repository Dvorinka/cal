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

## Phase 4 — Mobile & on-the-go (shipped 2026-09-10)

- [x] **Android release signing in CI** — `.github/workflows/android.yml`:
      keystore from secrets, tag builds attach a signed APK artifact.
- [x] **Android home-screen widget** — `CalWidgetProvider` renders today's
      agenda from `/api/widget/today`; the app pushes server+token via the
      WidgetConfig Capacitor plugin.
- [x] **Offline write queue** — create/update/delete queue in localStorage
      (tmp-ids, op coalescing, replay on `online`), pending pill in the topbar.
- [x] **Share target** — PWA `share_target` → `/share` prefills the editor
      (URL → link, text → note); Android `ACTION_SEND` intent deep-links the
      same route.
- [x] **iOS wrapper** — `apps/web/ios/` scaffolded; needs macOS + Xcode to
      build (no CI lane yet).
- [x] **Per-device push management** — subscriptions get labels (derived from
      user-agent), listed in Settings with per-device revoke.

## Phase 5 — Integrations (shipped 2026-09-10)

- [x] **OAuth for Google Calendar** — `GOOGLE_CLIENT_ID`/`_SECRET` env, consent
      → callback → refresh-token sync; events land as a "Google" feed (read-only,
      RFC 5545 conversion server-side, re-syncs every 15 min).
- [x] **CardDAV contacts** — addressbook REPORT → vCard FN+BDAY → yearly
      all-day "X's birthday" events tagged `birthday`.
- [x] **Email → task** — `POST /api/intake?token=` accepts `{subject, text}`;
      any forwarding recipe (procmail→curl, Mailgun route) lands an `inbox` task.
- [x] **Webhooks out** — `POST {event, entry}` on create/update/delete,
      HMAC-SHA256 in `X-Cal-Signature`.
- [x] **CalDAV collection discovery** — `POST /api/caldav/discover` walks
      principal → calendar-home-set → collections; Settings offers a picker.
- [x] **RSS/Atom feeds** — items convert to ICS at fetch time and ride the
      normal feed pipeline as dated, read-only entries.

## Phase 6 — Polish & hardening (in progress)

- [x] **E2E test suite** — Playwright specs in `apps/web/e2e`: auth, CRUD,
      navigation, palette, settings persistence.
- [x] **Load test** — `perf/k6.js`: 847 iterations, 0 failures, p95 156ms.
- [x] **a11y audit** — axe-core specs on Today/Tasks/Settings; contrast and
      focusability violations fixed (text-3/4 tokens, scrollable `<pre>`).
- [x] **Security headers** — CSP + X-Frame-Options + nosniff + Referrer-Policy
      in `apps/web/nginx.conf`.
- [x] **Rate-limit sensitive routes** — files 20/min, feeds 10/min, MCP 60/min,
      intake 10/min, per-IP on top of auth limits.
- [x] **Per-user storage quota** — `settings.quota_mb` enforced on upload,
      usage shown in Settings.

## Phase 7 — Depth (shipped 2026-09-10)

- [x] **Wiki-links** — `[[Note title]]` in markdown renders as a link that
      opens the note; the editor shows a "Linked from" backlinks row.
- [x] **Entry deep links** — `/entry/:id` opens the editor on that entry;
      palette/MCP-generated links can point straight at an entry.
- [x] **VALARM import** — `.ics` events with alarms keep their reminder as
      `remind` minutes on import.
- [x] **Webhook SSRF + retry** — registration and delivery refuse private
      IPs (env opt-out for dev); one retry after 30s on transport/4xx-5xx.
- [x] **Feed SSRF** — same guard on `.ics`/RSS subscription fetches (was
      previously unchecked — real hole, now closed).
- [x] **Files** — uploads now land in a `files` table; Files page with
      drag-drop, type icons, image preview, per-file public share links
      (`/api/shared/files/<token>`), delete. Self-hosted file bin.
- [x] **Memos-flavored notes** — stream view (Today/Yesterday/date groups),
      pinned notes (`pinned` column, pin toggle, sorted first), inline `#tag`
      chips extracted from content + tag filter row.
- [x] **Activity heatmap** — `GET /api/activity` date→count map; 17-week
      contributions grid on Today.
- [x] **MCP `list_files`** — 14 tools.
- [x] **Kanban boards** — `boards`/`board_columns` tables; cards are real
      task entries (`board_id`/`column_id`/float `position`), so board work
      appears on the calendar, Today, and Tasks. Drag between columns
      (fractional positions), inline card add, column add/delete, board
      list page, editor board/column picker. `POST /api/cards/:id/move`.
- [x] **MCP board tools** — `list_boards`, `board_view`, `create_card`,
      `move_card` — 18 tools.
- [x] **PM depth** — due-date chips on cards (overdue red, today accent),
      checklist progress from `- [ ]` content, WIP limits per column
      (`n/limit` badge, red when over), done-column convention (drop into
      done/completed/shipped auto-completes, dragging out reopens), board
      templates (blank/kanban/sprint/bug-tracker seed columns).

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

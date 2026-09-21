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
- [x] **Security headers** — CSP + X-Frame-Options + nosniff + Referrer-Policy,
      set by the Go SPA handler (`apps/desktop/app.go`).
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
- [x] **Focus timer** — `time_entries`, one running per user (partial
      unique index); sidebar pill ticks live, start from a task's context
      menu, Today shows today/week totals + per-entry breakdown.
- [x] **Card activity log** — `card_activity` records created/moved/
      completed/reopened/renamed/edited on board cards; an Activity pane
      in the editor.
- [x] **Trash** — `deleted_at` soft-delete; `/trash` restores or purges.
- [x] **Tags page** — `/tags` counts every tag across types; click →
      cross-type filtered list.
- [x] **Bulk ops** — Tasks select mode → complete-all / delete-all.
- [x] **Public boards** — `POST /boards/:id/share` mints a token;
      `/board/<token>` renders columns+cards with no auth.
- [x] **Board meta** — description + target date per board.
- [x] **Agenda export** — `GET /api/agenda?days=N` markdown; copyable
      from the palette or Settings.
- [x] **Note templates** — meeting notes / standup / decision log /
      project brief chips in an empty note editor.
- [x] **Timesheet** — `/time` groups sessions by day with totals and
      delete; pomodoro mode (`planned` minutes → countdown + notify).
- [x] **Daily digest push** — `digest_time` in Settings; the push loop
      sends one morning summary (open tasks + events) per day.
- [x] **Board progress** — done/total chip on every board tile.
- [x] **Billable time** — rate + project on sessions, `$` chips, default
      rate in Settings, solidtime-shaped CSV/JSON export.
- [x] **Rich link cards** — unfurl + YouTube oEmbed on save; cards with
      thumbnails, favicon list view, watched toggle on videos.
- [x] **Global search** — `/api/search` across entries + files; the
      palette returns both.
- [x] **GitHub** — PAT in Settings; `/github` inbox groups open
      issues+PRs by repo, one-click import to a board, weekly event
      count; completing a linked card closes the issue; a 15-min loop
      closes cards whose issues were closed remotely and moves them to
      the done column.

## Phase 8 — Trackeep merge (shipped 2026-09-11)

Single-user reinterpretation of Trackeep's team features, plus the
solidtime-style timesheet depth and a mail module.

- [x] **Workspaces** — `workspaces` table; nullable scope on entries and
      files (`NULL` = Personal). Sidebar switcher: All / Personal / named
      space. Settings → Workspaces manages them.
- [x] **Feature modules** — `settings.modules` JSON map; every section
      (tasks/boards/notes/links/files/time/mail/github/tags) toggles off
      and disappears from nav, routes, palette and settings.
- [x] **Dashboard** — `GET /api/dashboard` rolls up task counts, this
      week's completions, upcoming deadlines, per-day activity and a
      merged activity feed (entries/cards/files); Today renders deadlines
      (with days-left chips), recent activity and storage usage.
- [x] **Entry dependencies** — `blocked_by`; a blocked task can't be
      completed (409), shows a "blocked" chip on boards and in list view.
- [x] **Time-entry tags, manual entries, editing** — `POST /time/log`,
      `PATCH /time/log/:id`; tags filter the timesheet; sessions editable.
- [x] **File tags + workspace** — `PATCH /files/:id`, tag chips and
      filtering on the Files page, upload honors the active space.
- [x] **Saved filters** — `GET/POST /filters`; Tasks page applies them
      as chips (click to apply, right-click to delete).
- [x] **Link preview refresh** — `POST /entries/:id/refresh-link`.
- [x] **Mail** — IMAP/SMTP accounts with AES-256-GCM credentials;
      mailboxes, message list/reader, flags, delete, compose; `/mail`.
- [x] **Board list view** — boards toggle kanban ↔ column-grouped list.
- [x] **Board time sums** — `minutes` on board tiles.
- [x] **Desktop app** — `apps/desktop`: Wails 2 shell, API in-process,
      SPA embedded; embedded Postgres when `DATABASE_URL` unset (turnkey);
      `-tags headless` variant for servers/CI; app icon in build/.
- [x] **Channel feeds → link cards** — a feed kind "links" turns RSS/Atom
  items (e.g. `youtube.com/feeds/videos.xml?channel_id=…`) into real link
  entries with `media:thumbnail` cards; dedupes by URL, runs in the sync
  loop. Calendar feeds keep landing as dated events.
- [x] **Editable shared boards** — `share_edit` write tier: the public
  link gets a per-card "Move to" picker; view-only default unchanged.
- [x] **Mobile nav audit** — bottom nav now shows the first four enabled
  module tabs + a More tab that opens the full sidebar drawer (Files,
  Boards, Time, Mail, settings, trash all reachable on phones).
- [x] **Local mode (Android)** — "Use on this device — mail only" on the
  login screen runs the app with no Cal server at all: Mail talks IMAP/SMTP
  straight to the provider via the `CalMail` Capacitor plugin, accounts are
  AES-256-GCM'd in Keystore-backed prefs, and a later server login offers to
  import the on-device accounts. MailPage runs through a `MailBackend`
  interface so server and native transports are interchangeable.

### Still excluded (per spec)

- Trackeep: messages, learning, in-app AI assistant (MCP/API instead).

## Phase 9 — PeopleVault merge (shipped 2026-09-17)

Ports the relationship-manager depth of `ref/PeopleVault` onto the existing
people module — migration `00013_people.sql` already ships CRUD, named
yearly `dates`, and calendar/Today occurrence rendering. PV's separate
events/reminders tables are not ported: `dates` JSONB already covers
recurring yearly dates, and reminders ride the existing entry pipeline.

- [x] **Profile depth** — extend `people`: nickname, avatar, phone, email,
      address, `gift_ideas`, `interests`, `is_favorite`; `fields` and `links`
      JSONB columns cover PV's `custom_fields` + `social_links` tables (same
      pattern as `dates`, no extra joins).
- [x] **Person tags** — `tags` text[] on people; counted on `/tags`
      alongside entries and files.
- [x] **Relationships** — `person_links(from_person_id, to_person_id, kind)`;
      relation chips on the profile. The data model is the family-tree
      foundation (parent/child/sibling/partner/friend/coworker/mentor).
- [x] **Timeline** — `person_timeline(person_id, type, title, body,
      occurred_on)`; chronological history on the person page (met, gift,
      trip, achievement, memory).
- [x] **Person attachments** — `files.person_id`; photos/docs on the profile,
      reusing the Files pipeline (quota, share tokens, type icons).
- [x] **Namedays** — port `internal/nameday` (nameday.abalin.net V2, 19
      countries) with the `data/namedays/*.csv` fallback (CZ/SK/PL/HU/AT/DE)
      and 24h cache; debounced `GET /api/namedays/search` in the person
      editor writes a named `dates` entry, so namedays render on the
      calendar and Today for free. `settings.nameday_country` sets the
      default lookup country.
- [x] **Date reminders** — optional `remindDays` per `dates` entry,
      evaluated by the existing reminder/push/digest loops ("Mum's birthday
      in 7 days" as a push, not just a calendar chip).
- [x] **People in global search** — `GlobalSearch` covers name, nickname,
      relation, notes, tags; pg_trgm index on `people(name, nickname)`;
      palette results get a person section.
- [x] **MCP people tools** — `list_people`, `create_person`,
      `update_person`, `person_upcoming`. Agents stay first-class.
- [x] **Contact import** — the CardDAV vCard parser already exists; "import
      as people" turns an address book into person records (birthday →
      `dates`), upgrading today's read-only birthday events into real
      linked profiles.
- [x] **Wider holiday coverage** — the rule engine (47 countries, offline)
      stays primary; optional `date.nager.at` browse covers the other ~100
      countries with import-as-entry, PV-style.
- [x] **Today depth** — milestone chips ("turns 30"), a recently-added
      strip; nameday occasions ride the existing upcoming-dates card.
- [x] **Person profile page** — `/people/:id` carries the depth surface:
      relations both directions, timeline CRUD, attachments, upcoming
      dates, full-depth editing.
- [x] **CardDAV account management** — `GET /api/carddav` lists saved
      connections; Settings rows get sync, delete, and import-as-people.
- [x] **People-aware export & backups** — `/api/export`, `POST /api/restore`
      and the nightly snapshots now carry `people`, `personLinks` and
      `personTimeline`, additive and FK-guarded like entries.

### Still excluded (per spec)

- PeopleVault: multi-tenant `owner_user_id` model (Cal is single-user with
  workspace scoping), onboarding wizard, `audit_log`, the standalone
  events/reminders tables (absorbed into `dates` + existing reminders).

## Phase 10 — Parity & portability (shipped 2026-09-18)

The leftover "buildable now" gaps: agents reach the full people surface,
export carries real files, YouTube search works without a Google key, and
the last API-only features get UI handles.

- [x] **MCP people parity** — `delete_person`, `list_person_relations`
      (every link, or one person's both directions), `link_people`,
      `unlink_people`, `person_timeline`, `add_timeline_item`,
      `update_timeline_item`, `delete_timeline_item`. Same validation as
      the HTTP handlers (relation kinds, timeline types, no self-links,
      ownership) — 34 tools total. Principle 2 holds.
- [x] **Files in export** — `GET /api/export?format=zip` streams
      `cal-export.json` + `files/<name>` binaries; the plain JSON gains a
      `files` manifest array. `POST /api/restore` sniffs the zip magic and
      unpacks binaries additively: a file row is restored only when its
      binary is in the archive (no dangling attachments), and share
      tokens/person/workspace FKs are kept only when still valid locally.
      Nightly backups carry the manifest only — binaries already live in
      `uploads/` and are not duplicated 14×.
- [x] **YouTube search via Invidious** — `settings.invidious_url` +
      `GET /api/youtube/search` proxies the instance's `/api/v1/search`.
      The Links page gets a YouTube panel whose results save as link
      entries (oEmbed enriches on create, as usual). No Google key; the
      instance is user-configured like a CalDAV URL, so LAN hosts are
      allowed — the SSRF guard stays for attacker-influenced URLs only.
- [x] **Discoverability pass** — `POST /entries/:id/refresh-link` finally
      has a button (link editor → Preview); the palette gains Go-to
      actions for time tracking, GitHub, trash and settings.

## Remaining — buildable now

- [x] **YouTube video search** — shipped in Phase 10 via a configurable
      Invidious instance (no Data API key needed).
- [x] **Family tree view** — `/people/tree` renders an SVG generation graph
      over `person_links` (parent/child rows, partner/sibling arcs, other
      kinds dashed).
- [x] **Password reset email** — PeopleVault's forgot/reset flow, sent via
      the Mail module's configured SMTP account instead of a new provider.
- [x] **MCP person relations & timeline tools** — shipped in Phase 10.
- [x] **Files in export** — shipped in Phase 10 (zip archive + additive
      restore of binaries).

## Remaining — UX/discoverability (found in the visual pass)

- [x] ~~Files page header was unstyled~~ — now uses the shared
      `PageHeader` + `page-scroll` like every other page.
- [x] ~~Timer start was buried in a context menu~~ — Time header gets
      "Start timer", Today gets a Focus chip.
- [x] **Discoverability audit** — Phase 10: refresh-preview button for
      links (the endpoint existed API-only), palette Go-to for time,
      GitHub, trash and settings. Every page route is now in the palette.
- [x] **Mobile nav audit** — shipped in Phase 8: bottom nav + More tab
      opens the full sidebar drawer, everything reachable on phones.

## Remaining — needs something external

- [ ] **GitHub PAT** — the inbox, import, and sync loop are built and
      verified up to the token check; populate `settings.github_token`
      to exercise them end-to-end.
- [ ] **YouTube Data API key** — optional now; the Invidious path shipped
      in Phase 10 covers search without a Google key.
- [ ] **Google OAuth client pair** — the Google Calendar path is
      complete; needs real `GOOGLE_CLIENT_ID`/`_SECRET` to run consent.
- [ ] **iOS build** — `apps/web/ios/` is scaffolded; needs a Mac +
      Xcode. TestFlight after that.
- [ ] **Inbound email plumbing** — `/api/intake` works; wiring real
      email needs SendGrid/SES inbound (or a forwarding rule hitting
      the endpoint with the intake token).

## Future — candidates for the next phases

### Product

- [x] **Restore preview** — `POST /api/restore?dry=1` reports per-collection
      new/existing/invalid/orphaned counts and file rows lacking binaries;
      Settings shows the summary and asks before restoring.
- [x] **Upload dedup** — `files.sha256` (migration 0018): a second upload of
      an identical binary returns the original row (`deduped: true`), and
      restore hardlinks an on-disk twin instead of unpacking a duplicate.
- [x] **MCP resources for people** — `cal://people` lists everyone,
      `cal://people/<id>` returns profile + relations + timeline + files;
      declared via `resources/templates/list`.
- [x] **Public person pages** — `POST /api/people/share` mints a token;
      `/people/shared/<token>` is a no-auth birthday/date list and
      `/api/shared/people/<token>/calendar.ics` a yearly-recurring ICS feed.

### Engineering

- [x] **DB-backed API tests** — `testdb_test.go` boots embedded Postgres
      (or `CAL_TEST_DATABASE_URL`), migrates it, and covers restore dry-run,
      upload dedup, MCP person resources and shared people end-to-end.
- [x] **One installer** — npm workspaces is canonical; `@cal/api-client` is a
      plain `"0.1.0"` workspace dep, no `file:` snapshot to drift.
- [x] **Enrichment status** — `entries.link_meta_at` stamps every enrichment
      attempt (failures included); the Links UI shows a pulsing "Fetching…"
      chip until it lands instead of guessing.
- [x] **Streaming restore** — the request body spools to a temp file and the
      zip reads lazily via `io.ReaderAt`; big archives no longer sit in RAM.

## Deliberately out of scope

- Multi-user sharing / shared calendars (it's a personal planner).
- Real-time collaboration.
- E2E-encrypted storage (self-hosted already isolates; revisit if multi-user
  ever lands).

## Principles the roadmap answers to

1. **Offline is non-negotiable** — every feature degrades gracefully.
2. **Agents are first-class users** — anything a human can do, MCP can do.
3. **Your data is yours** — export always, no lock-in formats.
4. **Small dependency surface** — hand-rolled where sane (ICS, markdown,
   push plumbing); the repo should be auditable in an afternoon.

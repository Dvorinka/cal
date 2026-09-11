# Cal

The self-hosted daily toolkit people actually enjoy opening.

[![CI](https://github.com/Dvorinka/cal/actions/workflows/ci.yml/badge.svg)](https://github.com/Dvorinka/cal/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Dvorinka/cal)](https://github.com/Dvorinka/cal/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Calendar, tasks, notes, links, files, kanban boards, time tracking and a GitHub
inbox in one quiet place — month, week and day views, recurring tasks, billable
sessions, link previews, global search, holidays for 40+ regions, dark mode,
keyboard-first, installable as a PWA. No accounts to create on someone else's
server; one `docker run` and it's yours.

![Month view](docs/screenshot-month.png)

![Week view, dark theme](docs/screenshot-week-dark.png)

## Features

- **Eleven pages** — Today, Calendar, Tasks, Notes, Links, Files, Boards, Tags,
  Time, GitHub, Settings — plus Trash and a public board view
- **Kanban boards** — cards are task entries (they land on the calendar too);
  drag between columns, WIP limits, done-column conventions, checklists,
  templates, public read-only sharing
- **Time tracking** — one-tap focus timer in the sidebar, pomodoro mode,
  billable sessions with hourly rates and per-project rollups,
  solidtime-shaped CSV/JSON export
- **File bin** — drag-drop uploads, per-type icons, image lightbox, public
  share links, storage quota
- **Link library** — saving a link unfurls title/favicon/og:image; YouTube
  URLs get thumbnails + channel via oEmbed (no API key); watched toggle,
  All/Videos/Articles filters
- **Global search** — `⌘K` hits entry titles, content, tags, link URLs, file
  names and board names in one pass
- **GitHub inbox** — PAT in Settings; open issues and PRs grouped by repo,
  import-to-board, weekly activity count; completing a linked card closes
  the issue, and a 15-minute loop completes cards when issues close remotely
- **Three real views** — month grid, week and day time grids with an all-day row
  and a live now-line
- **Tasks, notes, links, events** — with colors, tags, details and optional
  start/end times
- **Calendar feeds** — subscribe to any iCalendar URL (Google Calendar's secret
  address, iCloud public calendars, Nextcloud shared links, Outlook published
  calendars); external events render read-only beside your own
- **CalDAV two-way sync** — connect Nextcloud, Radicale, Baikal or Fastmail
  collections; events flow both directions on a 15-minute cycle, deletes
  propagate, credentials AES-encrypted at rest. Collection discovery walks the
  server for you — paste the account root, pick a calendar
- **Google Calendar** — OAuth connect (self-hosted client id/secret); events
  sync read-only into a toggleable "Google" feed every 15 minutes
- **CardDAV birthdays** — connect an addressbook; contacts with birthdays
  become yearly all-day events
- **RSS/Atom feeds** — subscribe to blogs and changelogs; items land on the
  calendar on their publish date (SSRF-guarded like the webhook URLs)
- **Soft-delete trash** — deletions recover for 30 days; restore or purge
- **Note templates + wikilinks** — meeting/standup/decision starters;
  `[[Note]]` links between notes with a backlinks row
- **Activity** — per-card audit trail (created/moved/completed/renamed),
  activity heatmap on Today, morning digest push at your chosen hour
- **Tags page** — every tag counted across tasks, notes, links and cards;
  one tap filters the whole toolkit
- **Webhooks out** — POST `entry.created|updated|deleted` to any URL, signed
  with HMAC-SHA256 for n8n/Home Assistant
- **Email → task** — `POST /api/intake?token=` accepts `{subject, text}`;
  point any mail-forwarding recipe at it
- **.ics import + export** — drop in a calendar file, or subscribe to
  `/api/feed.ics?token=` to read your own planner in any calendar app
- **Offline write queue** — edits made offline persist and replay in order on
  reconnect; a pending-ops pill shows the backlog
- **Share target** — PWA share_target + Android `ACTION_SEND` prefill the
  editor (URL → link, text → note)
- **Android home widget** — native today-agenda widget backed by the
  token-gated widget endpoint
- **Entry history** — every save snapshots the prior version; restore any of
  them from the editor
- **Web push** — VAPID-based push for reminders, fires even with the tab closed
- **Natural-language quick add** — `dentist fri 5pm #health` parses the date,
  time and tags on the Tasks page
- **Reminders** — per-entry lead times; browser notifications fire while the
  app is open
- **Recurring tasks** — daily, weekly, monthly, yearly; completing one spawns
  the next occurrence
- **Holidays** — rule-based engine (Gregorian and Orthodox Easter, nth-weekday
  rules) covering 40+ countries, toggled per user
- **MCP endpoint** — `POST /api/mcp` speaks Model Context Protocol; 22 tools
  (entries, timer, boards, files, GitHub inbox, habits, weekly review) so AI
  assistants can read and run your planner (bearer-token auth)
- **Embeddable Today widget** — `/widget/today?token=…` renders a minimal
  agenda for dashboards and iframes
- **Command palette** — `⌘K` / `Ctrl+K` to search everything and run actions
- **Keyboard-first** — `1/2/3` switch views, `t` today, `←/→` navigate, `c` new
  entry, `/` search
- **Drag & drop** — move entries between days, reschedule onto the time grid,
  drop into all-day
- **Themes + accents** — light, dark, system; five accent colors
- **Offline-first PWA + Android** — installable, entries cached locally; a
  Capacitor shell produces a native APK
- **Self-contained** — cookie sessions, bcrypt passwords, login rate limiting,
  embedded Postgres, zero external services

## Stack

- **Web** — React 19, Vite, TypeScript, Tailwind CSS 4, Zustand, Framer Motion,
  `vite-plugin-pwa`
- **API** — Go, Gin, PostgreSQL (pgx), cookie sessions, Goose migrations
- **Contracts** — `openapi.yaml` plus a hand-written TS client in
  `packages/api-client`
- **Desktop** — Wails 2 (native window + system webview); the same binary runs
  headless as the all-in-one server
- **Infra** — one Docker image: SPA + API + embedded Postgres in a single
  container

## Install

### Docker — all-in-one (recommended)

```bash
docker run -d --name cal -p 8080:8080 -v cal-data:/data ghcr.io/dvorinka/cal:latest
```

Open http://localhost:8080 and create your account. Everything — UI, API and an
embedded Postgres — runs in that one container; the data volume is all you
need to back up. Prefer an external database? Set `DATABASE_URL` and the
embedded one stays off:

```bash
docker run -d --name cal -p 8080:8080 -v cal-data:/data \
  -e DATABASE_URL="postgres://user:pass@host:5432/cal?sslmode=disable" \
  ghcr.io/dvorinka/cal:latest
```

Or with Compose (`docker-compose.yml` ships both variants, db service included
but commented out):

```bash
docker compose up -d --build
```

### Desktop app

Windows, Linux and macOS builds attach to each
[release](https://github.com/Dvorinka/cal/releases). The desktop app bundles
its own server (embedded Postgres under `~/.config/cal`); it can also sign in
to a server URL to share data with your other devices. Server binaries
(`cal-server-*`) are the same all-in-one the Docker image runs.

### Android

The release APK is on the releases page (debug-signed when no release keystore
is configured — sideload freely). It asks for your server URL on first launch.

### Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | listen port |
| `DATA_DIR` | `/data` (container) | files, backups, embedded PG data |
| `DATABASE_URL` | unset → embedded PG | external Postgres DSN |
| `SESSION_SECURE` | `false` | set `true` behind HTTPS |
| `WEB_ORIGIN` | unset | extra CORS origin for the web app |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | unset | enable Google Calendar sync |
| `CAL_ALLOW_PRIVATE_FEEDS` / `CAL_ALLOW_PRIVATE_WEBHOOKS` | unset | allow private/LAN URLs (SSRF guard off) |

## Development

```bash
npm install
cp apps/api/.env.example apps/api/.env

# Postgres (or use the compose db service)
docker run -d --name cal-db -e POSTGRES_DB=cal -e POSTGRES_USER=cal \
  -e POSTGRES_PASSWORD=cal -p 5432:5432 postgres:16-alpine

npm run dev -w @cal/web          # vite dev server on :5173, proxies /api
cd apps/api && go run ./cmd/server   # api on :8080
```

If the API runs on a different port, point the dev proxy at it:
`VITE_API_ORIGIN=http://localhost:8081 npm run dev -w @cal/web`

## Verify

```bash
npm run typecheck
npm run test
npm run build
cd apps/api && go vet ./... && go test ./...
```

## Android APK

The web app ships with a Capacitor shell. With an Android SDK installed:

```bash
cd apps/web
npm run android:build        # builds the PWA, syncs, produces an APK
# -> android/app/build/outputs/apk/debug/app-debug.apk
```

A signing config is wired for releases — drop your keystore at
`android/app/keystore/cal-release.jks` (gitignored) and run
`./gradlew assembleRelease` for a signed `app-release.apk`.

## MCP

`POST /api/mcp` is a streamable-HTTP MCP endpoint (initialize, tools/list,
tools/call). Generate or rotate a token in Settings → MCP / API access, then
point a client at it:

```json
{ "mcpServers": { "cal": { "url": "https://your-host/api/mcp",
    "headers": { "Authorization": "Bearer <apiToken>" } } } }
```

Tools: `list_entries`, `today`, `create_entry`, `update_entry`, `delete_entry`,
`list_feeds`, `get_entry`, `append_note`, `search_entries`, `list_accounts`.
Resources `cal://today`, `cal://week`, `cal://open-tasks`; prompts
`daily-plan`, `weekly-review`.

## Integrations & webhooks

- **Google**: set `GOOGLE_CLIENT_ID` + `GOOGLE_CLIENT_SECRET` on the API
  (Google Cloud → OAuth consent → `calendar.readonly` scope), then Settings →
  Google Calendar → Connect.
- **Webhooks**: Settings → Webhooks. Each POST body `{"event","at","entry"}`;
  verify `X-Cal-Signature` as hex HMAC-SHA256 of the raw body with the shown
  secret.
- **Email intake**: `curl -X POST 'https://host/api/intake?token=<apiToken>'
  -d '{"subject":"…","text":"…"}'` — a Mailgun route or `procmail | curl`
  recipe turns mail into tasks tagged `inbox`.

## Testing

`npm test` runs vitest units; `cd apps/web && npx playwright test` runs the
Playwright + axe E2E suite against the dev stack; `k6 run perf/k6.js -e
EMAIL=… -e PASS=…` load-tests the entry pipeline.

## Layout

```
apps/web          React PWA + Capacitor android/ + ios/ shells
apps/api          Go API + migrations + MCP endpoint
apps/desktop      Wails desktop app; `-tags headless` is the all-in-one server
packages/api-client   shared typed client (mirrors openapi.yaml)
```

## Releases & CI

Every tag `v*` builds and attaches: the GHCR image
(`ghcr.io/dvorinka/cal:<version>`, amd64 + arm64), desktop apps for
Windows/Linux/macOS, headless server binaries, and the Android APK. CI
typechecks, tests, vets, gofmt-checks, and builds + smoke-tests the Docker
image on every push and PR.

## API surface

Auth is a secure, HttpOnly session cookie (`SESSION_SECURE=true` in production).

| Endpoint | Purpose |
| --- | --- |
| `POST /api/auth/register` `/login` `/logout`, `GET /api/me` | session auth (login/register rate-limited) |
| `GET/POST /api/entries`, `PATCH/DELETE /api/entries/:id` | entries; `q`, `from`, `to` filters |
| `GET/PUT /api/settings` | theme, week start, holiday region + toggle, accent, tokens |
| `GET /api/export` | full JSON export (user, settings, entries) |
| `GET/POST /api/feeds`, `DELETE /api/feeds/:id`, `POST /api/feeds/:id/refresh` | ICS feed subscriptions |
| `GET /api/feed-events?from=&to=` | expanded external events |
| `POST /api/import` | import a .ics file |
| `GET/POST /api/caldav`, `DELETE /api/caldav/:id`, `POST /api/caldav/:id/sync` | two-way CalDAV accounts |
| `GET /api/push/vapid`, `POST /api/push/subscribe`, `POST /api/push/unsubscribe` | web push |
| `GET /api/widget/today?token=` | token-gated read-only agenda |
| `POST /api/mcp` | MCP endpoint (bearer `apiToken`) |
| `GET /api/holidays?country=&year=` | computed holidays |
| `GET /api/holidays/countries` | supported regions |

See `openapi.yaml` for the full schema.

## License

MIT — see `LICENSE`.

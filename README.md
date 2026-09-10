# Cal

The self-hosted daily planner people actually enjoy opening.

Calendar, tasks, notes and links in one quiet place — month, week and day views,
recurring tasks, holidays for 40+ regions, dark mode, keyboard-first, installable
as a PWA. No accounts to create on someone else's server; one `docker compose up`
and it's yours.

![Month view](docs/screenshot-month.png)

![Week view, dark theme](docs/screenshot-week-dark.png)

## Features

- **Six pages** — Today (agenda, checklist, progress), Calendar, Tasks (grouped, quick-add, tag filters), Notes, Links, Settings
- **Three real views** — month grid, week and day time grids with an all-day row
  and a live now-line
- **Tasks, notes, links, events** — with colors, tags, details and optional
  start/end times
- **Calendar feeds** — subscribe to any iCalendar URL (Google Calendar's secret
  address, iCloud public calendars, Nextcloud shared links, Outlook published
  calendars); external events render read-only beside your own
- **.ics import** — drop in a calendar file; events land as real entries,
  recurring rules expand
- **Natural-language quick add** — `dentist fri 5pm #health` parses the date,
  time and tags on the Tasks page
- **Reminders** — per-entry lead times; browser notifications fire while the
  app is open
- **Recurring tasks** — daily, weekly, monthly, yearly; completing one spawns
  the next occurrence
- **Holidays** — rule-based engine (Gregorian and Orthodox Easter, nth-weekday
  rules) covering 40+ countries, toggled per user
- **MCP endpoint** — `POST /api/mcp` speaks Model Context Protocol so AI
  assistants can read and manage your planner (bearer-token auth)
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
  Postgres, zero external services

## Stack

- **Web** — React 19, Vite, TypeScript, Tailwind CSS 4, Zustand, Framer Motion,
  `vite-plugin-pwa`
- **API** — Go, Gin, PostgreSQL (pgx), cookie sessions, Goose migrations
- **Contracts** — `openapi.yaml` plus a hand-written TS client in
  `packages/api-client`
- **Infra** — Docker Compose: Postgres + API + Nginx static web

## Quick start

```bash
docker compose -f infra/docker-compose.yml up --build
```

Web: http://localhost:5173 — API: http://localhost:8080

Ports are overridable when they collide with other stacks:
`DB_PORT=5434 API_PORT=8082 WEB_PORT=5273 docker compose -f infra/docker-compose.yml up --build`

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

The shell wraps the same SPA — the API URL is relative, so the APK works
against whatever origin serves it. Release builds need a signing config
(`./gradlew assembleRelease` after adding a keystore).

## MCP

`POST /api/mcp` is a streamable-HTTP MCP endpoint (initialize, tools/list,
tools/call). Generate or rotate a token in Settings → MCP / API access, then
point a client at it:

```json
{ "mcpServers": { "cal": { "url": "https://your-host/api/mcp",
    "headers": { "Authorization": "Bearer <apiToken>" } } } }
```

Tools: `list_entries`, `today`, `create_entry`, `update_entry`, `delete_entry`,
`list_feeds`.

## Layout

```
apps/web          React PWA + Capacitor android/ shell
apps/api          Go API + migrations + goose entrypoint + MCP endpoint
packages/api-client   shared typed client (mirrors openapi.yaml)
infra             docker-compose
```

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
| `GET /api/widget/today?token=` | token-gated read-only agenda |
| `POST /api/mcp` | MCP endpoint (bearer `apiToken`) |
| `GET /api/holidays?country=&year=` | computed holidays |
| `GET /api/holidays/countries` | supported regions |

See `openapi.yaml` for the full schema.

## License

MIT — see `LICENSE`.

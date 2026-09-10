# Cal

The self-hosted daily planner people actually enjoy opening.

Calendar, tasks, notes and links in one quiet place — month, week and day views,
recurring tasks, holidays for 40+ regions, dark mode, keyboard-first, installable
as a PWA. No accounts to create on someone else's server; one `docker compose up`
and it's yours.

## Features

- **Three real views** — month grid, week and day time grids with an all-day row
  and a live now-line
- **Tasks, notes, links** — with colors, tags, details and optional start/end
  times
- **Recurring tasks** — daily, weekly, monthly, yearly; completing one spawns
  the next occurrence
- **Holidays** — rule-based engine (Gregorian and Orthodox Easter, nth-weekday
  rules) covering 40+ countries, toggled per user
- **Command palette** — `⌘K` / `Ctrl+K` to search everything and run actions
- **Keyboard-first** — `1/2/3` switch views, `t` today, `←/→` navigate, `c` new
  entry, `/` search
- **Drag & drop** — move entries between days, reschedule onto the time grid,
  drop into all-day
- **Light, dark and system themes** — warm paper light, calm dark
- **Offline-first PWA** — installable, entries cached locally, read-only
  fallback when the API is unreachable
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

## Layout

```
apps/web          React PWA
apps/api          Go API + migrations + goose entrypoint
packages/api-client   shared typed client (mirrors openapi.yaml)
infra             docker-compose
```

## API surface

Auth is a secure, HttpOnly session cookie (`SESSION_SECURE=true` in production).

| Endpoint | Purpose |
| --- | --- |
| `POST /api/auth/register` `/login` `/logout`, `GET /api/me` | session auth (login/register rate-limited) |
| `GET/POST /api/entries`, `PATCH/DELETE /api/entries/:id` | entries; `q`, `from`, `to` filters |
| `GET/PUT /api/settings` | theme, week start, holiday region + toggle |
| `GET /api/holidays?country=&year=` | computed holidays |
| `GET /api/holidays/countries` | supported regions |

See `openapi.yaml` for the full schema.

## License

MIT — see `LICENSE`.

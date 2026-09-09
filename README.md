# Cal

Self-hosted minimal planner: calendar, tasks, notes, links, holidays, PWA/offline-first UI.

## Stack

- Web: React, Vite, TypeScript, Tailwind, Zustand, Framer Motion, PWA plugin
- API: Go, Gin, PostgreSQL, cookie sessions
- Contracts: OpenAPI in `openapi.yaml`, generated-style TS client in `packages/api-client`
- Infra: Docker Compose with Postgres, API, web

## Local Dev

```bash
npm install
cp apps/api/.env.example apps/api/.env
docker compose -f infra/docker-compose.yml up --build
```

Web: http://localhost:5173  
API: http://localhost:8080

## Direct Dev

```bash
npm run dev -w @cal/web
cd apps/api && go run ./cmd/server
```

## Tests

```bash
npm run typecheck
npm run test
cd apps/api && go test ./...
```

## Features

- Email + password accounts with cookie sessions
- Calendar month grid with task / note / link entries
- Tags, colors, completion state, full-text search
- Public holidays per country
- Installable PWA with offline shell

## License

MIT — see `LICENSE`.

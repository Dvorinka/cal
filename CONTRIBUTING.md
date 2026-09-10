# Contributing

Small, calm contributions welcome.

## Ground rules

- Keep the surface minimal. Cal is a personal planner, not a calendar suite.
- Stdlib and existing deps before new ones.
- Frontend stays React + TypeScript, backend stays Go + Gin. API changes go in
  `openapi.yaml` and `packages/api-client` together.
- Migrations are Goose — `Up`/`Down` sections, never edit applied migrations.

## Local setup

See README → Development. Then:

```bash
npm run typecheck && npm run test && npm run build
cd apps/api && go vet ./... && go test ./...
```

All must pass before a PR.

## Style

- Frontend: function components, Zustand for state, CSS in `styles.css` using the
  existing design tokens (`--bg-*`, `--text-*`, `--accent`).
- Backend: handlers validate input, store package owns SQL, errors map to
  typed `store.Err*` sentinels.
- No emojis in UI or comments.

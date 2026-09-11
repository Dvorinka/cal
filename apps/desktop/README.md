# Cal Desktop

Wails 2 shell around the Cal web app. The full Go API runs **in-process** —
no separate server, no extra port. Postgres is still required (point
`DATABASE_URL` at it).

## Layout

- `app.go` — embeds `frontend/dist` and mounts the API (`cal/apps/api/app`) + SPA fallback on one `http.Handler`.
- `main.go` — Wails window (`wails build`, default build).
- `main_headless.go` — TCP server (`-tags headless`), for machines/CI without webkit.
- `frontend/dist/` — staged copy of `apps/web/dist` (produced by `scripts/build-frontend.sh`, gitignored).
- `wails.json` — `frontend:build` runs the staging script.

## Build

```bash
# Prerequisites: wails CLI, node, Go 1.26+, platform webview deps
#   Linux:  libgtk-3-dev libwebkit2gtk-4.1-dev (or -4.0)
#   Windows: WebView2 (bundled via -webview2 embed)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

cd apps/desktop
wails build                      # → build/bin/cal (Linux/Windows via -platform)
wails build -platform windows/amd64 -webview2 embed   # Windows from any host w/ mingw
wails dev                        # live-reload dev loop
```

## Headless smoke test (no webkit needed)

```bash
./scripts/build-frontend.sh
go build -tags headless -o /tmp/cal .
DATABASE_URL='postgres://cal:cal@localhost:5433/cal?sslmode=disable' /tmp/cal
```

## Runtime config

| Env var | Default | Notes |
|---|---|---|
| `DATABASE_URL` | `postgres://cal:cal@localhost:5432/cal` | Required — Postgres is not embedded |
| `DATA_DIR` | `~/.config/cal` | Uploads, backups |
| `SESSION_SECURE` | `false` | `true` needs HTTPS (not needed for Wails) |

## Android

The same web app ships via Capacitor — see `apps/web/android/`
(`npm run build -w @cal/web && npx cap sync`).

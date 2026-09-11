# Cal Desktop

Wails 2 shell around the Cal web app. The full Go API runs **in-process** —
no separate server, no extra port — and the SPA is embedded in the binary.

**Turnkey by default:** with `DATABASE_URL` unset the app boots an embedded
Postgres (`embedded-postgres`) inside `<DATA_DIR>/pg` on a free port and runs
the migrations itself. First launch downloads ~80MB of PG binaries into the
tool cache; afterwards it is fully offline. Set `DATABASE_URL` to use an
external Postgres instead.

## Layout

- `app.go` — embeds `frontend/dist`, mounts the API (`cal/apps/api/app`) + SPA
  fallback on one `http.Handler`, owns the embedded-Postgres lifecycle.
- `main.go` — Wails window (`wails build`, default build).
- `main_headless.go` — TCP server (`-tags headless`), for machines/CI without webkit.
- `frontend/dist/` — staged copy of `apps/web/dist` (produced by `scripts/build-frontend.mjs`, gitignored).
- `wails.json` — `frontend:build` runs the staging script (Node, so it works on Windows too).
- `build/appicon.png` — window/installer icon.

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
node ./scripts/build-frontend.mjs
go build -tags headless -o /tmp/cal .
/tmp/cal                       # embedded Postgres, PORT=8080
# or external DB:
DATABASE_URL='postgres://cal:cal@localhost:5433/cal?sslmode=disable' /tmp/cal
```

## Runtime config

| Env var | Default | Notes |
|---|---|---|
| `DATABASE_URL` | embedded | Set to use your own Postgres |
| `DATA_DIR` | `~/.config/cal` | Uploads, backups, embedded PG data |
| `SESSION_SECURE` | `false` | `true` needs HTTPS (not needed for Wails) |

## Android / iOS

The same web app ships via Capacitor — see `apps/web/android/` and
`apps/web/ios/`. Native builds ask for a server URL on first login (the API
is not embedded on mobile) and authenticate with a Bearer session.
Debug and signed-release builds:

```bash
npm run build -w @cal/web && npx cap sync android
cd apps/web/android && ./gradlew assembleDebug     # app-debug.apk
# release: needs app/keystore/cal-release.jks + CAL_STORE_PASSWORD/CAL_KEY_PASSWORD
./gradlew assembleRelease                        # app-release.apk
```

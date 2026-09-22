# Cal Desktop

Wails 2 shell around the Cal web app. The full Go API runs **in-process** —
no separate server, no extra port — and the SPA is embedded in the binary.

**Turnkey by default:** with `DATABASE_URL` unset the app runs its own
Postgres (`postgres.go`) in `<DATA_DIR>/pg` on a free port and applies the
migrations itself. The window opens instantly on a setup screen while the
database warms up; errors show there instead of dying silently.

Windows installers (`-setup.exe` NSIS, `.msi`) ship `pg-runtime/` next to the
exe — first launch is fully offline and no console windows appear. Without
it (portable/dev runs) the binaries download once into `<DATA_DIR>/pg-bin`.
Set `DATABASE_URL` to use an external Postgres instead.

## Layout

- `app.go` — embeds `frontend/dist`, mounts the API (`cal/apps/api/app`) + SPA
  fallback on one `http.Handler`.
- `postgres.go` — embedded Postgres manager: download/verify/extract once,
  `initdb`, spawn with hidden consoles, Job-Object/Pdeathsig parent-death
  cleanup, stale `postmaster.pid` recovery.
- `sysproc_{windows,linux,other}.go` — console hiding + graceful stop +
  postmaster detection per OS.
- `backend.go` — deferred handler: serves `setup.html` + `/__setup/*` until
  `NewHandler` reports ready, then delegates.
- `main.go` — Wails window (`wails build`, default build).
- `main_headless.go` — TCP server (`-tags headless`), for machines/CI without webkit.
- `frontend/dist/` — staged copy of `apps/web/dist` (produced by `scripts/build-frontend.mjs`, gitignored).
- `scripts/prepare-pg-runtime.mjs` — downloads + verifies the Postgres
  runtime into `build/windows/installer/pg-runtime` for the installers.
- `build/windows/installer/project.nsi` — NSIS template (wizard, WebView2
  bootstrap, shortcuts, uninstaller, pg-runtime payload).
- `build/windows/cal.wxs` — WiX MSI definition.
- `wails.json` — `frontend:build` runs the staging script (Node, so it works on Windows too).
- `build/appicon.png` — window/installer icon (`icon.ico` generated from it).

## Build

```bash
# Prerequisites: wails CLI, node, Go 1.26+, platform webview deps
#   Linux:  libgtk-3-dev libwebkit2gtk-4.1-dev (or -4.0), nsis for installers
#   Windows: WebView2 (the NSIS installer bootstraps it when missing)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

cd apps/desktop
wails build                      # → build/bin/cal (Linux/Windows via -platform)
wails build -platform windows/amd64   # Windows exe from any host w/ mingw
wails dev                        # live-reload dev loop

# Windows installer (NSIS — needs makensis):
node scripts/prepare-pg-runtime.mjs windows amd64 build/windows/installer/pg-runtime
wails build -platform windows/amd64 -nsis   # → build/bin/Cal-amd64-installer.exe
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
| `DATA_DIR` | `~/.config/cal` (`%LOCALAPPDATA%\cal` on Windows) | Uploads, backups, embedded PG data + binaries |
| `CAL_PG_RUNTIME` | `<exe>/pg-runtime` | Pre-seeded Postgres binaries; skips download |
| `SESSION_SECURE` | `false` | `true` needs HTTPS (not needed for Wails) |

Runtime logs land in `<DATA_DIR>/logs/cal.log` — the setup screen links it
when something fails.

## Android / iOS

The same web app ships via Capacitor — see `apps/web/android/` and
`apps/web/ios/`. Native builds ask for a server URL on first login (the API
is not embedded on mobile) and authenticate with a Bearer session. On
Android, the login screen also offers **local mode** — no server at all:
the `CalMail` plugin (`dev.cal.app.CalMailPlugin`) does IMAP/SMTP on-device
with Jakarta Mail, and accounts live in Keystore-encrypted SharedPreferences.
Connecting a server later offers to import the local accounts.
Debug and signed-release builds:

```bash
npm run build -w @cal/web && npx cap sync android
cd apps/web/android && ./gradlew assembleDebug     # app-debug.apk
# release: needs app/keystore/cal-release.jks + CAL_STORE_PASSWORD/CAL_KEY_PASSWORD
./gradlew assembleRelease                        # app-release.apk
```

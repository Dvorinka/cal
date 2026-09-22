## Download Cal

Pick your platform — the app bundles everything it needs, including its
database. First launch works fully offline.

| Platform | File | Notes |
|---|---|---|
| **Windows** | **`Cal-Setup-Windows.exe`** | Installer — recommended. Per-user, no admin needed |
| | `Cal-Windows.msi` | For managed / enterprise deployment |
| **macOS** | **`Cal-macOS.dmg`** | Apple Silicon + Intel. Open, drag to Applications |
| **Linux** | **`Cal-Linux.AppImage`** | Runs anywhere, no install |
| | `Cal-Linux.deb` | Debian / Ubuntu |
| | `Cal-Linux.rpm` | Fedora / RHEL / openSUSE |
| **Android** | `Cal-Android.apk` | Sideload — enable "install unknown apps" |

**Self-hosting?** Grab `cal-server-{platform}` below and point it at a
Postgres with `DATABASE_URL` — or run the all-in-one container:
`docker run -p 8080:8080 ghcr.io/dvorinka/cal`.

---

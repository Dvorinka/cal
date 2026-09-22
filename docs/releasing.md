# Releasing & code signing

## How to cut a release

```bash
git tag v0.1.0
git push --tags
```

`.github/workflows/release.yml` runs on `v*` tags and produces:

| Artifact | Contents |
|---|---|
| `ghcr.io/dvorinka/cal:{version,latest}` | All-in-one image (SPA + API + embedded Postgres), linux/amd64 + arm64 |
| `cal-desktop-windows-amd64-setup.exe` | NSIS installer — wizard, WebView2 bootstrap, Start Menu/Desktop shortcuts, uninstaller, bundled Postgres runtime (fully offline first launch). Signed when secrets set |
| `cal-desktop-windows-amd64.msi` | WiX package with the same bundled runtime, for managed/enterprise deployment. Signed when secrets set |
| `cal-desktop-linux-amd64.AppImage` | Portable AppImage, no install needed |
| `cal-desktop-linux-amd64.deb` / `.rpm` | `nfpm` packages — `/usr/bin/cal` + desktop entry + icon |
| `cal-desktop-macos-arm64.dmg` | `cal.app` inside a DMG |
| `cal-server-<os>-<arch>[.exe]` | Bare headless server binaries |
| `cal-android-debug.apk` / signed AAB-capable APK | via android.yml |

Everything attaches to a GitHub Release on the tag. `workflow_dispatch` builds
the same matrix without creating a release — useful for verifying CI.

## Signing state today

| Platform | Status | User-visible effect |
|---|---|---|
| Linux | unsigned (normal — Linux has no signing culture for loose binaries) | none |
| Windows | unsigned | SmartScreen "unrecognized app" prompt → *More info → Run anyway* |
| macOS | unsigned | Gatekeeper blocks open → right-click → *Open*, or remove quarantine |
| Android | signed (debug) or `CAL_STORE_PASSWORD`/`CAL_KEY_PASSWORD` secrets for release | debug APK installs fine |

Signing steps in `release.yml` are **gated on secrets** — they activate the
moment the secrets exist, no workflow edit needed.

## Windows signing

The workflow uses `signtool.exe` (ships in the windows-latest SDK image) to
sign `cal.exe`, then wraps it in the MSI and signs that too. Needs a
code-signing certificate exported as PFX:

- `WINDOWS_CERT_PFX_BASE64` — `base64 -w0 cert.pfx`
- `WINDOWS_CERT_PASSWORD` — the PFX export password

Options for obtaining a cert:

1. **OV/EV cert** from a CA (SSL.com, DigiCert, ~$100–400/yr). OV shows
   SmartScreen until reputation builds; EV skips reputation.
2. **Azure Trusted Signing** (~$10/mo) — modern managed path; needs
   `signtool` with a `.json` metadata file instead of a PFX. If you go this
   route, adapt the Sign step: keep the cert in Azure, sign with the Trusted
   Signing dlib. Not wired here — the PFX path covers most OSS projects.
3. Self-signed — pointless; SmartScreen still warns and users can't verify.

`osslsigncode` is the alternative if you ever sign from Linux (not needed —
the Windows runner has signtool).

## macOS signing

Needs an Apple Developer account ($99/yr) and a **Developer ID Application**
certificate. Secrets:

- `APPLE_CERT_P12_BASE64` — export the Developer ID cert+key from Keychain
  Access as .p12, then `base64 -i cert.p12 | pbcopy`
- `APPLE_CERT_PASSWORD` — p12 export password
- `APPLE_TEAM_ID` — 10-char team id (developer.apple.com → Membership)
- `APPLE_ID` — your Apple account email
- `APPLE_APP_PASSWORD` — app-specific password (appleid.apple.com →
  Sign-In and Security → App-Specific Passwords)

The step imports into a throwaway keychain, `codesign --deep --options
runtime` (hardened runtime — required for notarization), submits via
`xcrun notarytool`, then staples the ticket into `cal.app`.

Note: the macOS build is `macos-latest` = **arm64 only**. Intel users would
need a universal build (`wails build -platform darwin/amd64` as a second
matrix row) — left out until there's demand.

## Linux

No signing needed — unsigned binaries are the norm. Three formats ship:

- `.deb` / `.rpm` — built by `nfpm` from `apps/desktop/build/linux/nfpm.yaml`;
  installs `cal` to `/usr/bin` with the desktop entry and icon
- `.AppImage` — portable single file built with `appimagetool`

`install.sh` remains in `apps/desktop/build/linux/` for manual source builds.

## Verifying a signed build

- Windows: `signtool verify /pa cal.exe`
- macOS: `spctl -a -vv cal.app` should print `accepted source=Notarized Developer ID`

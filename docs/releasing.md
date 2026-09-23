# Releasing & code signing

## How to cut a release

Merge to `main`. `.github/workflows/auto-release.yml` bumps the patch
version (`apps/desktop/scripts/bump-version.mjs`), commits it as
`chore: release vX.Y.Z`, tags the commit, and dispatches the release
pipelines on the tag.

Two ways to control what ships:

- **Routine merge** — releases `latest tag + 1 patch` automatically.
- **Release PR** — if the merged commit already bumped the version files
  (run `node apps/desktop/scripts/bump-version.mjs <latest-tag>` locally to
  produce the diff), the tag is placed on the merge commit as-is. This is
  how to ship a minor/major bump: set the version files yourself.

Escape hatches: `[no-release]` in the commit message skips tagging for that
merge, and pushing a tag manually still works — the workflow skips commits
that already carry a `v*` tag:

```bash
git tag v0.1.0
git push --tags
```

`.github/workflows/release.yml` runs on `v*` tags (directly or via dispatch
from auto-release) and produces:

| Artifact | Contents |
|---|---|
| `ghcr.io/dvorinka/cal:{version,latest}` | All-in-one image (SPA + API + embedded Postgres), linux/amd64 + arm64 |
| `Cal-Setup-Windows.exe` | NSIS installer — per-user (no admin), wizard, WebView2 bootstrap, Start Menu/Desktop shortcuts, uninstaller, bundled Postgres runtime (fully offline first launch). Signed when secrets set |
| `Cal-Windows.msi` | WiX package with the same bundled runtime, for managed/enterprise deployment. Signed when secrets set |
| `Cal-Linux.AppImage` | Portable AppImage, no install needed |
| `Cal-Linux.deb` / `Cal-Linux.rpm` | `nfpm` packages — `/usr/bin/cal` + desktop entry + icon |
| `Cal-macOS.dmg` | `cal.app` inside a DMG |
| `cal-server-<os>-<arch>[.exe]` | Bare headless server binaries |
| `Cal-Android.apk` / `Cal-Android-debug.apk` | via android.yml — signed release when keystore secrets exist, debug otherwise |

The release body comes from `.github/release-notes.md` — a pick-your-platform
download table — with GitHub's generated changelog appended. `workflow_dispatch`
builds the same matrix without creating a release — useful for verifying CI.

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

The macOS build is `-platform darwin/universal` — one DMG covers Apple
Silicon and Intel.

## Linux

No signing needed — unsigned binaries are the norm. Three formats ship:

- `.deb` / `.rpm` — built by `nfpm` from `apps/desktop/build/linux/nfpm.yaml`;
  installs `cal` to `/usr/bin` with the desktop entry and icon, and declares
  `webkit2gtk`/`gtk3` as real package dependencies
- `.AppImage` — self-contained: linuxdeploy + gtk plugin bundles GTK, WebKit
  and its helper processes, so it runs on distros without webkit2gtk
  preinstalled

`install.sh` remains in `apps/desktop/build/linux/` for manual source builds.

## Verifying a signed build

- Windows: `signtool verify /pa cal.exe`
- macOS: `spctl -a -vv cal.app` should print `accepted source=Notarized Developer ID`

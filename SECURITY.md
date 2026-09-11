# Security Policy

## Reporting a vulnerability

Please do not open a public issue for security reports. Email
**info@tdvorak.dev** with a description and reproduction steps. You will get a
response within a few days.

## Scope notes

Cal is designed to be self-hosted behind your own reverse proxy. Notable
decisions:

- Passwords are bcrypt-hashed; sessions are HttpOnly cookies (`SESSION_SECURE`
  should be `true` when served over HTTPS).
- Mail/CalDAV credentials are encrypted at rest with AES-256-GCM.
- Feed, webhook and unfurl URLs are SSRF-guarded (private ranges rejected).
- Public board and file share links are unguessable tokens; treat them as
  capability URLs.
- The `/api/intake`, `/api/widget/today` and `/api/mcp` endpoints are
  token-gated. Rotate tokens in Settings.
- ICS export and the widget expose your agenda to anyone holding the token —
  rotate it if a link leaks.

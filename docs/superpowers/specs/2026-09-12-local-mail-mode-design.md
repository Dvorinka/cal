# Local mail mode — design

Date: 2026-09-12
Status: approved by user (chat)

## Intent

Let the Android app run with **no Cal server at all**. In this mode the only
module is Mail, which talks directly from the device to the user's own
IMAP/SMTP provider. Nothing is sent anywhere except to that provider —
self-hosting in spirit: the mail host is already theirs.

Desktop is unchanged: the Wails app already runs the full API + embedded
Postgres in-process, which counts as "no server running" for this purpose.
Browser/PWA cannot participate — the web platform has no raw TCP/TLS, so IMAP
is physically impossible there. iOS is out of scope for now (the local-mode
entry is gated to `Capacitor.getPlatform() === "android"`).

## Shape

### Android: `CalMailPlugin`

`apps/web/android/app/src/main/java/dev/cal/app/CalMailPlugin.java` —
a `@CapacitorPlugin(name = "CalMail")` registered in `MainActivity` alongside
`WidgetConfigPlugin`. Backed by Jakarta Mail for Android
(`com.sun.mail:android-mail` + `android-activation`). Methods mirror the
existing `/api/mail/*` contract so the UI is backend-agnostic:

| Plugin method | Mirrors | Notes |
| --- | --- | --- |
| `listAccounts` | `GET /mail/accounts` | metadata only, no passwords |
| `addAccount` | `POST /mail/accounts` | defaults: imap 993, smtp 465, username=email |
| `removeAccount` | `DELETE /mail/accounts/:id` | |
| `testAccount` | `POST .../test` | imap dial + login |
| `exportAccounts` | — | full records incl. passwords; used only by server import |
| `mailboxes` | `GET /mail/:id/mailboxes` | |
| `messages` | `GET /mail/:id/messages` | same newest-first seq window, pageSize 40 |
| `message` | `GET /mail/:id/message/:uid` | prefers text/plain, html fallback |
| `setSeen` | `POST .../flag` | |
| `deleteMessage` | `DELETE .../message/:uid` | copy→Trash + expunge fallback |
| `send` | `POST /mail/:id/send` | attachments arrive as base64 `{name,mime,data}` |

Accounts also carry `insecure` (skip TLS verification — self-hosted mail with
self-signed certs) and custom `imapPort`/`smtpPort`. Both were added to the
server schema (`mail_accounts.insecure_tls`, migration 00014) so the option
round-trips through the import path.

Accounts persist as a single JSON blob in SharedPreferences, encrypted with an
AES-256-GCM key from AndroidKeyStore (`iv‖ciphertext`, base64). Matches the
server story ("AES-256-GCM for stored credentials") without pulling in the
deprecated `security-crypto` library. All IMAP/SMTP work runs on a cached
executor; `PluginCall` resolves/rejects off the UI thread.

### Web: `MailBackend` abstraction

`apps/web/src/lib/mail.ts` defines the interface both transports satisfy:

```ts
interface MailBackend {
  accounts(): Promise<MailAccount[]>
  addAccount(input: MailAccountInput): Promise<MailAccount>
  removeAccount(id: string): Promise<void>
  testAccount(id: string): Promise<void>
  mailboxes(id: string): Promise<{ name: string; delimiter?: string }[]>
  messages(id: string, mailbox: string, page: number): Promise<{ total: number; messages: MailSummary[] }>
  message(id: string, uid: number, mailbox: string): Promise<MailMessage>
  setSeen(id: string, uid: number, seen: boolean, mailbox: string): Promise<void>
  remove(id: string, uid: number, mailbox: string): Promise<void>
  send(id: string, input: SendInput): Promise<void> // SendInput.files: File[]
}
```

- `httpMail(api)` — wraps the existing `CalApi` calls; `send` uploads each
  `File` to `/files` first, then posts stored names (same as today, moved from
  pick-time to send-time).
- `nativeMail` — `registerPlugin<…>("CalMail")` from `@capacitor/core`;
  base64s `File`s into `send`. Adds `exportAccounts()` for the import path.
- `mailBackend()` picks native when `isLocalMode()`, else http.

`MailPage` consumes `mailBackend()` instead of `usePlanner(s => s.api)`;
compose holds `File[]` and defers to the backend.

### Local mode lifecycle

`apps/web/src/lib/local.ts`:

- `canUseLocal()` — Android native only.
- `isLocalMode()` / `enterLocal()` / `exitLocal()` — `cal:mode` localStorage
  flag. Entering also sets `cal:import` so a later server connect offers the
  account import.

`planner.ts`:

- `mode: "server" | "local"` in state; `bootstrap()` short-circuits to a
  synthetic user (`{id:"local", email:"local device"}`) + cached settings when
  the flag is set — no `api.me()` call, zero network.
- `enterLocal()`, `exitLocal()` (clears mode, drops synthetic user → AuthPanel).
- After a successful `login()`/`register()`, if `cal:import` is set:
  `exportAccounts()` → fetch server `mailAccounts()` → create each account not
  already present (dedupe on `email`+`imapHost`) → toast a summary → clear the
  flag. Import is offered via toast action, not automatic — sending passwords
  to a server, even their own, gets an explicit tap. Local copies are kept;
  the server becomes source of truth while in server mode.

### Shell

`LocalShell.tsx` — minimal layout (brand + "local" chip, `MailPage`, footer
"Connect a server" → `exitLocal()`, `Toasts`). Rendered by `App.tsx`'s `Shell`
when `mode === "local"`, after the public-board early return. Keeps the full
planner shell (sidebar, palette, notifications, keybindings) untouched — no
conditional clutter for a mode that deliberately has no calendar.

`AuthPanel` gains a quiet "Use without a server — mail only" link, Android
only.

## Non-goals

- Syncing mail *content* — it lives on the provider; IMAP is the sync layer.
- iOS, desktop changes, any module besides Mail, merging server+local account
  lists into one view.

## Testing

- vitest: `mailBackend()` selection, mode flag round-trip, base64 file
  encoding helper.
- JVM unit test for Java helpers that are pure (address splitting, header
  sanitizing) if the plugin factors them static — else manual on-device check.
- Manual: add Gmail/Dovecot account on-device, list/read/flag/delete/send;
  connect a server, import accounts, verify dedupe.

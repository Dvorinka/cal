// Mail — IMAP/SMTP client. Left rail: accounts + mailboxes. Middle:
// envelope list. Right: reader/compose. Briefkescht-shaped triage —
// read, reply, flag, bin — without the bloat.

import type { MailAccount, MailMessage, MailSummary } from "@cal/api-client";
import { Inbox, Mail, Paperclip, PenSquare, Plus, RefreshCw, Send, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { reportErr, usePlanner } from "../stores/planner";

export function MailPage() {
  const api = usePlanner((s) => s.api);
  const toast = usePlanner((s) => s.toast);
  const [accounts, setAccounts] = useState<MailAccount[]>([]);
  const [account, setAccount] = useState<MailAccount | null>(null);
  const [mailboxes, setMailboxes] = useState<{ name: string }[]>([]);
  const [mailbox, setMailbox] = useState("INBOX");
  const [list, setList] = useState<MailSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [reading, setReading] = useState<MailMessage | null>(null);
  const [readingUid, setReadingUid] = useState<number | null>(null);
  const [compose, setCompose] = useState(false);
  const [adding, setAdding] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const loadAccounts = useCallback(() => {
    void api.mailAccounts().then(setAccounts).catch((e) => {
      setAccounts([]);
      reportErr("Could not load mail accounts")(e);
    });
  }, [api]);
  useEffect(loadAccounts, [loadAccounts]);

  const pickAccount = useCallback(
    (a: MailAccount) => {
      setAccount(a);
      setMailbox("INBOX");
      setReading(null);
      setReadingUid(null);
      void api.mailMailboxes(a.id).then(setMailboxes).catch((e) => {
        setMailboxes([]);
        reportErr("Could not load mailboxes")(e);
      });
    },
    [api],
  );

  const loadMessages = useCallback(
    (acct: MailAccount, box: string) => {
      setBusy(true);
      setErr("");
      void api
        .mailMessages(acct.id, box)
        .then((r) => {
          setList(r.messages);
          setTotal(r.total);
        })
        .catch((e: Error) => setErr(e.message))
        .finally(() => setBusy(false));
    },
    [api],
  );

  useEffect(() => {
    if (account) loadMessages(account, mailbox);
  }, [account, mailbox, loadMessages]);

  async function openMessage(uid: number) {
    if (!account) return;
    setReadingUid(uid);
    setReading(null);
    try {
      const msg = await api.mailMessage(account.id, uid, mailbox);
      setReading(msg);
      // Mark seen + reflect locally.
      void api.mailFlag(account.id, uid, true, mailbox).catch((e) => toast(e instanceof Error ? e.message : "Could not mark read"));
      setList((l) => l.map((m) => (m.uid === uid ? { ...m, seen: true } : m)));
    } catch {
      setErr("Could not open message");
      setReadingUid(null);
    }
  }

  async function removeMessage(uid: number) {
    if (!account) return;
    try {
      await api.mailDelete(account.id, uid, mailbox);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Delete failed");
      return;
    }
    setList((l) => l.filter((m) => m.uid !== uid));
    if (readingUid === uid) {
      setReading(null);
      setReadingUid(null);
    }
  }

  if (accounts.length === 0 && !adding) {
    return (
      <>
        <PageHeader title="Mail" sub="connect an account to begin" />
        <div className="page-scroll">
          <div className="empty-hint">
            <Inbox size={28} style={{ color: "var(--text-3)" }} />
            <strong>No mail accounts</strong>
            <span>IMAP/SMTP — works with Gmail, Fastmail, Dovecot, anything.</span>
            <button type="button" className="btn btn-primary" onClick={() => setAdding(true)}>
              <Plus size={14} /> Add account
            </button>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader title="Mail" sub={account ? `${account.email} · ${mailbox}` : "choose an account"}>
        {account && (
          <>
            <button type="button" className="btn btn-secondary btn-xs" onClick={() => loadMessages(account, mailbox)} disabled={busy}>
              <RefreshCw size={12} /> Refresh
            </button>
            <button type="button" className="btn btn-primary btn-xs" onClick={() => setCompose(true)}>
              <PenSquare size={12} /> Compose
            </button>
          </>
        )}
        {!account && accounts.length > 0 && (
          <button type="button" className="btn btn-secondary btn-xs" onClick={() => setAdding(true)}>
            <Plus size={12} /> Account
          </button>
        )}
      </PageHeader>
      <div className="page-scroll mail-layout">
        <aside className="mail-rail">
          <div className="side-label">Accounts</div>
          {accounts.map((a) => (
            <button
              key={a.id}
              type="button"
              className={`filter-item ${account?.id === a.id ? "on" : ""}`}
              onClick={() => pickAccount(a)}
            >
              <Mail size={13} />
              <span className="mail-acct-name">{a.name || a.email}</span>
            </button>
          ))}
          <button type="button" className="filter-item" onClick={() => setAdding(true)}>
            <Plus size={13} /> Add account
          </button>
          {account && mailboxes.length > 0 && (
            <>
              <div className="side-label" style={{ marginTop: 12 }}>Mailboxes</div>
              {mailboxes.slice(0, 12).map((b) => (
                <button
                  key={b.name}
                  type="button"
                  className={`filter-item ${mailbox === b.name ? "on" : ""}`}
                  onClick={() => {
                    setMailbox(b.name);
                    setReading(null);
                  }}
                >
                  {b.name}
                </button>
              ))}
            </>
          )}
        </aside>

        <section className="mail-list">
          {err && <p className="panel-note" style={{ color: "var(--c-red)" }}>{err}</p>}
          {!account && <p className="panel-empty">Select an account on the left.</p>}
          {account && busy && list.length === 0 && <p className="panel-empty">Loading…</p>}
          {account && !busy && list.length === 0 && !err && <p className="panel-empty">Empty mailbox.</p>}
          <ul className="mail-items">
            {list.map((m) => (
              <li key={m.uid}>
                <button
                  type="button"
                  className={`mail-item ${m.seen ? "seen" : ""} ${readingUid === m.uid ? "active" : ""}`}
                  onClick={() => void openMessage(m.uid)}
                >
                  <span className="mail-from">{m.from || "(unknown)"}</span>
                  <span className="mail-subject">{m.subject || "(no subject)"}</span>
                  <span className="mail-date">
                    {new Date(m.date).toLocaleDateString(undefined, { month: "short", day: "numeric" })}
                  </span>
                </button>
              </li>
            ))}
          </ul>
          {account && total > list.length && <p className="panel-note">{total} messages total</p>}
        </section>

        <section className="mail-reader">
          {readingUid && !reading && <p className="panel-empty">Opening…</p>}
          {!readingUid && <p className="panel-empty">Select a message.</p>}
          {reading && (
            <div className="mail-body">
              <h3>{reading.subject || "(no subject)"}</h3>
              <p className="panel-note">
                {reading.from} → {reading.to.join(", ")}
              </p>
              <div className="mail-actions">
                <button type="button" className="btn btn-secondary btn-xs" onClick={() => setCompose(true)}>
                  <PenSquare size={12} /> Reply
                </button>
                <button
                  type="button"
                  className="btn btn-secondary btn-xs"
                  onClick={() =>
                    account &&
                    void api
                      .mailFlag(account.id, reading.uid, false, mailbox)
                      .then(() => setList((l) => l.map((m) => (m.uid === reading.uid ? { ...m, seen: false } : m))))
                      .catch((e) => toast(e instanceof Error ? e.message : "Could not flag"))
                  }
                >
                  Unread
                </button>
                <button
                  type="button"
                  className="btn btn-secondary btn-xs danger"
                  onClick={() => void removeMessage(reading.uid)}
                >
                  <Trash2 size={12} /> Delete
                </button>
              </div>
              {reading.text ? (
                <pre className="mail-text">{reading.text}</pre>
              ) : (
                // Server returns raw HTML; sandbox it inside an iframe-less
                // viewer by stripping scripts before render.
                <div
                  className="mail-text mail-html"
                  dangerouslySetInnerHTML={{ __html: sanitizeMailHtml(reading.html) }}
                />
              )}
            </div>
          )}
        </section>
      </div>

      {adding && <AccountDialog onClose={() => setAdding(false)} onSaved={() => { setAdding(false); loadAccounts(); }} />}
      {compose && account && (
        <ComposeDialog
          account={account}
          replyTo={reading}
          onClose={() => setCompose(false)}
          onSent={() => {
            setCompose(false);
            toast("Sent");
          }}
        />
      )}
    </>
  );
}

// Naive sanitiser for mail HTML: drop scripts/iframes/objects and on* attrs.
// jarvis: ceiling — a real sanitizer (DOMPurify) if HTML mail gets heavy.
function sanitizeMailHtml(html: string): string {
  return html
    .replace(/<(script|iframe|object|embed|form)[\s\S]*?<\/\1>/gi, "")
    .replace(/<(script|iframe|object|embed|form)[^>]*\/?>/gi, "")
    .replace(/\son\w+\s*=\s*"[^"]*"/gi, "")
    .replace(/\son\w+\s*=\s*'[^']*'/gi, "")
    .replace(/\son\w+\s*=\s*[^\s>]+/gi, "")
    .replace(/javascript:/gi, "");
}

function AccountDialog({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const api = usePlanner((s) => s.api);
  const [f, setF] = useState({ name: "", email: "", imapHost: "", smtpHost: "", username: "", password: "" });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const set = (k: string) => (e: React.ChangeEvent<HTMLInputElement>) => setF({ ...f, [k]: e.target.value });

  async function save() {
    setBusy(true);
    setErr("");
    try {
      const acct = await api.createMailAccount(f);
      // Verify the credentials actually log in before declaring victory.
      await api.testMailAccount(acct.id).catch((e: Error) => {
        setErr(`Saved, but IMAP login failed: ${e.message}`);
        throw e;
      });
      onSaved();
    } catch (e) {
      if (!err) setErr(e instanceof Error ? e.message : "Failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="scrim" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="palette mail-dialog" role="dialog" aria-label="Add mail account">
        <h3>Add mail account</h3>
        <label className="field"><span>Name</span><input value={f.name} onChange={set("name")} placeholder="Work" /></label>
        <label className="field"><span>Email</span><input type="email" value={f.email} onChange={set("email")} placeholder="you@example.com" required /></label>
        <div className="field-row">
          <label className="field"><span>IMAP host</span><input value={f.imapHost} onChange={set("imapHost")} placeholder="imap.example.com" required /></label>
          <label className="field"><span>SMTP host</span><input value={f.smtpHost} onChange={set("smtpHost")} placeholder="smtp.example.com" required /></label>
        </div>
        <label className="field"><span>Username (blank = email)</span><input value={f.username} onChange={set("username")} /></label>
        <label className="field"><span>Password / app token</span><input type="password" value={f.password} onChange={set("password")} required /></label>
        {err && <p className="panel-note" style={{ color: "var(--c-red)" }}>{err}</p>}
        <div className="mail-dialog-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary" disabled={busy || !f.email || !f.imapHost || !f.password} onClick={() => void save()}>
            {busy ? "Verifying…" : "Connect"}
          </button>
        </div>
      </div>
    </div>
  );
}

function ComposeDialog({ account, replyTo, onClose, onSent }: { account: MailAccount; replyTo: MailMessage | null; onClose: () => void; onSent: () => void }) {
  const api = usePlanner((s) => s.api);
  const [to, setTo] = useState(replyTo?.from ?? "");
  const [subject, setSubject] = useState(replyTo ? `Re: ${replyTo.subject}` : "");
  const [text, setText] = useState("");
  const [files, setFiles] = useState<{ name: string; orig: string }[]>([]);
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function attach(list: FileList | null) {
    if (!list?.length) return;
    setUploading(true);
    setErr("");
    try {
      for (const f of Array.from(list)) {
        const out = await api.upload(f);
        setFiles((cur) => [...cur, { name: out.name, orig: f.name }]);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Upload failed");
    } finally {
      setUploading(false);
      if (fileRef.current) fileRef.current.value = "";
    }
  }

  async function send() {
    setBusy(true);
    setErr("");
    try {
      await api.mailSend(account.id, { to, subject, text, attachments: files.map((f) => f.name) });
      onSent();
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Send failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="scrim" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="palette mail-dialog" role="dialog" aria-label="Compose">
        <h3>{replyTo ? "Reply" : "Compose"}</h3>
        <label className="field"><span>To</span><input value={to} onChange={(e) => setTo(e.target.value)} required /></label>
        <label className="field"><span>Subject</span><input value={subject} onChange={(e) => setSubject(e.target.value)} /></label>
        <textarea className="mail-compose" rows={10} value={text} onChange={(e) => setText(e.target.value)} placeholder="Write…" />
        {files.length > 0 && (
          <div className="mail-attachments">
            {files.map((f) => (
              <span key={f.name} className="chip">
                <Paperclip size={11} /> {f.orig}
                <button type="button" className="chip-x" aria-label={`Remove ${f.orig}`} onClick={() => setFiles((cur) => cur.filter((x) => x.name !== f.name))}>×</button>
              </span>
            ))}
          </div>
        )}
        <input ref={fileRef} type="file" multiple hidden onChange={(e) => void attach(e.target.files)} />
        {err && <p className="panel-note" style={{ color: "var(--c-red)" }}>{err}</p>}
        <div className="mail-dialog-actions">
          <button type="button" className="btn btn-secondary" onClick={() => fileRef.current?.click()} disabled={uploading}>
            <Paperclip size={13} /> {uploading ? "Uploading…" : "Attach"}
          </button>
          <span style={{ flex: 1 }} />
          <button type="button" className="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary" disabled={busy || !to || uploading} onClick={() => void send()}>
            <Send size={13} /> {busy ? "Sending…" : "Send"}
          </button>
        </div>
      </div>
    </div>
  );
}

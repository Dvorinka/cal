import type { Accent, SessionInfo, Webhook } from "@cal/api-client";
import { Bell, BellOff, Copy, Download, FileText, LogOut, Plus, RefreshCw, Trash2, Upload } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { MODULES, moduleOn } from "../lib/modules";
import { disablePush, enablePush, pushEnabled } from "../lib/push";
import { usePlanner } from "../stores/planner";

const timezones: string[] = (() => {
  const zones: string[] = typeof Intl.supportedValuesOf === "function" ? Intl.supportedValuesOf("timeZone") : [];
  const local = Intl.DateTimeFormat().resolvedOptions().timeZone;
  return local && !zones.includes(local) ? [local, ...zones] : zones;
})();

const ACCENTS: { value: Accent; label: string }[] = [
  { value: "green", label: "Forest" },
  { value: "blue", label: "Lake" },
  { value: "violet", label: "Iris" },
  { value: "amber", label: "Marigold" },
  { value: "rose", label: "Rose" },
];

export function SettingsPage() {
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const countries = usePlanner((state) => state.countries);
  const entries = usePlanner((state) => state.entries);
  const feeds = usePlanner((state) => state.feeds);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const loadFeeds = usePlanner((state) => state.loadFeeds);
  const addFeed = usePlanner((state) => state.addFeed);
  const accounts = usePlanner((state) => state.accounts);
  const loadAccounts = usePlanner((state) => state.loadAccounts);
  const addAccount = usePlanner((state) => state.addAccount);
  const removeAccount = usePlanner((state) => state.removeAccount);
  const syncAccount = usePlanner((state) => state.syncAccount);
  const removeFeed = usePlanner((state) => state.removeFeed);
  const refreshFeed = usePlanner((state) => state.refreshFeed);
  const importIcs = usePlanner((state) => state.importIcs);
  const restore = usePlanner((state) => state.restore);
  const api = usePlanner((state) => state.api);
  const rotateToken = usePlanner((state) => state.rotateToken);
  const toast = usePlanner((state) => state.toast);
  const user = usePlanner((state) => state.user);
  const logout = usePlanner((state) => state.logout);
  const workspaces = usePlanner((state) => state.workspaces);
  const addWorkspace = usePlanner((state) => state.addWorkspace);
  const removeWorkspace = usePlanner((state) => state.removeWorkspace);
  const [wsName, setWsName] = useState("");

  const [feedName, setFeedName] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [feedKind, setFeedKind] = useState<"calendar" | "links">("calendar");
  const [adding, setAdding] = useState(false);
  const [pushOn, setPushOn] = useState<boolean>();
  const [davName, setDavName] = useState("");
  const [davUrl, setDavUrl] = useState("");
  const [davUser, setDavUser] = useState("");
  const [davPass, setDavPass] = useState("");
  const [addingAccount, setAddingAccount] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [discovered, setDiscovered] = useState<{ href: string; name: string }[]>([]);
  const [google, setGoogle] = useState<{ connected: boolean }>();
  const [cdName, setCdName] = useState("");
  const [cdUrl, setCdUrl] = useState("");
  const [cdUser, setCdUser] = useState("");
  const [cdPass, setCdPass] = useState("");
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [hookUrl, setHookUrl] = useState("");
  const [storage, setStorage] = useState<{ usedBytes: number; quotaBytes: number }>();
  const [pushDevices, setPushDevices] = useState<{ id: string; label: string; endpoint: string }[]>([]);
  const [pwCurrent, setPwCurrent] = useState("");
  const [pwNext, setPwNext] = useState("");
  const [pwSaving, setPwSaving] = useState(false);
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const fileRef = useRef<HTMLInputElement>(null);
  const restoreRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void pushEnabled().then(setPushOn).catch(() => setPushOn(false));
  }, []);

  useEffect(() => {
    void loadEntries({});
    void loadFeeds();
    void loadAccounts();
    void api.sessions().then(setSessions).catch(() => {});
    void api.webhooks().then(setWebhooks).catch(() => {});
    void api.googleStatus().then(setGoogle).catch(() => {});
    void api.storage().then(setStorage).catch(() => {});
    void api.pushSubscriptions().then(setPushDevices).catch(() => {});
  }, [loadEntries, loadFeeds, loadAccounts, api]);

  const stats = useMemo(() => {
    const counts = { task: 0, event: 0, note: 0, link: 0, done: 0 };
    for (const entry of entries) {
      if (entry.type === "task") {
        counts.task++;
        if (entry.completed) counts.done++;
      } else if (entry.type === "note") counts.note++;
      else if (entry.type === "event") counts.event++;
      else counts.link++;
    }
    return counts;
  }, [entries]);

  const set = (patch: Partial<typeof settings>) => void updateSettings({ ...settings, ...patch });

  async function submitPassword() {
    setPwSaving(true);
    try {
      await api.changePassword(pwCurrent, pwNext);
      setPwCurrent("");
      setPwNext("");
      void api.sessions().then(setSessions);
      toast("Password changed");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Failed");
    } finally {
      setPwSaving(false);
    }
  }

  async function revokeSession(id: string) {
    try {
      await api.revokeSession(id);
      setSessions((s) => s.filter((x) => x.id !== id));
      toast("Session signed out");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Failed");
    }
  }

  async function submitAccount() {
    setAddingAccount(true);
    const ok = await addAccount({
      name: davName.trim() || undefined,
      url: davUrl.trim(),
      username: davUser.trim(),
      password: davPass,
    });
    setAddingAccount(false);
    if (ok) {
      setDavName("");
      setDavUrl("");
      setDavUser("");
      setDavPass("");
    }
  }

  async function runDiscover() {
    setDiscovering(true);
    try {
      const cols = await api.discoverCaldav({ url: davUrl.trim(), username: davUser.trim(), password: davPass });
      setDiscovered(cols);
      if (cols.length === 0) toast("No calendar collections found — check the URL");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Discovery failed");
    } finally {
      setDiscovering(false);
    }
  }

  async function submitCarddav() {
    try {
      const out = await api.connectCarddav({
        name: cdName.trim() || undefined,
        url: cdUrl.trim(),
        username: cdUser.trim(),
        password: cdPass,
      });
      toast(`Imported ${out.imported} of ${out.found} birthdays`);
      setCdName(""); setCdUrl(""); setCdUser(""); setCdPass("");
      void loadEntries({});
    } catch (error) {
      toast(error instanceof Error ? error.message : "CardDAV connect failed");
    }
  }

  async function submitWebhook() {
    try {
      const hook = await api.addWebhook(hookUrl.trim());
      setWebhooks((w) => [...w, hook]);
      setHookUrl("");
      toast("Webhook added");
    } catch (error) {
      toast(error instanceof Error ? error.message : "Failed");
    }
  }

  async function removePushDevice(id: string) {
    await api.deletePushSubscription(id).catch(() => {});
    setPushDevices((d) => d.filter((x) => x.id !== id));
    toast("Device removed");
  }

  async function removeWebhook(id: string) {
    await api.deleteWebhook(id).catch(() => {});
    setWebhooks((w) => w.filter((x) => x.id !== id));
  }

  async function submitFeed() {
    if (!feedUrl.trim()) return;
    setAdding(true);
    const ok = await addFeed({ name: feedName.trim(), url: feedUrl.trim(), kind: feedKind });
    setAdding(false);
    if (ok) {
      setFeedName("");
      setFeedUrl("");
    }
  }

  const widgetUrl = `${location.origin}/widget/today?token=${settings.widgetToken}`;
  const icsUrl = `${location.origin}/api/feed.ics?token=${settings.widgetToken}`;
  const mcpConfig = JSON.stringify(
    { mcpServers: { cal: { url: `${location.origin}/api/mcp`, headers: { Authorization: `Bearer ${settings.apiToken}` } } } },
    null,
    2,
  );

  function copy(text: string, what: string) {
    void navigator.clipboard.writeText(text).then(() => toast(`${what} copied`));
  }

  return (
    <>
      <PageHeader title="Settings" />
      <div className="page-scroll settings-page">
        <section className="panel">
          <h3>Appearance</h3>
          <div className="settings-grid">
            <label className="field">
              <span>Theme</span>
              <select
                className="select"
                value={settings.theme}
                onChange={(e) => set({ theme: e.target.value as typeof settings.theme })}
              >
                <option value="system">System</option>
                <option value="dark">Dark</option>
                <option value="light">Light</option>
              </select>
            </label>
            <label className="field">
              <span>Week starts on</span>
              <select
                className="select"
                value={settings.weekStart}
                onChange={(e) => set({ weekStart: e.target.value as typeof settings.weekStart })}
              >
                <option value="monday">Monday</option>
                <option value="sunday">Sunday</option>
              </select>
            </label>
            <label className="field">
              <span>Timezone</span>
              <select
                className="select"
                value={settings.timezone || "UTC"}
                onChange={(e) => set({ timezone: e.target.value })}
              >
                {timezones.map((tz) => (
                  <option key={tz} value={tz}>
                    {tz}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span>GitHub token</span>
              <input
                type="password"
                className="input"
                placeholder="Personal access token — issues & PRs"
                value={settings.githubToken ?? ""}
                onChange={(e) => set({ githubToken: e.target.value })}
              />
            </label>
            <label className="field">
              <span>Default hourly rate</span>
              <input
                type="number"
                min="0"
                step="0.01"
                className="input"
                placeholder="$/h for billable timers"
                value={settings.defaultRate ?? ""}
                onChange={(e) => set({ defaultRate: e.target.value === "" ? undefined : Number(e.target.value) })}
              />
            </label>
            <label className="field">
              <span>Morning digest push</span>
              <input
                type="time"
                className="input"
                value={settings.digestTime ?? ""}
                onChange={(e) => set({ digestTime: e.target.value })}
                aria-label="Daily digest time"
              />
            </label>
            <label className="field">
              <span>Weather city</span>
              <input
                className="input"
                placeholder="e.g. Prague — blank hides the strip"
                value={settings.city}
                onChange={(e) => set({ city: e.target.value })}
              />
            </label>
            <div className="field">
              <span>Accent</span>
              <div className="dots">
                {ACCENTS.map(({ value, label }) => (
                  <button
                    key={value}
                    type="button"
                    className={`dot accent-${value} ${settings.accent === value ? "active" : ""}`}
                    onClick={() => set({ accent: value })}
                    aria-label={`Accent ${label}`}
                    title={label}
                  />
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className="panel">
          <h3>Workspaces</h3>
          <p className="panel-note">Separate scopes for entries, files and time — Work, Personal, anything. Deleting a space returns its contents to Personal.</p>
          <ul className="check-list" style={{ marginBottom: 10 }}>
            {workspaces.map((w) => (
              <li key={w.id}>
                <span className="swatch" style={{ "--swatch": `var(--c-${w.color})` } as React.CSSProperties} />
                <span className="row-title">{w.name}</span>
                <button
                  type="button"
                  className="icon-btn danger"
                  aria-label={`Delete ${w.name}`}
                  onClick={() => void removeWorkspace(w.id)}
                >
                  ×
                </button>
              </li>
            ))}
            {workspaces.length === 0 && <li className="panel-empty">No spaces yet — everything lives in Personal.</li>}
          </ul>
          <div className="field-row">
            <input
              className="input"
              placeholder="New space name"
              value={wsName}
              onChange={(e) => setWsName(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && wsName.trim() && void addWorkspace({ name: wsName.trim() }).then(() => setWsName(""))}
            />
            <button
              type="button"
              className="btn btn-secondary"
              disabled={!wsName.trim()}
              onClick={() => void addWorkspace({ name: wsName.trim() }).then((w) => w && setWsName(""))}
            >
              Create
            </button>
          </div>
        </section>

        <section className="panel">
          <h3>Modules</h3>
          <p className="panel-note">Turn off what you don't use — hidden from navigation, routes and search.</p>
          <div className="module-grid">
            {MODULES.map((m) => {
              const on = moduleOn(settings, m.key);
              return (
                <button
                  key={m.key}
                  type="button"
                  className={`module-toggle ${on ? "on" : ""}`}
                  aria-pressed={on}
                  title={m.hint}
                  onClick={() => set({ modules: { ...settings.modules, [m.key]: !on } })}
                >
                  <span className="module-dot" />
                  {m.label}
                  <i>{on ? "on" : "off"}</i>
                </button>
              );
            })}
          </div>
        </section>

        <section className="panel">
          <h3>Feeds</h3>
          <p className="panel-note">
            Subscribe to iCalendar (.ics) or RSS/Atom URLs — Google Calendar's secret address,
            iCloud public calendars, blog feeds. "To links" feeds (e.g. a YouTube channel's
            <code>feeds/videos.xml</code>) save new items as link cards with thumbnails instead.
          </p>
          {feeds.map((feed) => (
            <div key={feed.id} className="feed-row">
              <span className="swatch" style={{ "--swatch": `var(--c-${feed.color})` } as React.CSSProperties} />
              <div className="feed-meta">
                <span className="feed-name">{feed.name}{feed.kind === "links" && <span className="feed-kind">→ links</span>}</span>
                <span className="feed-url">{feed.url}</span>
              </div>
              {feed.fetchedAt && (
                <span className="feed-age">synced {new Date(feed.fetchedAt).toLocaleTimeString()}</span>
              )}
              <button type="button" className="icon-btn" aria-label="Refresh feed" onClick={() => void refreshFeed(feed.id)}>
                <RefreshCw size={14} />
              </button>
              <button type="button" className="icon-btn" aria-label="Delete feed" onClick={() => void removeFeed(feed.id)}>
                <Trash2 size={14} />
              </button>
            </div>
          ))}
          <div className="feed-add">
            <input className="input" placeholder="Name (e.g. Work)" value={feedName} onChange={(e) => setFeedName(e.target.value)} />
            <input
              className="input"
              placeholder="https://calendar.google.com/…/basic.ics"
              value={feedUrl}
              onChange={(e) => setFeedUrl(e.target.value)}
            />
            <select className="input feed-kind-select" value={feedKind} onChange={(e) => setFeedKind(e.target.value as "calendar" | "links")} aria-label="Feed kind">
              <option value="calendar">→ calendar</option>
              <option value="links">→ links</option>
            </select>
            <button type="button" className="btn btn-primary" disabled={adding || !feedUrl.trim()} onClick={() => void submitFeed()}>
              <Plus size={14} /> Add
            </button>
          </div>
          <p className="panel-note" style={{ marginTop: 10 }}>
            Or import a file once:
            <input
              ref={fileRef}
              type="file"
              accept=".ics,text/calendar"
              style={{ display: "none" }}
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) void importIcs(file);
                e.target.value = "";
              }}
            />
            <button type="button" className="btn btn-secondary" style={{ marginLeft: 8 }} onClick={() => fileRef.current?.click()}>
              <Upload size={14} /> Import .ics
            </button>
          </p>
        </section>

        <section className="panel">
          <h3>Connected calendars</h3>
          <p className="panel-note">
            Two-way sync over CalDAV — Nextcloud, Radicale, Baikal, Fastmail (app password). Events on a
            connected calendar stay editable in both directions. Paste the collection URL
            (e.g. <code>https://cloud.example/remote.php/dav/calendars/you/personal/</code>).
          </p>
          {accounts.map((a) => (
            <div key={a.id} className="feed-row">
              <span className="swatch" style={{ "--swatch": `var(--c-${a.color})` } as React.CSSProperties} />
              <div className="feed-meta">
                <span className="feed-name">{a.name}</span>
                <span className="feed-url">{a.url}</span>
              </div>
              {a.lastSynced && <span className="feed-age">synced {new Date(a.lastSynced).toLocaleTimeString()}</span>}
              <button type="button" className="icon-btn" aria-label="Sync now" onClick={() => void syncAccount(a.id)}>
                <RefreshCw size={14} />
              </button>
              <button type="button" className="icon-btn" aria-label="Disconnect" onClick={() => void removeAccount(a.id)}>
                <Trash2 size={14} />
              </button>
            </div>
          ))}
          <div className="feed-add" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <input className="input" placeholder="Name" value={davName} onChange={(e) => setDavName(e.target.value)} />
            <input className="input" placeholder="Calendar URL" value={davUrl} onChange={(e) => setDavUrl(e.target.value)} />
            <input className="input" placeholder="Username" value={davUser} onChange={(e) => setDavUser(e.target.value)} />
            <input
              className="input"
              type="password"
              placeholder="Password / app password"
              value={davPass}
              onChange={(e) => setDavPass(e.target.value)}
            />
          </div>
          {discovered.length > 0 && (
            <div className="discover-list">
              {discovered.map((c) => (
                <button
                  key={c.href}
                  type="button"
                  className="feed-row as-btn"
                  onClick={() => {
                    setDavUrl(c.href);
                    if (!davName) setDavName(c.name);
                  }}
                >
                  <div className="feed-meta">
                    <span className="feed-name">{c.name}</span>
                    <span className="feed-url">{c.href}</span>
                  </div>
                </button>
              ))}
            </div>
          )}
          <div style={{ marginTop: 8, display: "flex", justifyContent: "flex-end", gap: 8 }}>
            <button
              type="button"
              className="btn btn-secondary"
              disabled={discovering || !davUrl.trim() || !davUser.trim()}
              onClick={() => void runDiscover()}
            >
              {discovering ? "Discovering…" : "Discover calendars"}
            </button>
            <button
              type="button"
              className="btn btn-primary"
              disabled={addingAccount || !davUrl.trim()}
              onClick={() => void submitAccount()}
            >
              <Plus size={14} /> {addingAccount ? "Connecting…" : "Connect"}
            </button>
          </div>
          <p className="panel-note" style={{ marginTop: 8 }}>
            Proton Calendar does not expose CalDAV outside Bridge — import via .ics instead. For Google,
            connect below (OAuth) or use its .ics secret address above.
          </p>
        </section>

        <section className="panel">
          <h3>Google Calendar</h3>
          {google === undefined ? null : google.connected ? (
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <span className="feed-name">Connected — events sync every 15 min as the "Google" feed.</span>
              <span className="spacer" style={{ flex: 1 }} />
              <button type="button" className="btn btn-secondary" onClick={() => void api.googleSync().then(() => toast("Synced"))}>
                Sync now
              </button>
              <button
                type="button"
                className="btn btn-secondary"
                onClick={() => void api.googleDisconnect().then(() => { setGoogle({ connected: false }); void loadFeeds(); })}
              >
                Disconnect
              </button>
            </div>
          ) : (
            <div>
              <p className="panel-note">
                Requires <code>GOOGLE_CLIENT_ID</code>/<code>GOOGLE_CLIENT_SECRET</code> on the API
                (Google Cloud → OAuth consent → Calendar read scope). Events sync read-only into a
                "Google" feed — toggleable like any other feed.
              </p>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() =>
                  void api
                    .googleConnect()
                    .then((r) => window.open(r.url, "_blank", "noopener"))
                    .catch((e) => toast(e instanceof Error ? e.message : "Google not configured"))
                }
              >
                Connect Google
              </button>
            </div>
          )}
        </section>

        <section className="panel">
          <h3>Birthdays (CardDAV)</h3>
          <p className="panel-note">
            Connect an addressbook (Nextcloud contacts, Radicale, Baikal) — contacts with a birthday
            become yearly all-day events.
          </p>
          <div className="feed-add" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <input className="input" placeholder="Name" value={cdName} onChange={(e) => setCdName(e.target.value)} />
            <input className="input" placeholder="Addressbook URL" value={cdUrl} onChange={(e) => setCdUrl(e.target.value)} />
            <input className="input" placeholder="Username" value={cdUser} onChange={(e) => setCdUser(e.target.value)} />
            <input
              className="input"
              type="password"
              placeholder="Password / app password"
              value={cdPass}
              onChange={(e) => setCdPass(e.target.value)}
            />
          </div>
          <div style={{ marginTop: 8, display: "flex", justifyContent: "flex-end" }}>
            <button type="button" className="btn btn-primary" disabled={!cdUrl.trim()} onClick={() => void submitCarddav()}>
              <Plus size={14} /> Import birthdays
            </button>
          </div>
        </section>

        <section className="panel">
          <h3>Webhooks</h3>
          <p className="panel-note">
            Cal POSTs <code>{"{event, entry}"}</code> to each URL on create/update/delete, signed with
            HMAC-SHA256 in <code>X-Cal-Signature</code> (verify with the secret shown once).
          </p>
          {webhooks.map((w) => (
            <div key={w.id} className="feed-row">
              <div className="feed-meta">
                <span className="feed-name">{w.url}</span>
                <span className="feed-url">secret: {w.secret}</span>
              </div>
              <button type="button" className="icon-btn" aria-label="Remove webhook" onClick={() => void removeWebhook(w.id)}>
                <Trash2 size={14} />
              </button>
            </div>
          ))}
          <div className="feed-add">
            <input className="input" placeholder="https://example.com/hook" value={hookUrl} onChange={(e) => setHookUrl(e.target.value)} />
            <button type="button" className="btn btn-primary" disabled={!hookUrl.trim()} onClick={() => void submitWebhook()}>
              <Plus size={14} /> Add
            </button>
          </div>
          <p className="panel-note" style={{ marginTop: 8 }}>
            Email-to-task: forward mail to <code>POST /api/intake?token=&lt;api token&gt;</code> with
            {" "}<code>{"{subject, text}"}</code> — lands as a task tagged <code>inbox</code>.
          </p>
        </section>

        <section className="panel">
          <h3>Holidays</h3>
          <div className="settings-grid">
            <label className="switch-row">
              Show holidays
              <span className="switch">
                <input
                  type="checkbox"
                  checked={settings.showHolidays}
                  onChange={(e) => set({ showHolidays: e.target.checked })}
                />
                <i />
              </span>
            </label>
            <label className="field">
              <span>Region</span>
              <select className="select" value={settings.country} onChange={(e) => set({ country: e.target.value })}>
                {countries.length === 0 && <option value={settings.country}>{settings.country}</option>}
                {countries.map((country) => (
                  <option key={country.code} value={country.code}>
                    {country.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </section>

        <section className="panel">
          <h3>Notifications</h3>
          <p className="panel-note">
            Entries with a reminder fire a notification at start − lead time. Push works even when the
            tab is closed; in-app notifications cover the open app.
          </p>
          <button
            type="button"
            className="btn btn-secondary"
            onClick={() => {
              if (pushOn) {
                void disablePush(usePlanner.getState().api).then(() => setPushOn(false));
              } else {
                void enablePush(usePlanner.getState().api)
                  .then((ok) => {
                    setPushOn(ok);
                    toast(ok ? "Push notifications enabled" : "Notifications not available here");
                  })
                  .catch(() => toast("Could not enable notifications"));
              }
            }}
          >
            {pushOn ? <BellOff size={14} /> : <Bell size={14} />}
            {pushOn ? "Disable push notifications" : "Enable push notifications"}
          </button>
          {pushDevices.length > 0 && (
            <div style={{ marginTop: 12 }}>
              <span className="panel-note" style={{ display: "block", marginBottom: 6 }}>
                Devices receiving push:
              </span>
              {pushDevices.map((d) => (
                <div key={d.id} className="feed-row">
                  <div className="feed-meta">
                    <span className="feed-name">{d.label || "Device"}</span>
                    <span className="feed-url">{d.endpoint}</span>
                  </div>
                  <button
                    type="button"
                    className="icon-btn"
                    aria-label="Revoke device"
                    onClick={() => void removePushDevice(d.id)}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="panel">
          <h3>MCP / API access</h3>
          <p className="panel-note">
            Cal exposes a Model Context Protocol endpoint so assistants (Claude, Devin, others) can read and
            manage your planner. Token is full-access — keep it private.
          </p>
          <div className="token-row">
            <code className="token">{settings.apiToken}</code>
            <button type="button" className="icon-btn" aria-label="Copy token" onClick={() => copy(settings.apiToken, "Token")}>
              <Copy size={14} />
            </button>
            <button
              type="button"
              className="icon-btn"
              aria-label="Rotate token"
              onClick={() => void rotateToken("api")}
            >
              <RefreshCw size={14} />
            </button>
          </div>
          <pre className="code-block" tabIndex={0}>{mcpConfig}</pre>
          <button type="button" className="btn btn-secondary" onClick={() => copy(mcpConfig, "Config")}>
            <Copy size={14} /> Copy client config
          </button>
        </section>

        <section className="panel">
          <h3>Today widget</h3>
          <p className="panel-note">
            A minimal read-only agenda for embedding in dashboards (Homepage, Heimdall, an iframe, a TV).
          </p>
          <div className="token-row">
            <code className="token">{widgetUrl}</code>
            <button type="button" className="icon-btn" aria-label="Copy widget URL" onClick={() => copy(widgetUrl, "URL")}>
              <Copy size={14} />
            </button>
            <button
              type="button"
              className="icon-btn"
              aria-label="Rotate widget token"
              onClick={() => void rotateToken("widget")}
            >
              <RefreshCw size={14} />
            </button>
          </div>
          <pre className="code-block" tabIndex={0}>{`<iframe src="${widgetUrl}" style="border:0;width:100%;height:320px"></iframe>`}</pre>
          <p className="panel-note" style={{ marginTop: 10 }}>
            Or subscribe to Cal itself in any calendar app — this URL serves a live .ics feed of your tasks and
            events:
          </p>
          <div className="token-row">
            <code className="token">{icsUrl}</code>
            <button type="button" className="icon-btn" aria-label="Copy feed URL" onClick={() => copy(icsUrl, "URL")}>
              <Copy size={14} />
            </button>
          </div>
        </section>

        <section className="panel">
          <h3>Your data</h3>
          <p className="panel-note">
            {stats.task} tasks ({stats.done} done), {stats.event} events, {stats.note} notes, {stats.link}{" "}
            links — stored in your own database. Take them with you any time.
          </p>
          {storage && (
            <p className="panel-note">
              Attachments: {(storage.usedBytes / 1048576).toFixed(1)} MB of{" "}
              {(storage.quotaBytes / 1048576).toFixed(0)} MB used.
            </p>
          )}
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <a className="btn btn-secondary" href="/api/export" download>
              <Download size={14} /> Export everything (JSON)
            </a>
            <a className="btn btn-secondary" href="/api/agenda?days=7" target="_blank" rel="noreferrer">
              <FileText size={14} /> Week agenda (markdown)
            </a>
            <input
              ref={restoreRef}
              type="file"
              accept=".json,application/json"
              style={{ display: "none" }}
              onChange={(e) => {
                const file = e.target.files?.[0];
                if (file) void restore(file);
                e.target.value = "";
              }}
            />
            <button type="button" className="btn btn-secondary" onClick={() => restoreRef.current?.click()}>
              <Upload size={14} /> Restore from backup
            </button>
          </div>
          <p className="panel-note" style={{ marginTop: 10 }}>
            A restore merges — entries already present are kept, missing ones come back. The server also writes
            a nightly backup to <code>DATA_DIR/backups/</code> (14 days kept).
          </p>
        </section>

        <section className="panel">
          <h3>Account</h3>
          <p className="panel-note">{user?.email}</p>
          <button type="button" className="btn btn-secondary" onClick={() => void logout()}>
            <LogOut size={14} /> Sign out
          </button>
        </section>

        <section className="panel">
          <h3>Security</h3>
          <div className="feed-add" style={{ gridTemplateColumns: "1fr 1fr auto" }}>
            <input
              className="input"
              type="password"
              placeholder="Current password"
              value={pwCurrent}
              onChange={(e) => setPwCurrent(e.target.value)}
            />
            <input
              className="input"
              type="password"
              placeholder="New password (8+ chars)"
              value={pwNext}
              onChange={(e) => setPwNext(e.target.value)}
            />
            <button
              type="button"
              className="btn btn-primary"
              disabled={pwSaving || pwCurrent === "" || pwNext.length < 8}
              onClick={() => void submitPassword()}
            >
              {pwSaving ? "Saving…" : "Change"}
            </button>
          </div>
          <p className="panel-note" style={{ marginTop: 10 }}>
            Changing the password signs out every other session.
          </p>
          <h4 style={{ margin: "14px 0 8px" }}>Active sessions</h4>
          {sessions.map((s) => (
            <div key={s.id} className="feed-row">
              <div className="feed-meta">
                <span className="feed-name">
                  {s.current ? "This device" : s.userAgent || "Unknown device"}
                </span>
                <span className="feed-url">
                  {s.lastSeen ? `active ${new Date(s.lastSeen).toLocaleString()}` : `since ${new Date(s.createdAt).toLocaleDateString()}`}
                  {!s.current && s.userAgent ? ` — ${s.userAgent.slice(0, 60)}` : ""}
                </span>
              </div>
              {!s.current && (
                <button type="button" className="icon-btn" aria-label="Revoke session" onClick={() => void revokeSession(s.id)}>
                  <Trash2 size={14} />
                </button>
              )}
            </div>
          ))}
        </section>
      </div>
    </>
  );
}

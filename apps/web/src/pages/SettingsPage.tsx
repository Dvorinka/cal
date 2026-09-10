import type { Accent } from "@cal/api-client";
import { Copy, Download, LogOut, Plus, RefreshCw, Trash2, Upload } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { usePlanner } from "../stores/planner";

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
  const removeFeed = usePlanner((state) => state.removeFeed);
  const refreshFeed = usePlanner((state) => state.refreshFeed);
  const importIcs = usePlanner((state) => state.importIcs);
  const rotateToken = usePlanner((state) => state.rotateToken);
  const toast = usePlanner((state) => state.toast);
  const user = usePlanner((state) => state.user);
  const logout = usePlanner((state) => state.logout);

  const [feedName, setFeedName] = useState("");
  const [feedUrl, setFeedUrl] = useState("");
  const [adding, setAdding] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    void loadEntries({});
    void loadFeeds();
  }, [loadEntries, loadFeeds]);

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

  async function submitFeed() {
    if (!feedUrl.trim()) return;
    setAdding(true);
    const ok = await addFeed({ name: feedName.trim(), url: feedUrl.trim() });
    setAdding(false);
    if (ok) {
      setFeedName("");
      setFeedUrl("");
    }
  }

  const widgetUrl = `${location.origin}/widget/today?token=${settings.widgetToken}`;
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
          <h3>Calendar feeds</h3>
          <p className="panel-note">
            Subscribe to any iCalendar (.ics) URL — Google Calendar's secret address, iCloud public
            calendars, Nextcloud shared links, Outlook published calendars. Read-only, refreshed on demand.
          </p>
          {feeds.map((feed) => (
            <div key={feed.id} className="feed-row">
              <span className="swatch" style={{ "--swatch": `var(--c-${feed.color})` } as React.CSSProperties} />
              <div className="feed-meta">
                <span className="feed-name">{feed.name}</span>
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
          <pre className="code-block">{mcpConfig}</pre>
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
          <pre className="code-block">{`<iframe src="${widgetUrl}" style="border:0;width:100%;height:320px"></iframe>`}</pre>
        </section>

        <section className="panel">
          <h3>Your data</h3>
          <p className="panel-note">
            {stats.task} tasks ({stats.done} done), {stats.event} events, {stats.note} notes, {stats.link}{" "}
            links — stored in your own database. Take them with you any time.
          </p>
          <a className="btn btn-secondary" href="/api/export" download>
            <Download size={14} /> Export everything (JSON)
          </a>
        </section>

        <section className="panel">
          <h3>Account</h3>
          <p className="panel-note">{user?.email}</p>
          <button type="button" className="btn btn-secondary" onClick={() => void logout()}>
            <LogOut size={14} /> Sign out
          </button>
        </section>
      </div>
    </>
  );
}

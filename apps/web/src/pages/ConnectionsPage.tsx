import { CalendarDays, CalendarRange, Clapperboard, GitPullRequest, Link2, Mail, Play, Plug, Rss, Unplug, Users, Webhook } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { reportErr, usePlanner } from "../stores/planner";

interface Remote {
  google: boolean | null;
  carddav: number | null;
  hooks: number | null;
  mail: number | null;
}

const EMPTY: Remote = { google: null, carddav: null, hooks: null, mail: null };

function hostOf(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

export function ConnectionsPage() {
  const api = usePlanner((state) => state.api);
  const settings = usePlanner((state) => state.settings);
  const feeds = usePlanner((state) => state.feeds);
  const accounts = usePlanner((state) => state.accounts);
  const loadFeeds = usePlanner((state) => state.loadFeeds);
  const loadAccounts = usePlanner((state) => state.loadAccounts);
  const [remote, setRemote] = useState<Remote>(EMPTY);

  useEffect(() => {
    void loadFeeds();
    void loadAccounts();
    void api.googleStatus().then((g) => setRemote((s) => ({ ...s, google: g.connected }))).catch(reportErr("Could not load Google status"));
    void api.carddavAccounts().then((rows) => setRemote((s) => ({ ...s, carddav: rows.length }))).catch(reportErr("Could not load address books"));
    void api.webhooks().then((rows) => setRemote((s) => ({ ...s, hooks: rows.length }))).catch(reportErr("Could not load webhooks"));
    void api.mailAccounts().then((rows) => setRemote((s) => ({ ...s, mail: rows.length }))).catch(reportErr("Could not load mail"));
  }, [api, loadFeeds, loadAccounts]);

  const calendars = feeds.filter((feed) => feed.kind !== "links").length;
  const channels = feeds.filter((feed) => feed.kind === "links").length;

  const cards = [
    {
      key: "feeds",
      icon: Rss,
      title: "Subscribed calendars",
      detail: "ICS and RSS dates on the calendar.",
      status: calendars ? `${calendars} live` : "None",
      on: calendars > 0,
      to: "/settings#set-feeds",
    },
    {
      key: "channels",
      icon: Clapperboard,
      title: "Channel feeds",
      detail: "RSS items saved as link cards.",
      status: channels ? `${channels} live` : "None",
      on: channels > 0,
      to: "/settings#set-feeds",
    },
    {
      key: "caldav",
      icon: CalendarRange,
      title: "CalDAV",
      detail: "Two-way sync with a calendar server.",
      status: accounts.length ? `${accounts.length} account${accounts.length === 1 ? "" : "s"}` : "Not connected",
      on: accounts.length > 0,
      to: "/settings#set-calendars",
    },
    {
      key: "google",
      icon: CalendarDays,
      title: "Google Calendar",
      detail: "Read-only events, refreshed every 15 minutes.",
      status: remote.google === null ? "Checking" : remote.google ? "Connected" : "Not connected",
      on: remote.google === true,
      to: "/settings#set-google",
    },
    {
      key: "carddav",
      icon: Users,
      title: "CardDAV",
      detail: "Birthdays, and import as people.",
      status: remote.carddav === null ? "Checking" : remote.carddav ? `${remote.carddav} address book${remote.carddav === 1 ? "" : "s"}` : "Not connected",
      on: (remote.carddav ?? 0) > 0,
      to: "/settings#set-carddav",
    },
    {
      key: "github",
      icon: GitPullRequest,
      title: "GitHub",
      detail: "Issues and pull requests become board cards.",
      status: settings.githubToken ? "Token saved" : "No token",
      on: Boolean(settings.githubToken),
      to: settings.githubToken ? "/github" : "/settings#set-integrations",
    },
    {
      key: "mail",
      icon: Mail,
      title: "Mail",
      detail: "IMAP inbox and SMTP send.",
      status: remote.mail === null ? "Checking" : remote.mail ? `${remote.mail} account${remote.mail === 1 ? "" : "s"}` : "Not connected",
      on: (remote.mail ?? 0) > 0,
      to: "/mail",
    },
    {
      key: "invidious",
      icon: Play,
      title: "Invidious",
      detail: "YouTube search without a Google key.",
      status: settings.invidiousUrl ? hostOf(settings.invidiousUrl) : "Off",
      on: Boolean(settings.invidiousUrl),
      to: "/settings#set-integrations",
    },
    {
      key: "hooks",
      icon: Webhook,
      title: "Webhooks",
      detail: "Signed POST on every entry change. Email intake lives here too.",
      status: remote.hooks === null ? "Checking" : remote.hooks ? `${remote.hooks} endpoint${remote.hooks === 1 ? "" : "s"}` : "None",
      on: (remote.hooks ?? 0) > 0,
      to: "/settings#set-webhooks",
    },
    {
      key: "mcp",
      icon: Plug,
      title: "Agents",
      detail: "MCP endpoint so an assistant can drive the planner.",
      status: "Token ready",
      on: true,
      to: "/settings#set-mcp",
    },
    {
      key: "ics",
      icon: Link2,
      title: "Outbound calendar",
      detail: "Subscribe elsewhere with your read-only ICS feed.",
      status: "Ready",
      on: true,
      to: "/settings#set-widget",
    },
  ];

  const builtIn = new Set(["mcp", "ics"]);
  const live = cards.filter((card) => card.on && !builtIn.has(card.key)).length;

  return (
    <>
      <PageHeader title="Connections" sub={live === 0 ? "Nothing connected yet" : `${live} connected`}>
        <Link className="btn btn-secondary" to="/settings#set-integrations">
          <Unplug size={14} /> Configure
        </Link>
      </PageHeader>
      <div className="page-scroll" data-testid="connections-page">
        <div className="connect-grid">
          {cards.map((card) => {
            const Icon = card.icon;
            return (
              <Link key={card.key} className="connect-card" to={card.to}>
                <span className={`connect-status ${card.on ? "on" : "off"}`}>{card.status}</span>
                <strong>
                  <Icon size={14} /> {card.title}
                </strong>
                <span>{card.detail}</span>
              </Link>
            );
          })}
        </div>
      </div>
    </>
  );
}

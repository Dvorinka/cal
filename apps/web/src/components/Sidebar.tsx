import { CalendarDays, ChevronLeft, ChevronRight, FolderOpen, Hash, Link2, ListChecks, Plus, Search, Settings2, StickyNote, Sun, Trash2, Trello } from "lucide-react";
import { TimerPill } from "./TimerPill";
import { useEffect, useMemo, useState } from "react";
import { NavLink, useNavigate } from "react-router-dom";
import { addMonths, fromIso, iso, monthMatrix, todayIso, weekdayNames, type WeekStartPref } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

const FILTERS = [
  { type: "task", label: "Tasks", color: "var(--c-mint)" },
  { type: "event", label: "Events", color: "var(--c-orange)" },
  { type: "note", label: "Notes", color: "var(--c-sky)" },
  { type: "link", label: "Links", color: "var(--c-violet)" },
] as const;

const NAV: { to: string; label: string; icon: typeof Sun; end?: boolean }[] = [
  { to: "/today", label: "Today", icon: Sun },
  { to: "/", label: "Calendar", icon: CalendarDays, end: true },
  { to: "/tasks", label: "Tasks", icon: ListChecks },
  { to: "/notes", label: "Notes", icon: StickyNote },
  { to: "/links", label: "Links", icon: Link2 },
  { to: "/files", label: "Files", icon: FolderOpen },
  { to: "/boards", label: "Boards", icon: Trello },
  { to: "/tags", label: "Tags", icon: Hash },
];

function MiniMonth({ weekStart }: { weekStart: WeekStartPref }) {
  const selectedDate = useUi((state) => state.selectedDate);
  const selectDate = useUi((state) => state.selectDate);
  const closeSidebar = useUi((state) => state.closeSidebar);
  const navigate = useNavigate();
  const entries = usePlanner((state) => state.entries);
  const [cursor, setCursor] = useState(() => fromIso(selectedDate));
  const today = todayIso();

  useEffect(() => {
    setCursor((current) =>
      current.getMonth() === fromIso(selectedDate).getMonth() && current.getFullYear() === fromIso(selectedDate).getFullYear()
        ? current
        : fromIso(selectedDate),
    );
  }, [selectedDate]);

  const days = useMemo(() => monthMatrix(cursor, weekStart), [cursor, weekStart]);
  const names = useMemo(() => weekdayNames(weekStart, "narrow"), [weekStart]);
  const counts = useMemo(() => {
    const map = new Map<string, number>();
    for (const entry of entries) map.set(entry.date, (map.get(entry.date) ?? 0) + 1);
    return map;
  }, [entries]);

  return (
    <div className="mini-month">
      <div className="mini-month-head">
        <strong>{cursor.toLocaleDateString(undefined, { month: "long", year: "numeric" })}</strong>
        <span style={{ display: "flex", gap: 2 }}>
          <button type="button" className="icon-btn" aria-label="Previous month" onClick={() => setCursor(addMonths(cursor, -1))}>
            <ChevronLeft size={14} />
          </button>
          <button type="button" className="icon-btn" aria-label="Next month" onClick={() => setCursor(addMonths(cursor, 1))}>
            <ChevronRight size={14} />
          </button>
        </span>
      </div>
      <div className="mini-month-grid">
        {names.map((name, i) => (
          <span key={i} className="mini-dow">
            {name}
          </span>
        ))}
        {days.map((day) => {
          const date = iso(day);
          const outside = day.getMonth() !== cursor.getMonth();
          const count = counts.get(date) ?? 0;
          return (
            <button
              key={date}
              type="button"
              className={`mini-day ${outside ? "outside" : ""} ${date === today ? "today" : ""} ${date === selectedDate ? "selected" : ""}`}
              onClick={() => {
                selectDate(date);
                closeSidebar();
                navigate("/");
              }}
            >
              {day.getDate()}
              {count > 0 && (
                <span className="dots-row">
                  {Array.from({ length: Math.min(3, count) }).map((_, i) => (
                    <i key={i} />
                  ))}
                </span>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}



export function Sidebar() {
  const user = usePlanner((state) => state.user);
  const entries = usePlanner((state) => state.entries);
  const feeds = usePlanner((state) => state.feeds);
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const openPalette = useUi((state) => state.openPalette);
  const openCreate = useUi((state) => state.openCreate);
  const selectedDate = useUi((state) => state.selectedDate);
  const hiddenTypes = useUi((state) => state.hiddenTypes);
  const hiddenFeeds = useUi((state) => state.hiddenFeeds);
  const toggleType = useUi((state) => state.toggleType);
  const toggleFeed = useUi((state) => state.toggleFeed);
  const sidebarOpen = useUi((state) => state.sidebarOpen);
  const closeSidebar = useUi((state) => state.closeSidebar);

  const counts = useMemo(() => {
    const map: Record<string, number> = { task: 0, event: 0, note: 0, link: 0 };
    for (const entry of entries) map[entry.type] = (map[entry.type] ?? 0) + 1;
    return map;
  }, [entries]);

  return (
    <>
      {sidebarOpen && <div className="sidebar-scrim" onClick={closeSidebar} />}
      <aside className={`sidebar ${sidebarOpen ? "open" : ""}`}>
        <div className="brand">
          <CalendarDays size={19} strokeWidth={2.2} />
          <span>Cal</span>
          <span className="user" title={user?.email}>
            {user?.email}
          </span>
        </div>

        <button
          type="button"
          className="btn btn-primary new-btn"
          onClick={() => {
            closeSidebar();
            openCreate(selectedDate);
          }}
        >
          <Plus size={15} strokeWidth={2.5} /> New entry
        </button>

        <button
          type="button"
          className="search-box"
          onClick={() => {
            closeSidebar();
            openPalette();
          }}
        >
          <Search size={14} />
          <span>Search</span>
          <kbd>⌘K</kbd>
        </button>

        <nav className="side-nav" aria-label="Primary">
          {NAV.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) => `nav-item ${isActive ? "active" : ""}`}
              onClick={closeSidebar}
            >
              <Icon size={15} />
              {label}
            </NavLink>
          ))}
        </nav>

        <MiniMonth weekStart={settings.weekStart} />

        <div className="side-label">Lists</div>
        <div className="filter-list">
          {FILTERS.map(({ type, label, color }) => (
            <button
              key={type}
              type="button"
              className={`filter-item ${hiddenTypes.has(type) ? "off" : ""}`}
              onClick={() => toggleType(type)}
            >
              <span className="swatch" style={{ "--swatch": color } as React.CSSProperties} />
              {label}
              <span className="count">{counts[type]}</span>
            </button>
          ))}
          <button
            type="button"
            className={`filter-item ${settings.showHolidays ? "" : "off"}`}
            onClick={() => void updateSettings({ ...settings, showHolidays: !settings.showHolidays })}
          >
            <span className="swatch" style={{ "--swatch": "var(--accent)" } as React.CSSProperties} />
            Holidays
            <span className="count">{settings.showHolidays ? "on" : "off"}</span>
          </button>
        </div>

        {feeds.length > 0 && (
          <>
            <div className="side-label">Calendars</div>
            <div className="filter-list">
              {feeds.map((feed) => (
                <button
                  key={feed.id}
                  type="button"
                  className={`filter-item ${hiddenFeeds.has(feed.id) ? "off" : ""}`}
                  onClick={() => toggleFeed(feed.id)}
                  title={feed.url}
                >
                  <span className="swatch" style={{ "--swatch": `var(--c-${feed.color})` } as React.CSSProperties} />
                  {feed.name}
                </button>
              ))}
            </div>
          </>
        )}

        <TimerPill />

        <div className="sidebar-footer">
          <NavLink to="/trash" className="icon-btn" aria-label="Trash" onClick={closeSidebar}>
            <Trash2 size={15} />
          </NavLink>
          <NavLink to="/settings" className="icon-btn" aria-label="Settings" onClick={closeSidebar}>
            <Settings2 size={16} />
          </NavLink>
        </div>
      </aside>
    </>
  );
}

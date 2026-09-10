import { CalendarDays, ChevronLeft, ChevronRight, LogOut, Plus, Search, Settings2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { addMonths, fromIso, iso, monthMatrix, todayIso, weekdayNames, type WeekStartPref } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

const FILTERS = [
  { type: "task", label: "Tasks", color: "var(--c-mint)" },
  { type: "note", label: "Notes", color: "var(--c-sky)" },
  { type: "link", label: "Links", color: "var(--c-violet)" },
] as const;

function MiniMonth({ weekStart }: { weekStart: WeekStartPref }) {
  const selectedDate = useUi((state) => state.selectedDate);
  const selectDate = useUi((state) => state.selectDate);
  const closeSidebar = useUi((state) => state.closeSidebar);
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

function SettingsPopover() {
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const countries = usePlanner((state) => state.countries);
  const logout = usePlanner((state) => state.logout);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false);
    };
    window.addEventListener("mousedown", close);
    return () => window.removeEventListener("mousedown", close);
  }, [open]);

  const set = (patch: Partial<typeof settings>) => void updateSettings({ ...settings, ...patch });

  return (
    <div className="popover-wrap" ref={ref}>
      {open && (
        <div className="popover" role="dialog" aria-label="Settings">
          <label className="field">
            <span>Theme</span>
            <select className="select" value={settings.theme} onChange={(e) => set({ theme: e.target.value as typeof settings.theme })}>
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
          <label className="switch-row">
            Show holidays
            <span className="switch">
              <input type="checkbox" checked={settings.showHolidays} onChange={(e) => set({ showHolidays: e.target.checked })} />
              <i />
            </span>
          </label>
          <label className="field">
            <span>Holiday region</span>
            <select className="select" value={settings.country} onChange={(e) => set({ country: e.target.value })}>
              {countries.length === 0 && <option value={settings.country}>{settings.country}</option>}
              {countries.map((country) => (
                <option key={country.code} value={country.code}>
                  {country.name}
                </option>
              ))}
            </select>
          </label>
          <button type="button" className="btn btn-secondary" onClick={() => void logout()}>
            <LogOut size={13} /> Sign out
          </button>
        </div>
      )}
      <button type="button" className="icon-btn" aria-label="Settings" onClick={() => setOpen(!open)}>
        <Settings2 size={16} />
      </button>
    </div>
  );
}

export function Sidebar() {
  const user = usePlanner((state) => state.user);
  const entries = usePlanner((state) => state.entries);
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const openPalette = useUi((state) => state.openPalette);
  const openCreate = useUi((state) => state.openCreate);
  const selectedDate = useUi((state) => state.selectedDate);
  const hiddenTypes = useUi((state) => state.hiddenTypes);
  const toggleType = useUi((state) => state.toggleType);
  const sidebarOpen = useUi((state) => state.sidebarOpen);
  const closeSidebar = useUi((state) => state.closeSidebar);

  const counts = useMemo(() => {
    const map: Record<string, number> = { task: 0, note: 0, link: 0 };
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

        <div className="sidebar-footer">
          <SettingsPopover />
        </div>
      </aside>
    </>
  );
}

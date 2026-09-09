import { AnimatePresence, motion } from "framer-motion";
import { CalendarDays, ChevronLeft, ChevronRight, LogOut, Moon, Search, Sun } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { AuthPanel } from "./components/AuthPanel";
import { CalendarGrid } from "./components/CalendarGrid";
import { EntryComposer } from "./components/EntryComposer";
import { addDays, formatHeading, rangeFor, todayIso, type CalendarView } from "./lib/date";
import { usePlanner } from "./stores/planner";

export function App() {
  const {
    user,
    entries,
    holidays,
    settings,
    bootstrap,
    loadEntries,
    createEntry,
    updateEntry,
    deleteEntry,
    loadHolidays,
    updateSettings,
    logout,
    error,
  } = usePlanner();
  const [view, setView] = useState<CalendarView>("month");
  const [anchor, setAnchor] = useState(() => new Date());
  const [selectedDate, setSelectedDate] = useState(todayIso());
  const [query, setQuery] = useState("");

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  const range = useMemo(() => rangeFor(view, anchor), [view, anchor]);

  useEffect(() => {
    if (!user) return;
    void loadEntries({ ...range, q: query });
  }, [user, range.from, range.to, query, loadEntries]);

  useEffect(() => {
    if (!settings.showHolidays) return;
    void loadHolidays(settings.country, anchor.getFullYear());
  }, [settings.country, settings.showHolidays, anchor, loadHolidays]);

  useEffect(() => {
    document.documentElement.dataset.theme = settings.theme;
  }, [settings.theme]);

  if (!user) return <AuthPanel />;

  function shift(direction: -1 | 1) {
    const days = view === "day" ? 1 : view === "week" ? 7 : 32;
    setAnchor((current) => addDays(current, direction * days));
  }

  function showToday() {
    const next = new Date();
    setAnchor(next);
    setSelectedDate(todayIso());
  }

  const selectedEntries = entries.filter((entry) => entry.date === selectedDate);

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <CalendarDays size={22} />
          <span>Cal</span>
        </div>
        <div className="search">
          <Search size={16} />
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search" aria-label="Search entries" />
        </div>
        <nav className="view-tabs" aria-label="Calendar views">
          {(["month", "week", "day"] as const).map((item) => (
            <button key={item} className={view === item ? "active" : ""} type="button" onClick={() => setView(item)}>
              {item}
            </button>
          ))}
        </nav>
        <section className="side-section">
          <h2>{selectedDate}</h2>
          <EntryComposer date={selectedDate} onCreate={createEntry} />
          <div className="mini-list">
            <AnimatePresence initial={false}>
              {selectedEntries.map((entry) => (
                <motion.p key={entry.id} initial={{ opacity: 0, y: 4 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0 }}>
                  {entry.title}
                </motion.p>
              ))}
            </AnimatePresence>
          </div>
        </section>
        <section className="settings">
          <label>
            Country
            <select value={settings.country} onChange={(event) => void updateSettings({ ...settings, country: event.target.value })}>
              <option value="US">US</option>
              <option value="CZ">CZ</option>
            </select>
          </label>
          <label className="toggle">
            <input
              type="checkbox"
              checked={settings.showHolidays}
              onChange={(event) => void updateSettings({ ...settings, showHolidays: event.target.checked })}
            />
            Holidays
          </label>
          <button
            className="theme-toggle"
            type="button"
            onClick={() => void updateSettings({ ...settings, theme: settings.theme === "dark" ? "light" : "dark" })}
          >
            {settings.theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
            {settings.theme === "dark" ? "Light" : "Dark"}
          </button>
        </section>
        <button className="logout" type="button" onClick={() => void logout()}>
          <LogOut size={16} />
          Log out
        </button>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">{view}</p>
            <h1>{formatHeading(view, anchor)}</h1>
            {error && <p className="sync-note">{error}</p>}
          </div>
          <div className="nav-buttons">
            <button type="button" onClick={() => shift(-1)} aria-label="Previous">
              <ChevronLeft size={18} />
            </button>
            <button type="button" onClick={showToday}>
              Today
            </button>
            <button type="button" onClick={() => shift(1)} aria-label="Next">
              <ChevronRight size={18} />
            </button>
          </div>
        </header>
        <CalendarGrid
          view={view}
          anchor={anchor}
          entries={entries}
          holidays={settings.showHolidays ? holidays : []}
          selectedDate={selectedDate}
          onSelectDate={setSelectedDate}
          onMoveEntry={(id, date) => void updateEntry(id, { date })}
          onToggleEntry={(id, completed) => void updateEntry(id, { completed })}
          onDeleteEntry={(id) => void deleteEntry(id)}
        />
      </section>
    </main>
  );
}

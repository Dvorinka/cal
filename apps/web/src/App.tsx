import { CalendarDays } from "lucide-react";
import { useEffect, useMemo } from "react";
import { AuthPanel } from "./components/AuthPanel";
import { CommandPalette } from "./components/CommandPalette";
import { ContextMenu } from "./components/ContextMenu";
import { EntryEditor } from "./components/EntryEditor";
import { MonthView } from "./components/MonthView";
import { Sidebar } from "./components/Sidebar";
import { TimeGridView } from "./components/TimeGridView";
import { Toasts } from "./components/Toasts";
import { TopBar } from "./components/TopBar";
import { rangeFor } from "./lib/date";
import { usePlanner } from "./stores/planner";
import { useUi } from "./stores/ui";

export function App() {
  const { user, booted, entries, holidays, settings, bootstrap, loadEntries, updateEntry, loadHolidays } = usePlanner();
  const { view, anchor, selectedDate, hiddenTypes, paletteOpen, editor, contextMenu } = useUi();
  const { setView, shift, goToday, openPalette, closePalette, openCreate, setContextMenu } = useUi();

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

  const range = useMemo(() => rangeFor(view, anchor, settings.weekStart), [view, anchor, settings.weekStart]);

  useEffect(() => {
    if (!user) return;
    void loadEntries(range);
  }, [user, range.from, range.to, loadEntries]);

  useEffect(() => {
    if (!user || !settings.showHolidays) return;
    void loadHolidays(settings.country, anchor.getFullYear());
  }, [user, settings.country, settings.showHolidays, anchor, loadHolidays]);

  // Resolve theme: light | dark | system (follows prefers-color-scheme live).
  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      document.documentElement.dataset.theme = settings.theme === "system" ? (media.matches ? "dark" : "light") : settings.theme;
    };
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [settings.theme]);

  // Global keyboard shortcuts.
  useEffect(() => {
    if (!user) return;
    const onKey = (event: KeyboardEvent) => {
      const tag = (event.target as HTMLElement).tagName;
      const typing = tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        paletteOpen ? closePalette() : openPalette();
        return;
      }
      if (typing || paletteOpen || editor.mode !== "closed" || contextMenu) return;
      switch (event.key) {
        case "1":
          setView("month");
          break;
        case "2":
          setView("week");
          break;
        case "3":
          setView("day");
          break;
        case "t":
          goToday();
          break;
        case "c":
        case "n":
          openCreate(selectedDate);
          break;
        case "/":
          event.preventDefault();
          openPalette();
          break;
        case "ArrowLeft":
          shift(-1);
          break;
        case "ArrowRight":
          shift(1);
          break;
        case "Escape":
          setContextMenu(undefined);
          break;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [user, paletteOpen, editor.mode, contextMenu, selectedDate, setView, shift, goToday, openPalette, closePalette, openCreate, setContextMenu]);

  if (!booted) {
    return (
      <main className="splash">
        <div className="mark">
          <CalendarDays size={24} strokeWidth={2.2} />
        </div>
      </main>
    );
  }
  if (!user) return <AuthPanel />;

  const visible = entries.filter((entry) => !hiddenTypes.has(entry.type));
  const shownHolidays = settings.showHolidays ? holidays : [];

  return (
    <main className="app-shell">
      <Sidebar />
      <section className="workspace">
        <TopBar weekStart={settings.weekStart} />
        {view === "month" ? (
          <MonthView
            anchor={anchor}
            entries={visible}
            holidays={shownHolidays}
            weekStart={settings.weekStart}
            onMoveEntry={(id, date) => void updateEntry(id, { date })}
          />
        ) : (
          <TimeGridView
            view={view}
            anchor={anchor}
            entries={visible}
            holidays={shownHolidays}
            weekStart={settings.weekStart}
            onMoveEntry={(id, patch) => void updateEntry(id, patch)}
          />
        )}
      </section>
      <CommandPalette />
      <EntryEditor />
      <ContextMenu />
      <Toasts />
    </main>
  );
}

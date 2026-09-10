import { CalendarDays } from "lucide-react";
import { useEffect } from "react";
import { BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { AuthPanel } from "./components/AuthPanel";
import { BottomNav } from "./components/BottomNav";
import { CommandPalette } from "./components/CommandPalette";
import { ContextMenu } from "./components/ContextMenu";
import { EntryEditor } from "./components/EntryEditor";
import { Sidebar } from "./components/Sidebar";
import { Toasts } from "./components/Toasts";
import { CalendarPage } from "./pages/CalendarPage";
import { LinksPage } from "./pages/LinksPage";
import { NotesPage } from "./pages/NotesPage";
import { SettingsPage } from "./pages/SettingsPage";
import { TasksPage } from "./pages/TasksPage";
import { TodayPage } from "./pages/TodayPage";
import { usePlanner } from "./stores/planner";
import { useUi } from "./stores/ui";

function Shell() {
  const { user } = usePlanner();
  const { paletteOpen, editor, contextMenu, selectedDate } = useUi();
  const { setView, shift, goToday, openPalette, closePalette, openCreate, setContextMenu, closeSidebar } = useUi();
  const location = useLocation();
  const navigate = useNavigate();
  const onCalendar = location.pathname === "/";

  // Global keyboard shortcuts. View-specific ones apply on the calendar only.
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
        case "2":
        case "3":
          if (onCalendar) setView(event.key === "1" ? "month" : event.key === "2" ? "week" : "day");
          break;
        case "t":
          if (onCalendar) goToday();
          else navigate("/today");
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
        case "ArrowRight":
          if (onCalendar) shift(event.key === "ArrowLeft" ? -1 : 1);
          break;
        case "Escape":
          setContextMenu(undefined);
          closeSidebar();
          break;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [user, paletteOpen, editor.mode, contextMenu, selectedDate, onCalendar, navigate, setView, shift, goToday, openPalette, closePalette, openCreate, setContextMenu, closeSidebar]);

  if (!user) return <AuthPanel />;

  return (
    <main className="app-shell">
      <Sidebar />
      <section className="workspace">
        <Routes>
          <Route path="/" element={<CalendarPage />} />
          <Route path="/today" element={<TodayPage />} />
          <Route path="/tasks" element={<TasksPage />} />
          <Route path="/notes" element={<NotesPage />} />
          <Route path="/links" element={<LinksPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </section>
      <BottomNav />
      <CommandPalette />
      <EntryEditor />
      <ContextMenu />
      <Toasts />
    </main>
  );
}

export function App() {
  const { booted, bootstrap, settings } = usePlanner();

  useEffect(() => {
    void bootstrap();
  }, [bootstrap]);

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

  if (!booted) {
    return (
      <main className="splash">
        <div className="mark">
          <CalendarDays size={24} strokeWidth={2.2} />
        </div>
      </main>
    );
  }

  return (
    <BrowserRouter>
      <Shell />
    </BrowserRouter>
  );
}

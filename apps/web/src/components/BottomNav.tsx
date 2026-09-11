import { CalendarDays, Link2, ListChecks, StickyNote, Sun } from "lucide-react";
import { NavLink } from "react-router-dom";
import { moduleOn, type ModuleKey } from "../lib/modules";
import { usePlanner } from "../stores/planner";

const TABS: { to: string; label: string; icon: typeof Sun; end?: boolean; module?: ModuleKey }[] = [
  { to: "/today", label: "Today", icon: Sun },
  { to: "/", label: "Calendar", icon: CalendarDays, end: true },
  { to: "/tasks", label: "Tasks", icon: ListChecks, module: "tasks" },
  { to: "/notes", label: "Notes", icon: StickyNote, module: "notes" },
  { to: "/links", label: "Links", icon: Link2, module: "links" },
];

export function BottomNav() {
  const settings = usePlanner((state) => state.settings);
  return (
    <nav className="bottom-nav" aria-label="Primary">
      {TABS.filter((t) => !t.module || moduleOn(settings, t.module)).map(({ to, label, icon: Icon, end }) => (
        <NavLink key={to} to={to} end={end} className={({ isActive }) => `tab ${isActive ? "active" : ""}`}>
          <Icon size={18} />
          <span>{label}</span>
        </NavLink>
      ))}
    </nav>
  );
}

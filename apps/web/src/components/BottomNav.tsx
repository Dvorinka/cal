import { CalendarDays, Link2, ListChecks, StickyNote, Sun } from "lucide-react";
import { NavLink } from "react-router-dom";

const TABS = [
  { to: "/today", label: "Today", icon: Sun },
  { to: "/", label: "Calendar", icon: CalendarDays, end: true },
  { to: "/tasks", label: "Tasks", icon: ListChecks },
  { to: "/notes", label: "Notes", icon: StickyNote },
  { to: "/links", label: "Links", icon: Link2 },
];

export function BottomNav() {
  return (
    <nav className="bottom-nav" aria-label="Primary">
      {TABS.map(({ to, label, icon: Icon, end }) => (
        <NavLink key={to} to={to} end={end} className={({ isActive }) => `tab ${isActive ? "active" : ""}`}>
          <Icon size={18} />
          <span>{label}</span>
        </NavLink>
      ))}
    </nav>
  );
}

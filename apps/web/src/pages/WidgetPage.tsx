import type { Entry } from "@cal/api-client";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { formatTime, todayIso } from "../lib/date";

// Minimal, token-gated read-only agenda for dashboard embeds.
// Loaded outside the auth shell; the API does the permission check.
export function WidgetPage() {
  const [params] = useSearchParams();
  const token = params.get("token") ?? "";
  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [denied, setDenied] = useState(false);

  useEffect(() => {
    if (!token) {
      setDenied(true);
      return;
    }
    fetch(`/api/widget/today?token=${encodeURIComponent(token)}`)
      .then((res) => (res.ok ? res.json() : Promise.reject()))
      .then((data: Entry[]) => {
        data.sort((a, b) => (a.startTime ?? "99") < (b.startTime ?? "99") ? -1 : 1);
        setEntries(data);
      })
      .catch(() => setDenied(true));
  }, [token]);

  const date = new Date().toLocaleDateString("en-US", { weekday: "long", month: "long", day: "numeric" });

  if (denied) return <div className="widget-empty">Invalid or missing widget token.</div>;
  if (!entries) return <div className="widget-empty">Loading…</div>;

  return (
    <div className="widget-root">
      <div className="widget-head">
        <span className="widget-date">{date}</span>
        <span className="widget-brand">Cal</span>
      </div>
      {entries.length === 0 ? (
        <div className="widget-empty">Nothing planned today.</div>
      ) : (
        <ul className="widget-list">
          {entries.map((entry) => (
            <li key={entry.id} className={`widget-row ${entry.completed ? "done" : ""}`}>
              <span className="widget-dot" style={{ background: `var(--c-${entry.color}, var(--accent))` }} />
              <span className="widget-time">{entry.startTime ? formatTime(entry.startTime) : "—"}</span>
              <span className="widget-title">{entry.title}</span>
            </li>
          ))}
        </ul>
      )}
      <div className="widget-today">{todayIso()}</div>
    </div>
  );
}

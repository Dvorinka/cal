// Time — solidtime-style timesheet. Sessions grouped by day, per-day totals,
// delete to correct the record.

import type { TimeEntry } from "@cal/api-client";
import { Timer, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort } from "../lib/date";
import { usePlanner } from "../stores/planner";

function minutesOf(t: TimeEntry): number {
  if (!t.endAt) return 0;
  return Math.round((new Date(t.endAt).getTime() - new Date(t.startAt).getTime()) / 60000);
}

export function TimePage() {
  const api = usePlanner((s) => s.api);
  const [log, setLog] = useState<TimeEntry[]>([]);
  const [sum, setSum] = useState<{ todayMinutes: number; weekMinutes: number }>();

  const load = () => {
    void api.timeLog().then(setLog).catch(() => {});
    void api.timeSummary().then(setSum).catch(() => {});
  };
  useEffect(load, [api]);

  const days = useMemo(() => {
    const m = new Map<string, TimeEntry[]>();
    for (const t of log) {
      const d = t.startAt.slice(0, 10);
      m.set(d, [...(m.get(d) ?? []), t]);
    }
    return [...m.entries()];
  }, [log]);

  return (
    <>
      <PageHeader
        title="Time"
        sub={sum ? `${sum.todayMinutes}m today · ${sum.weekMinutes}m this week` : "sessions"}
      >
        <span className="time-hint"><Timer size={13} /> right-click a task → Start timer</span>
      </PageHeader>
      <div className="page-scroll">
        {days.length === 0 ? (
          <div className="empty-hint">
            <strong>No sessions yet</strong>
            <span>Right-click any task → Start timer. The pill in the sidebar keeps time.</span>
          </div>
        ) : (
          days.map(([day, items]) => {
            const total = items.reduce((n, t) => n + minutesOf(t), 0);
            return (
              <section key={day} className="panel" style={{ marginBottom: 10 }}>
                <h3>
                  {formatDayShort(day)}
                  <span className="kanban-count" style={{ marginLeft: 8 }}>{total}m</span>
                </h3>
                <ul className="trash-list" style={{ padding: 0 }}>
                  {items.map((t) => (
                    <li key={t.id} className="trash-row">
                      <span className="trash-title">{t.title || t.note || "Untitled"}</span>
                      <span className="trash-date">
                        {new Date(t.startAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}
                        {t.endAt ? `–${new Date(t.endAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}` : " · running"}
                        {` · ${minutesOf(t)}m`}
                        {t.planned ? ` /${t.planned}m` : ""}
                      </span>
                      <button
                        type="button"
                        className="icon-btn danger"
                        aria-label="Delete session"
                        onClick={() => void api.deleteTimeEntry(t.id).then(load)}
                      >
                        <Trash2 size={13} />
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            );
          })
        )}
      </div>
    </>
  );
}

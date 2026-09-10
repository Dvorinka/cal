// Time — solidtime-style timesheet. Sessions grouped by day, per-day totals,
// billable $ amounts, project rollups, CSV/JSON export, delete to correct.

import type { TimeEntry } from "@cal/api-client";
import { Download, Timer, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort } from "../lib/date";
import { usePlanner } from "../stores/planner";

function fmtMins(m: number): string {
  if (m < 1) return "<1m";
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  return m % 60 === 0 ? `${h}h` : `${h}h ${m % 60}m`;
}

function minutesOf(t: TimeEntry): number {
  if (!t.endAt) return 0;
  return Math.round((new Date(t.endAt).getTime() - new Date(t.startAt).getTime()) / 60000);
}

function amountOf(t: TimeEntry): number {
  if (!t.billable || !t.rate || !t.endAt) return 0;
  return (minutesOf(t) / 60) * t.rate;
}

export function TimePage() {
  const api = usePlanner((s) => s.api);
  const settings = usePlanner((s) => s.settings);
  const [log, setLog] = useState<TimeEntry[]>([]);
  const [sum, setSum] = useState<{ todayMinutes: number; weekMinutes: number; billableAmount: number }>();
  const [billableOnly, setBillableOnly] = useState(false);

  const load = () => {
    void api.timeLog().then(setLog).catch(() => {});
    void api.timeSummary().then(setSum).catch(() => {});
  };
  useEffect(load, [api]);

  const visible = useMemo(() => (billableOnly ? log.filter((t) => t.billable) : log), [log, billableOnly]);

  const days = useMemo(() => {
    const m = new Map<string, TimeEntry[]>();
    for (const t of visible) {
      const d = t.startAt.slice(0, 10);
      m.set(d, [...(m.get(d) ?? []), t]);
    }
    return [...m.entries()];
  }, [visible]);

  const totalEarned = visible.reduce((n, t) => n + amountOf(t), 0);

  return (
    <>
      <PageHeader
        title="Time"
        sub={
          sum
            ? `${fmtMins(sum.todayMinutes)} today · ${fmtMins(sum.weekMinutes)} this week${sum.billableAmount > 0 ? ` · $${sum.billableAmount.toFixed(2)} billable` : ""}`
            : "sessions"
        }
      >
        <button
          type="button"
          className="btn btn-primary btn-xs"
          onClick={() => void api.startTimer({}).then(load).catch(() => {})}
        >
          <Timer size={12} /> Start timer
        </button>
        <button
          type="button"
          className={`btn btn-secondary btn-xs ${billableOnly ? "on" : ""}`}
          onClick={() => setBillableOnly((v) => !v)}
        >
          Billable only
        </button>
        <a className="btn btn-secondary btn-xs" href="/api/time/export?format=csv" download>
          <Download size={12} /> CSV
        </a>
        <a className="btn btn-secondary btn-xs" href="/api/time/export?format=json" target="_blank" rel="noreferrer">
          <Download size={12} /> JSON
        </a>
      </PageHeader>
      <div className="page-scroll">
        {totalEarned > 0 && (
          <p className="time-total">
            <b>${totalEarned.toFixed(2)}</b> earned in this window
            {settings.defaultRate ? ` · default rate $${settings.defaultRate}/h` : ""}
          </p>
        )}
        {days.length === 0 ? (
          <div className="empty-hint">
            <strong>No sessions yet</strong>
            <span>Right-click any task → Start timer. The pill in the sidebar keeps time.</span>
          </div>
        ) : (
          days.map(([day, items]) => {
            const total = items.reduce((n, t) => n + minutesOf(t), 0);
            const earned = items.reduce((n, t) => n + amountOf(t), 0);
            return (
              <section key={day} className="panel" style={{ marginBottom: 10 }}>
                <h3>
                  {formatDayShort(day)}
                  <span className="kanban-count" style={{ marginLeft: 8 }}>
                    {fmtMins(total)}{earned > 0 ? ` · $${earned.toFixed(2)}` : ""}
                  </span>
                </h3>
                <ul className="trash-list" style={{ padding: 0 }}>
                  {items.map((t) => (
                    <li key={t.id} className="trash-row">
                      <span className="trash-title">
                        {t.title || t.note || "Untitled"}
                        {t.project && <span className="meta-chip" style={{ marginLeft: 6 }}>{t.project}</span>}
                        {t.billable && <span className="meta-chip billable" style={{ marginLeft: 4 }}>$</span>}
                      </span>
                      <span className="trash-date">
                        {new Date(t.startAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}
                        {t.endAt ? `–${new Date(t.endAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}` : " · running"}
                        {` · ${fmtMins(minutesOf(t))}`}
                        {amountOf(t) > 0 ? ` · $${amountOf(t).toFixed(2)}` : ""}
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

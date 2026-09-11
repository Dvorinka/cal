// Time — solidtime-style timesheet. Sessions grouped by day, per-day
// totals, billable amounts, project rollups, tags, CSV/JSON export.
// Manual entries and editing mean no delete-and-restart corrections.

import type { TimeEntry, TimeEntryInput, TimeEntryPatch } from "@cal/api-client";
import { Download, Pencil, Plus, Tag, Timer, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort, todayIso } from "../lib/date";
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

// datetime-local helpers — value is local wall time.
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function TimePage() {
  const api = usePlanner((s) => s.api);
  const settings = usePlanner((s) => s.settings);
  const entries = usePlanner((s) => s.entries);
  const [log, setLog] = useState<TimeEntry[]>([]);
  const [sum, setSum] = useState<{ todayMinutes: number; weekMinutes: number; billableAmount: number }>();
  const [billableOnly, setBillableOnly] = useState(false);
  const [tagFilter, setTagFilter] = useState("");
  const [editing, setEditing] = useState<TimeEntry | null>(null);
  const [adding, setAdding] = useState(false);

  const load = () => {
    void api.timeLog().then(setLog).catch(() => {});
    void api.timeSummary().then(setSum).catch(() => {});
  };
  useEffect(load, [api]);

  const allTags = useMemo(() => {
    const set = new Set<string>();
    for (const t of log) for (const tag of t.tags ?? []) set.add(tag);
    return [...set].sort();
  }, [log]);

  const visible = useMemo(
    () =>
      log.filter((t) => {
        if (billableOnly && !t.billable) return false;
        if (tagFilter && !(t.tags ?? []).includes(tagFilter)) return false;
        return true;
      }),
    [log, billableOnly, tagFilter],
  );

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
        <button type="button" className="btn btn-secondary btn-xs" onClick={() => setAdding(true)}>
          <Plus size={12} /> Log time
        </button>
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
        {allTags.length > 0 && (
          <div className="tag-filter-row">
            <Tag size={12} style={{ color: "var(--text-3)" }} />
            <button
              type="button"
              className={`meta-chip ${tagFilter === "" ? "on" : ""}`}
              onClick={() => setTagFilter("")}
            >
              all
            </button>
            {allTags.map((tag) => (
              <button
                key={tag}
                type="button"
                className={`meta-chip ${tagFilter === tag ? "on" : ""}`}
                onClick={() => setTagFilter(tagFilter === tag ? "" : tag)}
              >
                {tag}
              </button>
            ))}
          </div>
        )}
        {totalEarned > 0 && (
          <p className="time-total">
            <b>${totalEarned.toFixed(2)}</b> earned in this window
            {settings.defaultRate ? ` · default rate $${settings.defaultRate}/h` : ""}
          </p>
        )}
        {days.length === 0 ? (
          <div className="empty-hint">
            <strong>No sessions yet</strong>
            <span>Start the timer or log time manually. Right-click any task for its own timer.</span>
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
                        {(t.tags ?? []).map((tag) => (
                          <span key={tag} className="meta-chip" style={{ marginLeft: 4 }}>{tag}</span>
                        ))}
                      </span>
                      <span className="trash-date">
                        {new Date(t.startAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}
                        {t.endAt ? `–${new Date(t.endAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" })}` : " · running"}
                        {` · ${fmtMins(minutesOf(t))}`}
                        {amountOf(t) > 0 ? ` · $${amountOf(t).toFixed(2)}` : ""}
                      </span>
                      <button
                        type="button"
                        className="icon-btn"
                        aria-label="Edit session"
                        onClick={() => setEditing(t)}
                      >
                        <Pencil size={13} />
                      </button>
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

      {adding && (
        <TimeDialog
          title="Log time"
          initial={{ startAt: `${todayIso()}T09:00`, endAt: `${todayIso()}T10:00` }}
          tasks={entries.filter((e) => e.type === "task" && !e.completed)}
          onClose={() => setAdding(false)}
          onSave={async (f) => {
            await api.createTimeEntry(f).then(load).catch(() => {});
            setAdding(false);
          }}
        />
      )}
      {editing && (
        <TimeDialog
          title="Edit session"
          entry={editing}
          tasks={entries.filter((e) => e.type === "task")}
          onClose={() => setEditing(null)}
          onSave={async (f) => {
            const patch: TimeEntryPatch = {
              startAt: f.startAt,
              endAt: f.endAt || null,
              note: f.note,
              tags: f.tags,
              billable: f.billable,
              rate: f.rate ?? null,
              projectId: f.projectId || null,
              entryId: f.entryId || null,
            };
            await api.updateTimeEntry(editing.id, patch).then(load).catch(() => {});
            setEditing(null);
          }}
        />
      )}
    </>
  );
}

// TimeDialog — shared create/edit form. Local wall-clock times.
function TimeDialog({
  title,
  entry,
  initial,
  tasks,
  onClose,
  onSave,
}: {
  title: string;
  entry?: TimeEntry;
  initial?: { startAt: string; endAt: string };
  tasks: { id: string; title: string }[];
  onClose: () => void;
  onSave: (form: TimeEntryInput) => Promise<void>;
}) {
  const [f, setF] = useState({
    entryId: entry?.entryId ?? "",
    note: entry?.note ?? "",
    startAt: entry ? toLocalInput(entry.startAt) : initial?.startAt ?? "",
    endAt: entry ? toLocalInput(entry.endAt) : initial?.endAt ?? "",
    billable: entry?.billable ?? false,
    rate: entry?.rate ? String(entry.rate) : "",
    projectId: entry?.projectId ?? "",
    tags: (entry?.tags ?? []).join(", "),
  });
  const [busy, setBusy] = useState(false);
  const set = (k: keyof typeof f) => (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
    setF({ ...f, [k]: e.target.type === "checkbox" ? (e.target as HTMLInputElement).checked : e.target.value });

  async function save() {
    if (!f.startAt) return;
    setBusy(true);
    const toIso = (v: string) => (v ? new Date(v).toISOString() : undefined);
    await onSave({
      entryId: f.entryId || undefined,
      note: f.note || undefined,
      startAt: toIso(f.startAt),
      endAt: toIso(f.endAt),
      billable: f.billable,
      rate: f.rate ? Number(f.rate) : undefined,
      projectId: f.projectId || undefined,
      tags: f.tags.split(",").map((t) => t.trim()).filter(Boolean),
    });
    setBusy(false);
  }

  return (
    <div className="scrim" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="palette mail-dialog" role="dialog" aria-label={title}>
        <h3>{title}</h3>
        <label className="field">
          <span>Task</span>
          <select value={f.entryId} onChange={set("entryId")}>
            <option value="">— no linked task —</option>
            {tasks.map((t) => (
              <option key={t.id} value={t.id}>{t.title}</option>
            ))}
          </select>
        </label>
        <label className="field"><span>Note</span><input value={f.note} onChange={set("note")} placeholder="What was this?" /></label>
        <div className="field-row">
          <label className="field"><span>Start</span><input type="datetime-local" value={f.startAt} onChange={set("startAt")} required /></label>
          <label className="field"><span>End</span><input type="datetime-local" value={f.endAt} onChange={set("endAt")} /></label>
        </div>
        <div className="field-row">
          <label className="field">
            <span>Tags (comma)</span>
            <input value={f.tags} onChange={set("tags")} placeholder="client, deep-work" />
          </label>
          <label className="field">
            <span>Rate $/h</span>
            <input type="number" min="0" step="0.01" value={f.rate} onChange={set("rate")} />
          </label>
        </div>
        <label className="field field-check">
          <input type="checkbox" checked={f.billable} onChange={set("billable")} />
          <span>Billable</span>
        </label>
        <div className="mail-dialog-actions">
          <button type="button" className="btn btn-secondary" onClick={onClose}>Cancel</button>
          <button type="button" className="btn btn-primary" disabled={busy || !f.startAt} onClick={() => void save()}>
            {busy ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}

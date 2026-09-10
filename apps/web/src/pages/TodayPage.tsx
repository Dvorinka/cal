import { Check, Link2, Plus, StickyNote } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { addDays, formatTime, iso, timeToMinutes, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

const addDaysIso = (date: string, n: number) => iso(addDays(new Date(`${date}T12:00:00`), n));

function greeting(now: Date): string {
  const h = now.getHours();
  if (h < 5) return "Still up";
  if (h < 12) return "Good morning";
  if (h < 18) return "Good afternoon";
  return "Good evening";
}

export function TodayPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const selectDate = useUi((state) => state.selectDate);
  const [now, setNow] = useState(() => new Date());
  const today = todayIso();

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => selectDate(today), [today, selectDate]);

  // Deep-link landing: ensure entries exist even if the calendar never mounted.
  useEffect(() => {
    void loadEntries({ from: addDaysIso(today, -14), to: addDaysIso(today, 14) });
  }, [today, loadEntries]);

  const dayEntries = useMemo(
    () => entries.filter((e) => e.date === today).sort((a, b) => (timeToMinutes(a.startTime) ?? 9999) - (timeToMinutes(b.startTime) ?? 9999)),
    [entries, today],
  );
  const timed = dayEntries.filter((e) => timeToMinutes(e.startTime) !== undefined);
  const untimedTasks = dayEntries.filter((e) => e.type === "task" && timeToMinutes(e.startTime) === undefined);
  const notes = dayEntries.filter((e) => e.type === "note");
  const links = dayEntries.filter((e) => e.type === "link");
  const tasksTotal = dayEntries.filter((e) => e.type === "task").length;
  const tasksDone = dayEntries.filter((e) => e.type === "task" && e.completed).length;
  const nowMinutes = now.getHours() * 60 + now.getMinutes();
  const nextUp = timed.find((e) => (timeToMinutes(e.startTime) ?? 0) >= nowMinutes || (timeToMinutes(e.endTime) ?? 0) > nowMinutes);

  return (
    <>
      <PageHeader
        title={now.toLocaleDateString(undefined, { weekday: "long" })}
        sub={now.toLocaleDateString(undefined, { month: "long", day: "numeric", year: "numeric" })}
      >
        <button type="button" className="btn btn-primary" onClick={() => openCreate(today)}>
          <Plus size={14} strokeWidth={2.5} /> Add
        </button>
      </PageHeader>

      <div className="page-scroll">
        <div className="today-hero">
          <p className="greeting">{greeting(now)}.</p>
          <p className="today-line">
            {tasksTotal === 0 && timed.length === 0
              ? "Nothing planned. A quiet page is also a plan."
              : `${tasksDone} of ${tasksTotal} task${tasksTotal === 1 ? "" : "s"} done`}
            {nextUp ? ` — next up at ${formatTime(nextUp.startTime!)}` : ""}
          </p>
          {tasksTotal > 0 && (
            <div className="meter" aria-hidden>
              <i style={{ width: `${(tasksDone / tasksTotal) * 100}%` }} />
            </div>
          )}
        </div>

        <div className="today-cols">
          <section className="panel">
            <h3>Schedule</h3>
            {timed.length === 0 && <p className="panel-empty">No timed entries today.</p>}
            <ol className="agenda">
              {timed.map((e) => {
                const start = timeToMinutes(e.startTime)!;
                const end = timeToMinutes(e.endTime) ?? start + 60;
                const live = start <= nowMinutes && nowMinutes < end;
                const past = end <= nowMinutes;
                return (
                  <li
                    key={e.id}
                    className={`agenda-item color-${e.color} ${live ? "live" : ""} ${past ? "past" : ""} ${e.completed ? "done" : ""}`}
                    onClick={() => openEdit(e)}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(ev) => ev.key === "Enter" && openEdit(e)}
                  >
                    <span className="agenda-time">
                      {formatTime(e.startTime!)}
                      <i>{formatTime(e.endTime ?? "")}</i>
                    </span>
                    <span className="agenda-title">{e.title}</span>
                    {live && <span className="live-pill">Now</span>}
                  </li>
                );
              })}
            </ol>
          </section>

          <section className="panel">
            <h3>Tasks</h3>
            {untimedTasks.length === 0 && <p className="panel-empty">No untimed tasks today.</p>}
            <ul className="check-list">
              {untimedTasks.map((e) => (
                <li key={e.id} className={e.completed ? "done" : ""}>
                  <button
                    type="button"
                    className="tickbox"
                    aria-label={e.completed ? "Reopen" : "Complete"}
                    onClick={() => void updateEntry(e.id, { completed: !e.completed })}
                  >
                    {e.completed && <Check size={12} strokeWidth={3} />}
                  </button>
                  <button type="button" className="row-title" onClick={() => openEdit(e)}>
                    {e.title}
                  </button>
                  {e.recur !== "none" && <span className="meta-chip">{e.recur}</span>}
                </li>
              ))}
            </ul>

            {(notes.length > 0 || links.length > 0) && <h3 style={{ marginTop: 18 }}>Notes & links</h3>}
            <ul className="check-list">
              {notes.map((e) => (
                <li key={e.id}>
                  <StickyNote size={13} className="row-ic" />
                  <button type="button" className="row-title" onClick={() => openEdit(e)}>
                    {e.title}
                  </button>
                </li>
              ))}
              {links.map((e) => (
                <li key={e.id}>
                  <Link2 size={13} className="row-ic" />
                  <button type="button" className="row-title" onClick={() => openEdit(e)}>
                    {e.title}
                  </button>
                </li>
              ))}
            </ul>
          </section>
        </div>
      </div>
    </>
  );
}

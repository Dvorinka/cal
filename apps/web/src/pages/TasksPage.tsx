import { Check, Plus } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { allTags, groupTasks } from "../lib/entries";
import { parseQuickAdd } from "../lib/quickadd";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";
import type { Entry } from "@cal/api-client";

function TaskRow({ entry, muted }: { entry: Entry; muted?: boolean }) {
  const openEdit = useUi((state) => state.openEdit);
  const updateEntry = usePlanner((state) => state.updateEntry);
  return (
    <li className={`task-row ${entry.completed ? "done" : ""} ${muted ? "muted" : ""}`}>
      <button
        type="button"
        className="tickbox"
        aria-label={entry.completed ? "Reopen" : "Complete"}
        onClick={() => void updateEntry(entry.id, { completed: !entry.completed })}
      >
        {entry.completed && <Check size={12} strokeWidth={3} />}
      </button>
      <button type="button" className="row-title" onClick={() => openEdit(entry)}>
        {entry.title}
      </button>
      {entry.tags.map((tag) => (
        <span key={tag} className="meta-chip">
          {tag}
        </span>
      ))}
      {entry.recur !== "none" && <span className="meta-chip">{entry.recur}</span>}
      <span className="meta-chip date">{formatDayShort(entry.date)}</span>
    </li>
  );
}

export function TasksPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const createEntry = usePlanner((state) => state.createEntry);
  const [tag, setTag] = useState<string>();
  const [quick, setQuick] = useState("");
  const [showDone, setShowDone] = useState(false);
  const today = todayIso();

  // The task list needs every entry, not just the visible calendar range.
  useEffect(() => {
    void loadEntries({});
  }, [loadEntries]);

  const tasks = useMemo(
    () => (tag ? entries.filter((e) => e.tags.includes(tag)) : entries),
    [entries, tag],
  );
  const groups = useMemo(() => groupTasks(tasks, today), [tasks, today]);
  const tags = useMemo(() => allTags(entries.filter((e) => e.type === "task")), [entries]);

  async function addQuick() {
    const parsed = parseQuickAdd(quick);
    if (!parsed) return;
    setQuick("");
    await createEntry({
      title: parsed.title,
      type: "task",
      date: parsed.date,
      startTime: parsed.startTime,
      tags: [...parsed.tags, ...(tag ? [tag] : [])],
    });
  }

  return (
    <>
      <PageHeader title="Tasks" sub={`${groups.overdue.length + groups.today.length + groups.upcoming.length} open`} />
      <div className="page-scroll">
        <div className="quick-add">
          <Plus size={15} />
          <input
            value={quick}
            onChange={(e) => setQuick(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && void addQuick()}
            placeholder='Try "dentist fri 5pm #health" — Enter to add'
            aria-label="Quick add task"
          />
        </div>

        {tags.length > 0 && (
          <div className="tag-row">
            <button type="button" className={`tag-chip ${!tag ? "active" : ""}`} onClick={() => setTag(undefined)}>
              All
            </button>
            {tags.map((t) => (
              <button
                key={t}
                type="button"
                className={`tag-chip ${tag === t ? "active" : ""}`}
                onClick={() => setTag(tag === t ? undefined : t)}
              >
                {t}
              </button>
            ))}
          </div>
        )}

        <div className="task-groups">
          {groups.overdue.length > 0 && (
            <section className="panel">
              <h3 className="overdue">Overdue</h3>
              <ul className="task-list">{groups.overdue.map((e) => <TaskRow key={e.id} entry={e} />)}</ul>
            </section>
          )}
          <section className="panel">
            <h3>Today</h3>
            {groups.today.length === 0 && <p className="panel-empty">Nothing due today.</p>}
            <ul className="task-list">{groups.today.map((e) => <TaskRow key={e.id} entry={e} />)}</ul>
          </section>
          {groups.upcoming.length > 0 && (
            <section className="panel">
              <h3>Upcoming</h3>
              <ul className="task-list">{groups.upcoming.map((e) => <TaskRow key={e.id} entry={e} />)}</ul>
            </section>
          )}
          {groups.done.length > 0 && (
            <section className="panel">
              <button type="button" className="group-toggle" onClick={() => setShowDone(!showDone)}>
                Done · {groups.done.length}
              </button>
              {showDone && <ul className="task-list">{groups.done.map((e) => <TaskRow key={e.id} entry={e} muted />)}</ul>}
            </section>
          )}
        </div>
      </div>
    </>
  );
}

import { Flame, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import type { Habit } from "@cal/api-client";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort, todayIso } from "../lib/date";
import { openHabitOccurrence } from "../lib/habits";
import { reportErr, usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function HabitsPage() {
  const api = usePlanner((state) => state.api);
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const [habits, setHabits] = useState<Habit[]>([]);
  const [loading, setLoading] = useState(true);
  const today = todayIso();

  useEffect(() => {
    void loadEntries({});
  }, [api, loadEntries]);

  useEffect(() => {
    let gone = false;
    setLoading(true);
    void api
      .habits()
      .then((rows) => {
        if (!gone) setHabits(rows);
      })
      .catch(reportErr("Could not load habits"))
      .finally(() => {
        if (!gone) setLoading(false);
      });
    return () => {
      gone = true;
    };
  }, [api, entries]);

  async function markDone(id: string) {
    await updateEntry(id, { completed: true });
    void loadEntries({});
  }

  return (
    <>
      <PageHeader title="Habits" sub="Repeating tasks tagged habit. A missed day breaks the streak.">
        <button
          type="button"
          className="btn btn-primary"
          onClick={() => openCreate(today, undefined, { type: "task", tags: "habit", recur: "daily" })}
        >
          <Plus size={14} strokeWidth={2.5} /> New habit
        </button>
      </PageHeader>
      <div className="page-scroll" data-testid="habits-page">
        {loading && habits.length === 0 && <p className="panel-empty">Loading habits…</p>}
        {!loading && habits.length === 0 && (
          <div className="empty-hint">
            <Flame size={22} />
            <strong>No habits yet</strong>
            <span>A habit is a repeating task with the tag habit. Check it off each period to keep the streak.</span>
          </div>
        )}
        {habits.length > 0 && (
          <ul className="habit-list">
            {habits.map((habit) => {
              const due = openHabitOccurrence(entries, habit, today);
              const source = entries.find((entry) => entry.id === habit.id);
              return (
                <li key={habit.id} className="habit-card">
                  <div className="habit-streak" title="Consecutive periods completed">
                    <b>{habit.streak}</b>
                    <span>streak</span>
                  </div>
                  <div className="habit-copy">
                    <button type="button" className="row-title" disabled={!source} onClick={() => source && openEdit(source)}>
                      {habit.title}
                    </button>
                    <p>
                      <span className="meta-chip">{habit.recur}</span>
                      {habit.lastDone ? <span>Last done {formatDayShort(habit.lastDone)}</span> : <span>Not done yet</span>}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="btn btn-secondary btn-xs"
                    disabled={!due}
                    onClick={() => due && void markDone(due.id)}
                  >
                    {due ? "Done" : "Up to date"}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </>
  );
}

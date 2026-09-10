import { Plus } from "lucide-react";
import { useEffect, useMemo } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function NotesPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);

  useEffect(() => {
    void loadEntries({});
  }, [loadEntries]);

  const notes = useMemo(
    () =>
      entries
        .filter((e) => e.type === "note")
        .sort((a, b) => b.date.localeCompare(a.date) || b.createdAt.localeCompare(a.createdAt)),
    [entries],
  );

  return (
    <>
      <PageHeader title="Notes" sub={`${notes.length} note${notes.length === 1 ? "" : "s"}`}>
        <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
          <Plus size={14} strokeWidth={2.5} /> New note
        </button>
      </PageHeader>
      <div className="page-scroll">
        {notes.length === 0 ? (
          <div className="empty-hint">
            <strong>No notes yet</strong>
            <span>Notes live on the calendar too — they pin context to a day.</span>
            <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
              <Plus size={14} /> Write one
            </button>
          </div>
        ) : (
          <div className="note-grid">
            {notes.map((note) => (
              <button
                key={note.id}
                type="button"
                className={`note-card color-${note.color}`}
                onClick={() => openEdit(note)}
              >
                <strong>{note.title}</strong>
                {note.content && <p>{note.content.slice(0, 160)}</p>}
                <span className="note-date">{formatDayShort(note.date)}</span>
              </button>
            ))}
          </div>
        )}
      </div>
    </>
  );
}

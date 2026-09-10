// Tags — every tag across entries with counts; click filters a cross-type list.

import { Hash } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function TagsPage() {
  const api = usePlanner((s) => s.api);
  const entries = usePlanner((s) => s.entries);
  const loadEntries = usePlanner((s) => s.loadEntries);
  const openEdit = useUi((s) => s.openEdit);
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [sel, setSel] = useState("");

  useEffect(() => {
    void loadEntries({});
    void api.tags().then(setCounts).catch(() => {});
  }, [api, loadEntries]);

  const filtered = useMemo(
    () => entries.filter((e) => e.tags.includes(sel)).sort((a, b) => b.date.localeCompare(a.date)),
    [entries, sel],
  );

  return (
    <>
      <PageHeader title="Tags" sub={sel ? `#${sel} — ${filtered.length} entries` : `${Object.keys(counts).length} tags in use`} />
      <div className="tag-filter-row" style={{ paddingTop: 8 }}>
        <button type="button" className={`tag-chip ${sel === "" ? "on" : ""}`} onClick={() => setSel("")}>
          All
        </button>
        {Object.entries(counts).map(([t, n]) => (
          <button key={t} type="button" className={`tag-chip ${sel === t ? "on" : ""}`} onClick={() => setSel(sel === t ? "" : t)}>
            #{t} <span className="tag-n">{n}</span>
          </button>
        ))}
      </div>
      <div className="page-scroll">
        {sel === "" ? (
          <div className="empty-hint">
            <strong>Pick a tag</strong>
            <span>Tags span tasks, notes, links, and board cards — one tap filters everything.</span>
          </div>
        ) : filtered.length === 0 ? (
          <div className="empty-hint">
            <strong>No entries tagged #{sel}</strong>
          </div>
        ) : (
          <ul className="trash-list">
            {filtered.map((e) => (
              <li key={e.id} className="trash-row">
                <span className="trash-type">{e.type}</span>
                <button type="button" className="trash-title tag-open" onClick={() => openEdit(e)}>
                  <Hash size={12} /> {e.title || "Untitled"}
                </button>
                <span className="trash-date">{formatDayShort(e.date)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}

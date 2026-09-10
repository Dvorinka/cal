// Trash — soft-deleted entries. Restore returns them in place; purge is final.

import type { Entry } from "@cal/api-client";
import { RotateCcw, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort } from "../lib/date";
import { usePlanner } from "../stores/planner";

const TYPE_LABEL: Record<string, string> = { task: "Task", event: "Event", note: "Note", link: "Link" };

export function TrashPage() {
  const api = usePlanner((s) => s.api);
  const loadEntries = usePlanner((s) => s.loadEntries);
  const toast = usePlanner((s) => s.toast);
  const [items, setItems] = useState<Entry[]>([]);

  const load = () => void api.trash().then(setItems).catch(() => {});
  useEffect(load, [api]);

  return (
    <>
      <PageHeader title="Trash" sub={`${items.length} deleted item${items.length === 1 ? "" : "s"}`} />
      <div className="page-scroll">
        {items.length === 0 ? (
          <div className="empty-hint">
            <strong>Nothing in trash</strong>
            <span>Deleted entries land here — restore or purge them for good.</span>
          </div>
        ) : (
          <ul className="trash-list">
            {items.map((e) => (
              <li key={e.id} className="trash-row">
                <span className="trash-type">{TYPE_LABEL[e.type] ?? e.type}</span>
                <span className="trash-title">{e.title || "Untitled"}</span>
                <span className="trash-date">{formatDayShort(e.date)}</span>
                <button
                  type="button"
                  className="icon-btn"
                  aria-label={`Restore ${e.title}`}
                  onClick={() =>
                    void api.restoreEntry(e.id).then(() => {
                      setItems((xs) => xs.filter((x) => x.id !== e.id));
                      void loadEntries({});
                      toast("Restored");
                    })
                  }
                >
                  <RotateCcw size={14} />
                </button>
                <button
                  type="button"
                  className="icon-btn danger"
                  aria-label={`Permanently delete ${e.title}`}
                  onClick={() => {
                    if (!window.confirm(`Delete "${e.title}" permanently? This can't be undone.`)) return;
                    void api.purgeEntry(e.id).then(() => setItems((xs) => xs.filter((x) => x.id !== e.id)));
                  }}
                >
                  <Trash2 size={14} />
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}

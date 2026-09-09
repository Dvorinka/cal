import type { Entry } from "@cal/api-client";
import { Check, Link2, StickyNote, Trash2 } from "lucide-react";

interface Props {
  entry: Entry;
  onToggle: (id: string, completed: boolean) => void;
  onDelete: (id: string) => void;
  onDragStart: (entry: Entry) => void;
}

export function EntryCard({ entry, onToggle, onDelete, onDragStart }: Props) {
  const Icon = entry.type === "link" ? Link2 : entry.type === "note" ? StickyNote : Check;
  return (
    <article className={`entry color-${entry.color}`} draggable onDragStart={() => onDragStart(entry)}>
      <button className="check" type="button" onClick={() => onToggle(entry.id, !entry.completed)} aria-label="Toggle complete">
        {entry.completed && <Check size={12} />}
      </button>
      <div className="entry-copy">
        <p className={entry.completed ? "done" : ""}>{entry.title}</p>
        <span>
          <Icon size={12} /> {entry.type}
        </span>
      </div>
      <button className="icon-button" type="button" onClick={() => onDelete(entry.id)} aria-label="Delete entry">
        <Trash2 size={14} />
      </button>
    </article>
  );
}

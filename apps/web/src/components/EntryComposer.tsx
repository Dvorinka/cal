import type { EntryInput, EntryType } from "@cal/api-client";
import { Plus } from "lucide-react";
import { useState } from "react";
import { todayIso } from "../lib/date";

interface Props {
  date: string;
  onCreate: (input: EntryInput) => Promise<void>;
}

export function EntryComposer({ date, onCreate }: Props) {
  const [title, setTitle] = useState("");
  const [type, setType] = useState<EntryType>("task");
  const [color, setColor] = useState("slate");

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const trimmed = title.trim();
    if (!trimmed) return;
    await onCreate({ title: trimmed, type, date: date || todayIso(), color, tags: [] });
    setTitle("");
  }

  return (
    <form className="composer" onSubmit={submit}>
      <input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="Quick add" aria-label="Entry title" />
      <select value={type} onChange={(event) => setType(event.target.value as EntryType)} aria-label="Entry type">
        <option value="task">Task</option>
        <option value="note">Note</option>
        <option value="link">Link</option>
      </select>
      <select value={color} onChange={(event) => setColor(event.target.value)} aria-label="Entry color">
        <option value="slate">Slate</option>
        <option value="mint">Mint</option>
        <option value="rose">Rose</option>
        <option value="amber">Amber</option>
      </select>
      <button type="submit" aria-label="Create entry">
        <Plus size={16} />
      </button>
    </form>
  );
}

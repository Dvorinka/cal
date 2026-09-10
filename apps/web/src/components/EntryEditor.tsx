import type { EntryType, Recur } from "@cal/api-client";
import { AnimatePresence, motion } from "framer-motion";
import { CalendarClock, Check, Link2, StickyNote, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

const COLORS = ["slate", "mint", "sky", "violet", "amber", "orange", "rose", "red"] as const;

const TYPE_META: { value: EntryType; label: string; icon: typeof Check }[] = [
  { value: "task", label: "Task", icon: Check },
  { value: "event", label: "Event", icon: CalendarClock },
  { value: "note", label: "Note", icon: StickyNote },
  { value: "link", label: "Link", icon: Link2 },
];

const REMINDS: { value: number | ""; label: string }[] = [
  { value: "", label: "No reminder" },
  { value: 0, label: "At start" },
  { value: 5, label: "5 min before" },
  { value: 15, label: "15 min before" },
  { value: 30, label: "30 min before" },
  { value: 60, label: "1 hour before" },
  { value: 1440, label: "1 day before" },
];

const RECURS: { value: Recur; label: string }[] = [
  { value: "none", label: "Does not repeat" },
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
  { value: "monthly", label: "Monthly" },
  { value: "yearly", label: "Yearly" },
];

export function EntryEditor() {
  const editor = useUi((state) => state.editor);
  const close = useUi((state) => state.closeEditor);
  const createEntry = usePlanner((state) => state.createEntry);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const deleteEntry = usePlanner((state) => state.deleteEntry);
  const toast = usePlanner((state) => state.toast);

  const entries = usePlanner((state) => state.entries);
  const accounts = usePlanner((state) => state.accounts);
  const loadAccounts = usePlanner((state) => state.loadAccounts);
  const open = editor.mode !== "closed";
  // Resolve against the live store so optimistic updates (e.g. Done) re-render.
  const editing =
    editor.mode === "edit" ? (entries.find((e) => e.id === editor.entry.id) ?? editor.entry) : undefined;

  const [title, setTitle] = useState("");
  const [type, setType] = useState<EntryType>("task");
  const [date, setDate] = useState("");
  const [startTime, setStartTime] = useState("");
  const [endTime, setEndTime] = useState("");
  const [recur, setRecur] = useState<Recur>("none");
  const [remind, setRemind] = useState<number | "">("");
  const [accountId, setAccountId] = useState("");
  const [linkUrl, setLinkUrl] = useState("");
  const [color, setColor] = useState("slate");
  const [tags, setTags] = useState("");
  const [content, setContent] = useState("");
  const [saving, setSaving] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (editor.mode === "create") {
      setTitle("");
      setType("task");
      setDate(editor.date);
      setStartTime(editor.startTime ?? "");
      setEndTime(editor.startTime ? addHour(editor.startTime) : "");
      setRecur("none");
      setRemind("");
      setAccountId("");
      setLinkUrl("");
      setColor("slate");
      setTags("");
      setContent("");
    } else if (editor.mode === "edit") {
      const e = editor.entry;
      setTitle(e.title);
      setType(e.type);
      setDate(e.date);
      setStartTime(e.startTime ?? "");
      setEndTime(e.endTime ?? "");
      setRecur(e.recur);
      setRemind(e.remind ?? "");
      setAccountId(e.accountId ?? "");
      setLinkUrl(e.linkUrl ?? "");
      setColor(e.color);
      setTags(e.tags.join(", "));
      setContent(e.content ?? "");
    }
  }, [editor]);

  useEffect(() => {
    if (open) {
      window.setTimeout(() => titleRef.current?.focus(), 30);
      if (accounts.length === 0) void loadAccounts();
    }
  }, [open, accounts.length, loadAccounts]);

  useEffect(() => {
    if (!open) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") close();
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        event.preventDefault();
        void save();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  async function save() {
    const trimmed = title.trim();
    if (!trimmed || !date) {
      toast(!trimmed ? "A title is required" : "A date is required");
      return;
    }
    if (endTime && !startTime) {
      toast("Set a start time or clear the end time");
      return;
    }
    if (startTime && endTime && endTime <= startTime) {
      toast("End time must be after the start time");
      return;
    }
    setSaving(true);
    const fields = {
      title: trimmed,
      type,
      date,
      startTime,
      endTime,
      color,
      tags: tags.split(",").map((t) => t.trim()).filter(Boolean),
      content: content.trim(),
      linkUrl: type === "link" ? linkUrl.trim() : "",
      recur: type === "task" ? recur : ("none" as Recur),
      remind: startTime && remind !== "" ? remind : null,
    };
    try {
      if (editing) await updateEntry(editing.id, fields);
      else await createEntry({ ...fields, type, accountId: accountId || undefined });
      close();
    } finally {
      setSaving(false);
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="scrim"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.15 }}
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) close();
          }}
        >
          <motion.div
            className="editor"
            role="dialog"
            aria-label={editing ? "Edit entry" : "New entry"}
            initial={{ opacity: 0, scale: 0.96, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.97, y: 6 }}
            transition={{ type: "spring", duration: 0.35, bounce: 0 }}
          >
            <input
              ref={titleRef}
              className="editor-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Title"
              aria-label="Title"
            />
            <div className="seg" role="group" aria-label="Entry type">
              {TYPE_META.map(({ value, label, icon: Icon }) => (
                <button key={value} type="button" className={type === value ? "active" : ""} onClick={() => setType(value)}>
                  <Icon size={13} /> {label}
                </button>
              ))}
            </div>
            <div className="editor-row">
              <label className="field">
                <span>Date</span>
                <input className="input" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
              </label>
              <label className="field">
                <span>Repeats</span>
                <select
                  className="select"
                  value={type === "task" ? recur : "none"}
                  disabled={type !== "task"}
                  onChange={(e) => setRecur(e.target.value as Recur)}
                >
                  {RECURS.map((r) => (
                    <option key={r.value} value={r.value}>
                      {r.label}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <div className="editor-row">
              <label className="field">
                <span>Start</span>
                <input className="input" type="time" value={startTime} onChange={(e) => setStartTime(e.target.value)} />
              </label>
              <label className="field">
                <span>End</span>
                <input className="input" type="time" value={endTime} onChange={(e) => setEndTime(e.target.value)} />
              </label>
              {startTime && (
                <label className="field">
                  <span>Remind</span>
                  <select
                    className="select"
                    value={remind}
                    onChange={(e) => setRemind(e.target.value === "" ? "" : Number(e.target.value))}
                  >
                    {REMINDS.map((r) => (
                      <option key={r.label} value={r.value}>
                        {r.label}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
            {accounts.length > 0 && (
              <div className="editor-row">
                <label className="field">
                  <span>Calendar</span>
                  {editing?.accountId ? (
                    <select className="select" disabled value={editing.accountId}>
                      {accounts.map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.name} (synced)
                        </option>
                      ))}
                    </select>
                  ) : (
                    <select className="select" value={accountId} onChange={(e) => setAccountId(e.target.value)}>
                      <option value="">Cal (local)</option>
                      {accounts.map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.name}
                        </option>
                      ))}
                    </select>
                  )}
                </label>
              </div>
            )}
            {type === "link" && (
              <label className="field">
                <span>URL</span>
                <input
                  className="input"
                  type="url"
                  value={linkUrl}
                  onChange={(e) => setLinkUrl(e.target.value)}
                  placeholder="https://…"
                />
              </label>
            )}
            <div className="field">
              <span>Color</span>
              <div className="dots">
                {COLORS.map((c) => (
                  <button
                    key={c}
                    type="button"
                    className={`dot ${color === c ? "active" : ""}`}
                    style={{ "--dot": `var(--c-${c})` } as React.CSSProperties}
                    onClick={() => setColor(c)}
                    aria-label={`Color ${c}`}
                  />
                ))}
              </div>
            </div>
            <label className="field">
              <span>Tags</span>
              <input
                className="input"
                value={tags}
                onChange={(e) => setTags(e.target.value)}
                placeholder="work, personal"
              />
            </label>
            <label className="field">
              <span>Details</span>
              <textarea
                className="textarea"
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Notes, context, links…"
              />
            </label>
            <div className="editor-foot">
              {editing?.type === "task" && (
                <label className="switch-row" style={{ fontSize: 12.5 }}>
                  Done
                  <span className="switch">
                    <input
                      type="checkbox"
                      checked={editing.completed}
                      onChange={(e) => void updateEntry(editing.id, { completed: e.target.checked })}
                    />
                    <i />
                  </span>
                </label>
              )}
              {editing && (
                <button
                  type="button"
                  className="btn btn-danger"
                  onClick={() => {
                    void deleteEntry(editing.id);
                    close();
                  }}
                >
                  <Trash2 size={14} /> Delete
                </button>
              )}
              <span className="spacer" />
              <span className="hint">⌘↵ to save</span>
              <button type="button" className="btn btn-ghost" onClick={close}>
                Cancel
              </button>
              <button type="button" className="btn btn-primary" disabled={saving} onClick={() => void save()}>
                {editing ? "Save" : "Create"}
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

function addHour(time: string): string {
  const [h, m] = time.split(":").map(Number);
  return `${String(Math.min(23, h + 1)).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

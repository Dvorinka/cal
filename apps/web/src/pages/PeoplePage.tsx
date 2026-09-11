// People — a warm, private spot for the people who matter: birthdays,
// anniversaries, namedays, notes. Personal by design, not a sales CRM.

import type { Person, PersonDate, PersonInput } from "@cal/api-client";
import { Cake, Heart, Pencil, Plus, Search, Trash2, X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort } from "../lib/date";
import { daysLabel, nextByPerson, upcomingDates, type PersonOccurrence } from "../lib/people";
import { usePlanner } from "../stores/planner";

const RELATIONS = ["family", "partner", "friend", "colleague", "acquaintance"];
const COLORS = ["slate", "mint", "sky", "violet", "amber", "orange", "rose", "red"];

function relationLabel(relation: string): string {
  return relation === "" ? "Other" : relation[0].toUpperCase() + relation.slice(1);
}

function occasionTitle(o: PersonOccurrence): string {
  const turned = o.turning ? ` — turns ${o.turning}` : "";
  return `${o.name} · ${o.label}${turned}`;
}

export function PeoplePage() {
  const people = usePlanner((s) => s.people);
  const workspaces = usePlanner((s) => s.workspaces);
  const activeWorkspace = usePlanner((s) => s.settings.activeWorkspace);
  const loadPeople = usePlanner((s) => s.loadPeople);
  const addPerson = usePlanner((s) => s.addPerson);
  const savePerson = usePlanner((s) => s.savePerson);
  const removePerson = usePlanner((s) => s.removePerson);
  const [query, setQuery] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [editing, setEditing] = useState<Person | "new" | null>(null);
  const [params, setParams] = useSearchParams();

  useEffect(() => {
    void loadPeople().then(() => setLoaded(true));
  }, [loadPeople]);

  // /people?edit=<id> — deep link from calendar chips.
  useEffect(() => {
    const id = params.get("edit");
    if (!id || !loaded) return;
    setParams({}, { replace: true });
    const target = people.find((p) => p.id === id);
    if (target) setEditing(target);
  }, [params, people, loaded, setParams]);

  const scoped = useMemo(
    () =>
      people.filter((p) => {
        if (activeWorkspace === "none" && p.workspaceId) return false;
        if (activeWorkspace && activeWorkspace !== "none" && p.workspaceId !== activeWorkspace) return false;
        return true;
      }),
    [people, activeWorkspace],
  );

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return scoped;
    return scoped.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.relation.toLowerCase().includes(q) ||
        p.notes.toLowerCase().includes(q) ||
        (p.dates ?? []).some((d) => d.label.toLowerCase().includes(q)),
    );
  }, [scoped, query]);

  const upcoming = useMemo(() => upcomingDates(scoped, 30), [scoped]);
  const next = useMemo(() => nextByPerson(scoped), [scoped]);

  const groups = useMemo(() => {
    const map = new Map<string, Person[]>();
    for (const p of visible) {
      const key = p.relation.trim().toLowerCase();
      map.set(key, [...(map.get(key) ?? []), p]);
    }
    const rank = (r: string) => {
      const i = RELATIONS.indexOf(r);
      return i === -1 ? RELATIONS.length + (r === "" ? 1 : 0) : i;
    };
    return [...map.entries()].sort((a, b) => rank(a[0]) - rank(b[0]) || a[0].localeCompare(b[0]));
  }, [visible]);

  function confirmRemove(p: Person) {
    if (!window.confirm(`Delete ${p.name}? This can't be undone.`)) return;
    void removePerson(p.id);
  }

  return (
    <>
      <PageHeader title="People" sub={`${scoped.length} ${scoped.length === 1 ? "person" : "people"}${activeWorkspace && activeWorkspace !== "none" ? " in this space" : ""}`}>
        <span className="people-search">
          <Search size={13} />
          <input
            className="people-search-input"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search people"
            aria-label="Search people"
          />
        </span>
        <button type="button" className="btn btn-primary" onClick={() => setEditing("new")}>
          <Plus size={14} strokeWidth={2.5} /> Add person
        </button>
      </PageHeader>

      <div className="page-scroll">
        {upcoming.length > 0 && (
          <div className="people-upcoming">
            {upcoming.map((o) => (
              <button
                key={o.id}
                type="button"
                className={`habit-chip ${o.daysUntil <= 1 ? "on" : ""}`}
                title={occasionTitle(o)}
                onClick={() => {
                  const p = people.find((x) => x.id === o.personId);
                  if (p) setEditing(p);
                }}
              >
                {o.label === "birthday" ? <Cake size={12} /> : <Heart size={12} />}
                {o.name} <b>{daysLabel(o.daysUntil)}</b>
              </button>
            ))}
          </div>
        )}

        {!loaded ? (
          <p className="panel-empty">Loading…</p>
        ) : visible.length === 0 ? (
          <div className="empty-hint">
            <strong>{query ? "No matches" : "No people yet"}</strong>
            <span>
              {query
                ? `Nothing matches “${query.trim()}”.`
                : "Add family, friends, your partner — birthdays and anniversaries land on the calendar."}
            </span>
            {!query && (
              <button type="button" className="btn btn-primary" onClick={() => setEditing("new")}>
                <Plus size={14} /> Add someone
              </button>
            )}
          </div>
        ) : (
          groups.map(([relation, members]) => (
            <section key={relation || "other"} className="people-group">
              <div className="side-label people-group-label">{relationLabel(relation)}</div>
              <ul className="people-list">
                {members.map((p) => {
                  const upcomingFor = next.get(p.id);
                  return (
                    <li key={p.id} className="person-row">
                      <span className="person-avatar" style={{ "--pc": `var(--c-${p.color || "slate"})` } as React.CSSProperties}>
                        {p.name.trim().charAt(0).toUpperCase() || "?"}
                      </span>
                      <button type="button" className="row-title person-name" onClick={() => setEditing(p)}>
                        {p.name}
                        {p.notes.trim() !== "" && <span className="person-note">{p.notes.split("\n")[0]}</span>}
                      </button>
                      {upcomingFor && (
                        <span className="meta-chip person-next" title={occasionTitle(upcomingFor)}>
                          {upcomingFor.label === "birthday" ? <Cake size={11} /> : <Heart size={11} />}
                          {formatDayShort(upcomingFor.date)} · {daysLabel(upcomingFor.daysUntil)}
                        </span>
                      )}
                      <button type="button" className="icon-btn" aria-label={`Edit ${p.name}`} onClick={() => setEditing(p)}>
                        <Pencil size={14} />
                      </button>
                      <button
                        type="button"
                        className="icon-btn danger"
                        aria-label={`Delete ${p.name}`}
                        onClick={() => confirmRemove(p)}
                      >
                        <Trash2 size={14} />
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))
        )}
      </div>

      {editing && (
        <PersonDialog
          person={editing === "new" ? undefined : editing}
          workspaces={workspaces}
          onClose={() => setEditing(null)}
          onDelete={editing === "new" ? undefined : () => {
            const p = editing as Person;
            if (!window.confirm(`Delete ${p.name}? This can't be undone.`)) return;
            void removePerson(p.id);
            setEditing(null);
          }}
          onSave={async (input) => {
            const out = editing === "new" ? await addPerson(input) : await savePerson((editing as Person).id, input);
            if (out) setEditing(null);
          }}
        />
      )}
    </>
  );
}

// PersonDialog — name, relation, birthday, named dates, notes, color, space.
function PersonDialog({
  person,
  workspaces,
  onClose,
  onSave,
  onDelete,
}: {
  person?: Person;
  workspaces: { id: string; name: string }[];
  onClose: () => void;
  onSave: (input: PersonInput) => Promise<void>;
  onDelete?: () => void;
}) {
  const toast = usePlanner((s) => s.toast);
  const [name, setName] = useState(person?.name ?? "");
  const [relation, setRelation] = useState(person?.relation ?? "");
  const [birthday, setBirthday] = useState(person?.birthday ?? "");
  const [dates, setDates] = useState<PersonDate[]>(person?.dates ?? []);
  const [notes, setNotes] = useState(person?.notes ?? "");
  const [color, setColor] = useState(person?.color ?? "slate");
  const [wsId, setWsId] = useState(person?.workspaceId ?? "");
  const [saving, setSaving] = useState(false);

  const relationOptions = relation && !RELATIONS.includes(relation) ? [...RELATIONS, relation] : RELATIONS;

  async function save() {
    const trimmed = name.trim();
    if (!trimmed) {
      toast("A name is required");
      return;
    }
    const kept = dates.filter((d) => d.label.trim() !== "" || d.date !== "");
    if (kept.some((d) => d.label.trim() === "" || d.date === "")) {
      toast("Each date needs a label and a day");
      return;
    }
    setSaving(true);
    try {
      await onSave({
        name: trimmed,
        relation,
        birthday,
        dates: kept.map((d) => ({ label: d.label.trim(), date: d.date })),
        notes: notes.trim(),
        color,
        workspaceId: wsId,
      });
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      className="scrim"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
      onKeyDown={(e) => {
        if (e.key === "Escape") onClose();
      }}
    >
      <div className="palette mail-dialog person-dialog" role="dialog" aria-label={person ? `Edit ${person.name}` : "New person"}>
        <h3>{person ? person.name : "New person"}</h3>
        <div className="field-row">
          <label className="field">
            <span>Name</span>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ada" autoFocus />
          </label>
          <label className="field">
            <span>Relation</span>
            <select className="select" value={relation} onChange={(e) => setRelation(e.target.value)}>
              <option value="">—</option>
              {relationOptions.map((r) => (
                <option key={r} value={r}>
                  {relationLabel(r)}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label className="field">
          <span>Birthday</span>
          <input className="input" type="date" value={birthday} onChange={(e) => setBirthday(e.target.value)} />
        </label>
        <div className="field">
          <span>Important dates</span>
          {dates.map((d, i) => (
            <div key={i} className="person-date-row">
              <input
                className="input"
                value={d.label}
                placeholder="Anniversary, nameday…"
                aria-label="Date label"
                onChange={(e) => setDates(dates.map((x, j) => (j === i ? { ...x, label: e.target.value } : x)))}
              />
              <input
                className="input person-date-when"
                type="date"
                value={d.date}
                aria-label="Date"
                onChange={(e) => setDates(dates.map((x, j) => (j === i ? { ...x, date: e.target.value } : x)))}
              />
              <button
                type="button"
                className="icon-btn"
                aria-label="Remove date"
                onClick={() => setDates(dates.filter((_, j) => j !== i))}
              >
                <X size={13} />
              </button>
            </div>
          ))}
          <button
            type="button"
            className="btn btn-secondary btn-xs"
            onClick={() => setDates([...dates, { label: "", date: "" }])}
          >
            <Plus size={12} /> Add date
          </button>
        </div>
        <label className="field">
          <span>Notes</span>
          <textarea
            className="textarea"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Likes, gift ideas, how you met…"
          />
        </label>
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
        {workspaces.length > 0 && (
          <label className="field">
            <span>Space</span>
            <select className="select" value={wsId} onChange={(e) => setWsId(e.target.value)}>
              <option value="">Personal</option>
              {workspaces.map((w) => (
                <option key={w.id} value={w.id}>
                  {w.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <div className="mail-dialog-actions">
          {onDelete && (
            <button type="button" className="btn btn-danger" onClick={onDelete}>
              <Trash2 size={14} /> Delete
            </button>
          )}
          <span style={{ flex: 1 }} />
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            Cancel
          </button>
          <button type="button" className="btn btn-primary" disabled={saving} onClick={() => void save()}>
            {person ? "Save" : "Create"}
          </button>
        </div>
      </div>
    </div>
  );
}

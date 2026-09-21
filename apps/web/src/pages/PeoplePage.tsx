// People — a warm, private spot for the people who matter: birthdays,
// anniversaries, namedays, notes. Personal by design, not a sales CRM.

import type { Person } from "@cal/api-client";
import { Cake, GitBranch, Heart, Link2, Pencil, Plus, Search, Star, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { PersonDialog, relationLabel } from "../components/PersonDialog";
import { formatDayShort } from "../lib/date";
import { daysLabel, nextByPerson, upcomingDates, type PersonOccurrence } from "../lib/people";
import { usePlanner } from "../stores/planner";

const RELATIONS = ["family", "partner", "friend", "colleague", "acquaintance"];

function occasionTitle(o: PersonOccurrence): string {
  const turned = o.turning ? ` — turns ${o.turning}` : "";
  return `${o.name} · ${o.label}${turned}`;
}

export function PeoplePage() {
  const api = usePlanner((s) => s.api);
  const people = usePlanner((s) => s.people);
  const workspaces = usePlanner((s) => s.workspaces);
  const activeWorkspace = usePlanner((s) => s.settings.activeWorkspace);
  const peopleShareToken = usePlanner((s) => s.settings.peopleShareToken);
  const toast = usePlanner((s) => s.toast);
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
        (p.nickname ?? "").toLowerCase().includes(q) ||
        p.relation.toLowerCase().includes(q) ||
        p.notes.toLowerCase().includes(q) ||
        (p.tags ?? []).some((t) => t.toLowerCase().includes(q)) ||
        (p.dates ?? []).some((d) => d.label.toLowerCase().includes(q)),
    );
  }, [scoped, query]);

  const upcoming = useMemo(() => upcomingDates(scoped, 30), [scoped]);
  const next = useMemo(() => nextByPerson(scoped), [scoped]);
  const favorites = useMemo(() => visible.filter((p) => p.isFavorite), [visible]);

  const groups = useMemo(() => {
    const map = new Map<string, Person[]>();
    for (const p of visible.filter((p) => !p.isFavorite)) {
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

  function row(p: Person) {
    const upcomingFor = next.get(p.id);
    return (
      <li key={p.id} className="person-row">
        <Link to={`/people/${p.id}`} className="person-avatar" style={{ "--pc": `var(--c-${p.color || "slate"})` } as React.CSSProperties}>
          {p.avatar ? <img src={api.assetUrl(`/files/${p.avatar}`)} alt="" /> : p.name.trim().charAt(0).toUpperCase() || "?"}
        </Link>
        <Link to={`/people/${p.id}`} className="row-title person-name">
          {p.name}
          {p.isFavorite && <Star size={11} className="person-fav" fill="currentColor" />}
          {(p.tags ?? []).slice(0, 3).map((t) => (
            <span key={t} className="tag-chip person-tag">
              {t}
            </span>
          ))}
          {p.notes.trim() !== "" && <span className="person-note">{p.notes.split("\n")[0]}</span>}
        </Link>
        {upcomingFor && (
          <span className="meta-chip person-next" title={occasionTitle(upcomingFor)}>
            {upcomingFor.label === "birthday" ? <Cake size={11} /> : <Heart size={11} />}
            {formatDayShort(upcomingFor.date)} · {daysLabel(upcomingFor.daysUntil)}
          </span>
        )}
        <button type="button" className="icon-btn" aria-label={`Edit ${p.name}`} onClick={() => setEditing(p)}>
          <Pencil size={14} />
        </button>
        <button type="button" className="icon-btn danger" aria-label={`Delete ${p.name}`} onClick={() => confirmRemove(p)}>
          <Trash2 size={14} />
        </button>
      </li>
    );
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
        <button
          type="button"
          className="btn btn-secondary"
          title={peopleShareToken ? "Copy the public birthday-list link" : "Share birthdays & dates as a public page"}
          onClick={() => {
            if (peopleShareToken) {
              const url = `${api.remote || location.origin}/people/shared/${peopleShareToken}`;
              void navigator.clipboard.writeText(url).then(
                () => toast("Share link copied"),
                () => toast(`Share link: ${url}`),
              );
              return;
            }
            void api
              .sharePeople(true)
              .then(({ shareToken }) => {
                if (!shareToken) return;
                usePlanner.setState({ settings: { ...usePlanner.getState().settings, peopleShareToken: shareToken } });
                const url = `${api.remote || location.origin}/people/shared/${shareToken}`;
                void navigator.clipboard.writeText(url).then(
                  () => toast("Share link copied"),
                  () => toast(`Share link: ${url}`),
                );
              })
              .catch(() => toast("Share failed"));
          }}
        >
          <Link2 size={14} /> {peopleShareToken ? "Copy link" : "Share"}
        </button>
        {peopleShareToken && (
          <button
            type="button"
            className="icon-btn danger"
            aria-label="Turn off the public people page"
            title="Revoke public link"
            onClick={() => {
              if (!window.confirm("Turn off the public people page? The link stops working.")) return;
              void api
                .sharePeople(false)
                .then(() => {
                  usePlanner.setState({ settings: { ...usePlanner.getState().settings, peopleShareToken: "" } });
                  toast("Sharing off");
                })
                .catch(() => toast("Share failed"));
            }}
          >
            <Trash2 size={14} />
          </button>
        )}
        <Link className="btn btn-secondary" to="/people/tree" title="Family tree">
          <GitBranch size={14} /> Tree
        </Link>
        <button type="button" className="btn btn-primary" onClick={() => setEditing("new")}>
          <Plus size={14} strokeWidth={2.5} /> Add person
        </button>
      </PageHeader>

      <div className="page-scroll">
        {upcoming.length > 0 && (
          <div className="people-upcoming">
            {upcoming.map((o) => (
              <Link
                key={o.id}
                to={`/people/${o.personId}`}
                className={`habit-chip ${o.daysUntil <= 1 ? "on" : ""}`}
                title={occasionTitle(o)}
              >
                {o.label === "birthday" ? <Cake size={12} /> : <Heart size={12} />}
                {o.name} <b>{daysLabel(o.daysUntil)}</b>
              </Link>
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
          <>
            {favorites.length > 0 && (
              <section className="people-group">
                <div className="side-label people-group-label">Favorites</div>
                <ul className="people-list">{favorites.map(row)}</ul>
              </section>
            )}
            {groups.map(([relation, members]) => (
              <section key={relation || "other"} className="people-group">
                <div className="side-label people-group-label">{relationLabel(relation)}</div>
                <ul className="people-list">{members.map(row)}</ul>
              </section>
            ))}
          </>
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

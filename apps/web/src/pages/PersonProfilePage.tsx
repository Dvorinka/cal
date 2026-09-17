// Person profile — the full record for one person: contact details, dates,
// relationships, timeline history and attachments. Edits go through the
// shared PersonDialog; relations, timeline and files are managed inline.

import type { FileRec, Person, PersonRelation, TimelineItem } from "@cal/api-client";
import {
  ArrowLeft, Cake, File as FileIcon, FileArchive, FileAudio, FileImage, FileText, FileVideo,
  Gift, Heart, Link2, Mail, MapPin, Pencil, Phone, Plus, Star, Trash2, Upload, X,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { PersonDialog, relationLabel } from "../components/PersonDialog";
import { formatDayShort } from "../lib/date";
import { daysLabel, nextByPerson, personToInput, upcomingDates } from "../lib/people";
import { reportErr, usePlanner } from "../stores/planner";

const REL_KINDS = ["parent", "child", "sibling", "partner", "friend", "coworker", "mentor"];
const TIMELINE_TYPES = ["met", "gift", "trip", "achievement", "memory", "note"];

function fileIcon(mime: string) {
  if (mime.startsWith("image/")) return FileImage;
  if (mime.startsWith("video/")) return FileVideo;
  if (mime.startsWith("audio/")) return FileAudio;
  if (mime.startsWith("text/") || mime === "application/pdf") return FileText;
  if (/zip|tar|gzip|7z|rar/.test(mime)) return FileArchive;
  return FileIcon;
}

// Edges read "to is from's kind" — Mum --child--> Ana shows "child: Ana" on
// Mum's profile and "parent: Mum" on Ana's. Incoming edges show the inverse.
const INV_KIND: Record<string, string> = {
  parent: "child",
  child: "parent",
  mentor: "mentee",
};

function relKindLabel(kind: string, outgoing: boolean): string {
  return outgoing ? kind : (INV_KIND[kind] ?? kind);
}

export function PersonProfilePage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const api = usePlanner((s) => s.api);
  const people = usePlanner((s) => s.people);
  const workspaces = usePlanner((s) => s.workspaces);
  const loadPeople = usePlanner((s) => s.loadPeople);
  const savePerson = usePlanner((s) => s.savePerson);
  const removePerson = usePlanner((s) => s.removePerson);
  const toast = usePlanner((s) => s.toast);

  const person = people.find((p) => p.id === id);
  const [missing, setMissing] = useState(false);
  const [editing, setEditing] = useState(false);
  const [relations, setRelations] = useState<PersonRelation[]>([]);
  const [timeline, setTimeline] = useState<TimelineItem[]>([]);
  const [files, setFiles] = useState<FileRec[]>([]);
  const [linkForm, setLinkForm] = useState<{ toId: string; kind: string }>({ toId: "", kind: "friend" });
  const [tlForm, setTlForm] = useState<{ type: string; title: string; body: string; occurredOn: string } | null>(null);
  const [tlEdit, setTlEdit] = useState<TimelineItem | null>(null);
  const avatarInput = useRef<HTMLInputElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (people.length === 0) void loadPeople();
  }, [people.length, loadPeople]);

  // Cold navigation: the list may be loaded without this id (deleted or
  // filtered out) — try the direct fetch before declaring it missing.
  useEffect(() => {
    if (!id || person) return;
    api
      .person(id)
      .then((p) => {
        usePlanner.setState({ people: [...usePlanner.getState().people.filter((x) => x.id !== p.id), p] });
      })
      .catch(() => setMissing(true));
  }, [id, person, api]);

  const reload = () => {
    if (!id) return;
    api.personRelations(id).then(setRelations).catch(() => setRelations([]));
    api.personTimeline(id).then(setTimeline).catch(() => setTimeline([]));
    api.personFiles(id).then(setFiles).catch(() => setFiles([]));
  };
  useEffect(reload, [id, api]);

  const upcoming = useMemo(() => (person ? upcomingDates([person], 366) : []), [person]);
  const nextAll = useMemo(() => nextByPerson(people), [people]);

  async function patch(partial: Partial<ReturnType<typeof personToInput>>) {
    if (!person) return;
    const out = await savePerson(person.id, { ...personToInput(person), ...partial });
    if (!out) toast("Save failed");
  }

  async function uploadAvatar(file: File) {
    if (!person) return;
    try {
      const up = await api.upload(file, { personId: person.id });
      await patch({ avatar: up.name });
      reload();
    } catch (e) {
      reportErr("Avatar upload failed")(e);
    }
  }

  async function uploadFiles(list: FileList | null) {
    if (!list || !person) return;
    for (const f of Array.from(list)) {
      try {
        await api.upload(f, { personId: person.id });
      } catch (e) {
        reportErr(`Upload failed: ${f.name}`)(e);
      }
    }
    reload();
  }

  if (missing) {
    return (
      <>
        <PageHeader title="Person" />
        <div className="page-scroll">
          <div className="empty-hint">
            <strong>Not found</strong>
            <span>This person may have been deleted.</span>
            <Link className="btn btn-primary" to="/people">
              <ArrowLeft size={14} /> Back to people
            </Link>
          </div>
        </div>
      </>
    );
  }

  if (!person) {
    return (
      <>
        <PageHeader title="Person" />
        <div className="page-scroll">
          <p className="panel-empty">Loading…</p>
        </div>
      </>
    );
  }

  const workspace = workspaces.find((w) => w.id === person.workspaceId);
  const others = people.filter((p) => p.id !== person.id);

  return (
    <>
      <PageHeader
        title={person.name}
        sub={[person.nickname && `“${person.nickname}”`, relationLabel(person.relation), workspace?.name].filter(Boolean).join(" · ")}
      >
        <Link className="btn btn-ghost" to="/people">
          <ArrowLeft size={14} /> People
        </Link>
        <button type="button" className="btn btn-secondary" onClick={() => setEditing(true)}>
          <Pencil size={14} /> Edit
        </button>
      </PageHeader>

      <div className="page-scroll person-profile">
        <div className="person-hero">
          <button
            type="button"
            className="person-hero-avatar"
            style={{ "--pc": `var(--c-${person.color || "slate"})` } as React.CSSProperties}
            title="Change photo"
            onClick={() => avatarInput.current?.click()}
          >
            {person.avatar ? (
              <img src={api.assetUrl(`/files/${person.avatar}`)} alt={person.name} />
            ) : (
              <span>{person.name.trim().charAt(0).toUpperCase() || "?"}</span>
            )}
          </button>
          <input
            ref={avatarInput}
            type="file"
            accept="image/*"
            hidden
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) void uploadAvatar(f);
              e.target.value = "";
            }}
          />
          <div className="person-hero-meta">
            <div className="person-hero-name">
              {person.name}
              <button
                type="button"
                className={`icon-btn fav-toggle ${person.isFavorite ? "on" : ""}`}
                aria-label={person.isFavorite ? "Remove from favorites" : "Mark as favorite"}
                onClick={() => void patch({ isFavorite: !person.isFavorite })}
              >
                <Star size={15} fill={person.isFavorite ? "currentColor" : "none"} />
              </button>
            </div>
            <div className="person-hero-chips">
              {person.relation && <span className="meta-chip">{relationLabel(person.relation)}</span>}
              {(person.tags ?? []).map((t) => (
                <span key={t} className="tag-chip">
                  {t}
                </span>
              ))}
            </div>
          </div>
        </div>

        <div className="person-grid">
          <section className="panel person-card">
            <div className="side-label">Dates</div>
            <ul className="person-facts">
              {person.birthday && (
                <li>
                  <Cake size={13} />
                  <span className="fact-label">Birthday</span>
                  <span className="fact-value">
                    {formatDayShort(person.birthday)}
                    {person.birthdayRemind !== undefined && <em className="remind-badge">−{person.birthdayRemind}d</em>}
                  </span>
                </li>
              )}
              {(person.dates ?? []).map((d, i) => {
                const occ = upcoming.find((o) => o.label === d.label);
                return (
                  <li key={i}>
                    <Heart size={13} />
                    <span className="fact-label">{d.label}</span>
                    <span className="fact-value">
                      {formatDayShort(d.date)}
                      {occ && <em className="next-badge">{daysLabel(occ.daysUntil)}</em>}
                      {d.remindDays !== undefined && <em className="remind-badge">−{d.remindDays}d</em>}
                    </span>
                  </li>
                );
              })}
              {!person.birthday && (person.dates ?? []).length === 0 && <li className="fact-empty">No dates yet — birthdays and anniversaries land on the calendar.</li>}
            </ul>
          </section>

          <section className="panel person-card">
            <div className="side-label">Contact</div>
            <ul className="person-facts">
              {person.phone && (
                <li>
                  <Phone size={13} />
                  <span className="fact-label">Phone</span>
                  <a className="fact-value" href={`tel:${person.phone}`}>
                    {person.phone}
                  </a>
                </li>
              )}
              {person.email && (
                <li>
                  <Mail size={13} />
                  <span className="fact-label">Email</span>
                  <a className="fact-value" href={`mailto:${person.email}`}>
                    {person.email}
                  </a>
                </li>
              )}
              {person.address && (
                <li>
                  <MapPin size={13} />
                  <span className="fact-label">Address</span>
                  <span className="fact-value">{person.address}</span>
                </li>
              )}
              {(person.links ?? []).map((l, i) => (
                <li key={i}>
                  <Link2 size={13} />
                  <span className="fact-label">{l.platform || "link"}</span>
                  <a className="fact-value" href={l.url} target="_blank" rel="noreferrer">
                    {l.url.replace(/^https?:\/\//, "").slice(0, 40)}
                  </a>
                </li>
              ))}
              {!person.phone && !person.email && !person.address && (person.links ?? []).length === 0 && (
                <li className="fact-empty">No contact details yet.</li>
              )}
            </ul>
            {(person.fields ?? []).length > 0 && (
              <ul className="person-facts person-custom">
                {(person.fields ?? []).map((f, i) => (
                  <li key={i}>
                    <span className="fact-label">{f.key}</span>
                    <span className="fact-value">{f.value}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="panel person-card">
            <div className="side-label">Relationships</div>
            <div className="rel-chips">
              {relations.map((r) => (
                <span key={r.id} className="meta-chip rel-chip">
                  {relKindLabel(r.kind, r.outgoing)}{" "}
                  <Link to={`/people/${r.otherId}`} className="rel-name">
                    {r.otherName ?? nextAll.get(r.otherId)?.name ?? "…"}
                  </Link>
                  <button
                    type="button"
                    className="icon-btn rel-x"
                    aria-label="Remove relationship"
                    onClick={() => void api.unlinkPersons(r.id).then(reload).catch(reportErr("Remove failed"))}
                  >
                    <X size={11} />
                  </button>
                </span>
              ))}
              {relations.length === 0 && <span className="fact-empty">No links yet — parents, partners, friends.</span>}
            </div>
            {others.length > 0 && (
              <div className="rel-add">
                <select
                  className="select"
                  value={linkForm.toId}
                  aria-label="Person"
                  onChange={(e) => setLinkForm({ ...linkForm, toId: e.target.value })}
                >
                  <option value="">Link to…</option>
                  {others.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
                <select
                  className="select"
                  value={linkForm.kind}
                  aria-label="Relationship"
                  onChange={(e) => setLinkForm({ ...linkForm, kind: e.target.value })}
                >
                  {REL_KINDS.map((k) => (
                    <option key={k} value={k}>
                      {k}
                    </option>
                  ))}
                </select>
                <button
                  type="button"
                  className="btn btn-secondary btn-xs"
                  disabled={!linkForm.toId}
                  onClick={() => {
                    if (!linkForm.toId) return;
                    void api
                      .linkPersons(person.id, linkForm.toId, linkForm.kind)
                      .then(() => {
                        setLinkForm({ toId: "", kind: linkForm.kind });
                        reload();
                      })
                      .catch(reportErr("Link failed"));
                  }}
                >
                  <Plus size={12} /> Link
                </button>
              </div>
            )}
          </section>

          <section className="panel person-card">
            <div className="side-label">About</div>
            {person.notes && <p className="person-about-text">{person.notes}</p>}
            {person.giftIdeas && (
              <p className="person-about-text">
                <Gift size={12} /> <b>Gift ideas:</b> {person.giftIdeas}
              </p>
            )}
            {person.interests && (
              <p className="person-about-text">
                <b>Interests:</b> {person.interests}
              </p>
            )}
            {!person.notes && !person.giftIdeas && !person.interests && <p className="fact-empty">Nothing here yet.</p>}
          </section>

          <section className="panel person-card person-timeline-card">
            <div className="side-label">Timeline</div>
            <ul className="person-timeline">
              {timeline.map((t) =>
                tlEdit?.id === t.id ? (
                  <li key={t.id} className="tl-edit">
                    <TimelineForm
                      initial={{
                        type: tlEdit.type,
                        title: tlEdit.title,
                        body: tlEdit.body ?? "",
                        occurredOn: tlEdit.occurredOn ?? "",
                      }}
                      onCancel={() => setTlEdit(null)}
                      onSave={async (form) => {
                        await api.updateTimelineItem(person.id, t.id, form);
                        setTlEdit(null);
                        reload();
                      }}
                    />
                  </li>
                ) : (
                  <li key={t.id}>
                    <span className={`tl-dot tl-${t.type}`} />
                    <div className="tl-body">
                      <div className="tl-head">
                        <b>{t.title}</b>
                        <span className="tl-meta">
                          {t.type}
                          {t.occurredOn ? ` · ${formatDayShort(t.occurredOn)}` : ""}
                        </span>
                      </div>
                      {t.body && <p className="tl-text">{t.body}</p>}
                    </div>
                    <span className="tl-actions">
                      <button type="button" className="icon-btn" aria-label="Edit" onClick={() => setTlEdit(t)}>
                        <Pencil size={12} />
                      </button>
                      <button
                        type="button"
                        className="icon-btn danger"
                        aria-label="Delete"
                        onClick={() => void api.deleteTimelineItem(person.id, t.id).then(reload).catch(reportErr("Delete failed"))}
                      >
                        <Trash2 size={12} />
                      </button>
                    </span>
                  </li>
                ),
              )}
              {timeline.length === 0 && <li className="fact-empty">No memories yet — first meeting, trips, gifts.</li>}
            </ul>
            {tlForm ? (
              <TimelineForm
                initial={tlForm}
                onCancel={() => setTlForm(null)}
                onSave={async (form) => {
                  await api.createTimelineItem(person.id, form);
                  setTlForm(null);
                  reload();
                }}
              />
            ) : (
              <button
                type="button"
                className="btn btn-secondary btn-xs"
                onClick={() => setTlForm({ type: "note", title: "", body: "", occurredOn: "" })}
              >
                <Plus size={12} /> Add memory
              </button>
            )}
          </section>

          <section className="panel person-card">
            <div className="side-label">
              Attachments
              <button type="button" className="icon-btn card-action" aria-label="Upload file" onClick={() => fileInput.current?.click()}>
                <Upload size={13} />
              </button>
            </div>
            <input
              ref={fileInput}
              type="file"
              multiple
              hidden
              onChange={(e) => {
                void uploadFiles(e.target.files);
                e.target.value = "";
              }}
            />
            <ul className="person-files">
              {files.map((f) => {
                const Icon = fileIcon(f.mime);
                const isImage = f.mime.startsWith("image/");
                return (
                  <li key={f.id}>
                    <a href={api.assetUrl(`/files/${f.name}`)} target="_blank" rel="noreferrer" className="person-file">
                      {isImage ? (
                        <img src={api.assetUrl(`/files/${f.name}`)} alt={f.origName} className="person-file-thumb" />
                      ) : (
                        <Icon size={15} />
                      )}
                      <span className="person-file-name">{f.origName}</span>
                    </a>
                    <button
                      type="button"
                      className="icon-btn danger"
                      aria-label={`Delete ${f.origName}`}
                      onClick={() => {
                        if (!window.confirm(`Delete ${f.origName}?`)) return;
                        void api.deleteFile(f.id).then(reload).catch(reportErr("Delete failed"));
                      }}
                    >
                      <Trash2 size={12} />
                    </button>
                  </li>
                );
              })}
              {files.length === 0 && <li className="fact-empty">No attachments — photos, documents, voice notes.</li>}
            </ul>
          </section>
        </div>
      </div>

      {editing && (
        <PersonDialog
          person={person}
          workspaces={workspaces}
          onClose={() => setEditing(false)}
          onDelete={() => {
            if (!window.confirm(`Delete ${person.name}? This can't be undone.`)) return;
            void removePerson(person.id).then(() => navigate("/people"));
          }}
          onSave={async (input) => {
            const out = await savePerson(person.id, input);
            if (out) setEditing(false);
          }}
        />
      )}
    </>
  );
}

// TimelineForm — add/edit one timeline entry inline.
function TimelineForm({
  initial,
  onSave,
  onCancel,
}: {
  initial: { type: string; title: string; body: string; occurredOn: string };
  onSave: (form: { type: string; title: string; body: string; occurredOn: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const [form, setForm] = useState(initial);
  const [busy, setBusy] = useState(false);
  const toast = usePlanner((s) => s.toast);
  return (
    <div className="tl-form">
      <div className="field-row">
        <select className="select" value={form.type} aria-label="Type" onChange={(e) => setForm({ ...form, type: e.target.value })}>
          {TIMELINE_TYPES.map((t) => (
            <option key={t} value={t}>
              {t}
            </option>
          ))}
        </select>
        <input
          className="input"
          value={form.title}
          placeholder="What happened?"
          aria-label="Title"
          autoFocus
          onChange={(e) => setForm({ ...form, title: e.target.value })}
        />
        <input
          className="input tl-form-date"
          type="date"
          value={form.occurredOn}
          aria-label="Date"
          onChange={(e) => setForm({ ...form, occurredOn: e.target.value })}
        />
      </div>
      <textarea
        className="textarea textarea-sm"
        value={form.body}
        placeholder="Details…"
        aria-label="Details"
        onChange={(e) => setForm({ ...form, body: e.target.value })}
      />
      <div className="tl-form-actions">
        <button type="button" className="btn btn-ghost btn-xs" onClick={onCancel}>
          Cancel
        </button>
        <button
          type="button"
          className="btn btn-primary btn-xs"
          disabled={busy || !form.title.trim()}
          onClick={() => {
            if (!form.title.trim()) {
              toast("A title is required");
              return;
            }
            setBusy(true);
            void onSave(form)
              .catch(reportErr("Save failed"))
              .finally(() => setBusy(false));
          }}
        >
          Save
        </button>
      </div>
    </div>
  );
}

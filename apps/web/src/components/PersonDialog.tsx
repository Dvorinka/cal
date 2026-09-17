// PersonDialog — the full person editor: basics, contact, dates (with a
// nameday lookup), tags, about-text, custom fields and social links. Saves
// a complete PersonInput — the API replaces the whole record.

import type { NamedayResult, Person, PersonDate, PersonField, PersonInput, PersonLink } from "@cal/api-client";
import { Plus, Search, Sparkles, Star, Trash2, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { usePlanner } from "../stores/planner";

const RELATIONS = ["family", "partner", "friend", "colleague", "acquaintance"];
const COLORS = ["slate", "mint", "sky", "violet", "amber", "orange", "rose", "red"];
const REMIND_PRESETS = [0, 1, 3, 7, 14, 30];

export function relationLabel(relation: string): string {
  return relation === "" ? "Other" : relation[0].toUpperCase() + relation.slice(1);
}

// remindLabel renders a lead-time select shared by the birthday row and
// each named date row.
function RemindSelect({ value, onChange, label }: { value: number | undefined; onChange: (v: number | undefined) => void; label: string }) {
  return (
    <select
      className="select remind-select"
      value={value ?? -1}
      aria-label={label}
      title={label}
      onChange={(e) => {
        const v = Number(e.target.value);
        onChange(v < 0 ? undefined : v);
      }}
    >
      <option value={-1}>no reminder</option>
      {REMIND_PRESETS.map((d) => (
        <option key={d} value={d}>
          {d === 0 ? "on the day" : `${d}d before`}
        </option>
      ))}
    </select>
  );
}

// NamedayLookup — type a first name, pick a country's date, it lands as a
// "nameday" entry in the dates list. Debounced 400ms.
function NamedayLookup({ onPick }: { onPick: (date: string, name: string) => void }) {
  const api = usePlanner((s) => s.api);
  const namedayCountry = usePlanner((s) => s.settings.namedayCountry);
  const [q, setQ] = useState("");
  const [results, setResults] = useState<NamedayResult[]>([]);
  const [open, setOpen] = useState(false);
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => {
    window.clearTimeout(timer.current);
    const name = q.trim();
    if (name.length < 2) {
      setResults([]);
      return;
    }
    timer.current = window.setTimeout(() => {
      void api
        .searchNamedays(name)
        .then(setResults)
        .catch(() => setResults([]));
    }, 400);
    return () => window.clearTimeout(timer.current);
  }, [q, api]);

  const flat = useMemo(
    () =>
      results.flatMap((r) =>
        r.dates.map((d) => ({
          key: `${r.country}-${d.month}-${d.day}-${d.name}`,
          country: r.country,
          label: `${r.country.toUpperCase()} · ${d.name}`,
          date: `2000-${String(d.month).padStart(2, "0")}-${String(d.day).padStart(2, "0")}`,
        })),
      ),
    [results],
  );
  // The configured country floats first so the common case is one click.
  const sorted = useMemo(
    () =>
      namedayCountry
        ? [...flat].sort((a, b) => Number(b.country === namedayCountry) - Number(a.country === namedayCountry))
        : flat,
    [flat, namedayCountry],
  );

  return (
    <div className="nameday-lookup">
      <span className="people-search nameday-search">
        <Search size={13} />
        <input
          className="people-search-input"
          value={q}
          placeholder="Find a nameday…"
          aria-label="Find a nameday"
          onChange={(e) => {
            setQ(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => window.setTimeout(() => setOpen(false), 150)}
        />
      </span>
      {open && flat.length > 0 && (
        <ul className="nameday-results">
          {sorted.slice(0, 8).map((r) => (
            <li key={r.key}>
              <button
                type="button"
                className="nameday-result"
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => {
                  onPick(r.date, r.label);
                  setQ("");
                  setResults([]);
                  setOpen(false);
                }}
              >
                <Sparkles size={11} /> {r.label} <span className="nameday-date">{r.date.slice(5)}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function PersonDialog({
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
  const [nickname, setNickname] = useState(person?.nickname ?? "");
  const [relation, setRelation] = useState(person?.relation ?? "");
  const [favorite, setFavorite] = useState(person?.isFavorite ?? false);
  const [birthday, setBirthday] = useState(person?.birthday ?? "");
  const [birthdayRemind, setBirthdayRemind] = useState<number | undefined>(person?.birthdayRemind);
  const [dates, setDates] = useState<PersonDate[]>(person?.dates ?? []);
  const [phone, setPhone] = useState(person?.phone ?? "");
  const [email, setEmail] = useState(person?.email ?? "");
  const [address, setAddress] = useState(person?.address ?? "");
  const [tags, setTags] = useState((person?.tags ?? []).join(", "));
  const [notes, setNotes] = useState(person?.notes ?? "");
  const [giftIdeas, setGiftIdeas] = useState(person?.giftIdeas ?? "");
  const [interests, setInterests] = useState(person?.interests ?? "");
  const [fields, setFields] = useState<PersonField[]>(person?.fields ?? []);
  const [links, setLinks] = useState<PersonLink[]>(person?.links ?? []);
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
        nickname: nickname.trim(),
        relation,
        isFavorite: favorite,
        birthday,
        birthdayRemind: birthdayRemind ?? null,
        dates: kept.map((d) => ({ label: d.label.trim(), date: d.date, remindDays: d.remindDays })),
        phone: phone.trim(),
        email: email.trim(),
        address: address.trim(),
        tags: tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
        notes: notes.trim(),
        giftIdeas: giftIdeas.trim(),
        interests: interests.trim(),
        fields: fields.filter((f) => f.key.trim() !== "").map((f) => ({ key: f.key.trim(), value: f.value })),
        links: links.filter((l) => l.url.trim() !== "").map((l) => ({ platform: l.platform.trim(), url: l.url.trim() })),
        avatar: person?.avatar ?? "",
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
        <div className="person-dialog-head">
          <h3>{person ? person.name : "New person"}</h3>
          <button
            type="button"
            className={`icon-btn fav-toggle ${favorite ? "on" : ""}`}
            aria-label={favorite ? "Remove from favorites" : "Mark as favorite"}
            title={favorite ? "Favorite" : "Mark as favorite"}
            onClick={() => setFavorite(!favorite)}
          >
            <Star size={16} fill={favorite ? "currentColor" : "none"} />
          </button>
        </div>
        <div className="field-row">
          <label className="field">
            <span>Name</span>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} placeholder="Ada" autoFocus />
          </label>
          <label className="field">
            <span>Nickname</span>
            <input className="input" value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="Addie" />
          </label>
        </div>
        <div className="field-row">
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
          <label className="field">
            <span>Birthday</span>
            <input className="input" type="date" value={birthday} onChange={(e) => setBirthday(e.target.value)} />
          </label>
        </div>
        {birthday && (
          <div className="field birthday-remind">
            <span>Birthday reminder</span>
            <RemindSelect value={birthdayRemind} onChange={setBirthdayRemind} label="Birthday reminder" />
          </div>
        )}
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
              <RemindSelect
                value={d.remindDays}
                label={`Reminder for ${d.label || "this date"}`}
                onChange={(v) => setDates(dates.map((x, j) => (j === i ? { ...x, remindDays: v } : x)))}
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
          <div className="person-date-actions">
            <button
              type="button"
              className="btn btn-secondary btn-xs"
              onClick={() => setDates([...dates, { label: "", date: "" }])}
            >
              <Plus size={12} /> Add date
            </button>
            <NamedayLookup
              onPick={(date, label) =>
                setDates([...dates, { label: `nameday (${label.split("·")[1]?.trim() ?? ""})`.replace(" ()", ""), date }])
              }
            />
          </div>
        </div>

        <details className="person-section">
          <summary>Contact</summary>
          <div className="field-row">
            <label className="field">
              <span>Phone</span>
              <input className="input" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+420 …" />
            </label>
            <label className="field">
              <span>Email</span>
              <input className="input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="ada@…" />
            </label>
          </div>
          <label className="field">
            <span>Address</span>
            <input className="input" value={address} onChange={(e) => setAddress(e.target.value)} placeholder="Street, city" />
          </label>
        </details>

        <details className="person-section" open={!!person?.notes || !!person?.giftIdeas || !!person?.interests}>
          <summary>About</summary>
          <label className="field">
            <span>Notes</span>
            <textarea
              className="textarea"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="How you met, things to remember…"
            />
          </label>
          <label className="field">
            <span>Gift ideas</span>
            <textarea
              className="textarea textarea-sm"
              value={giftIdeas}
              onChange={(e) => setGiftIdeas(e.target.value)}
              placeholder="Books, tea, that thing they mentioned…"
            />
          </label>
          <label className="field">
            <span>Interests</span>
            <textarea
              className="textarea textarea-sm"
              value={interests}
              onChange={(e) => setInterests(e.target.value)}
              placeholder="Gardening, chess, jazz…"
            />
          </label>
        </details>

        <details className="person-section" open={(person?.links?.length ?? 0) > 0}>
          <summary>Links</summary>
          <div className="field">
            {links.map((l, i) => (
              <div key={i} className="person-date-row">
                <input
                  className="input person-link-platform"
                  value={l.platform}
                  placeholder="instagram"
                  aria-label="Platform"
                  onChange={(e) => setLinks(links.map((x, j) => (j === i ? { ...x, platform: e.target.value } : x)))}
                />
                <input
                  className="input"
                  value={l.url}
                  placeholder="https://…"
                  aria-label="URL"
                  onChange={(e) => setLinks(links.map((x, j) => (j === i ? { ...x, url: e.target.value } : x)))}
                />
                <button type="button" className="icon-btn" aria-label="Remove link" onClick={() => setLinks(links.filter((_, j) => j !== i))}>
                  <X size={13} />
                </button>
              </div>
            ))}
            <button type="button" className="btn btn-secondary btn-xs" onClick={() => setLinks([...links, { platform: "", url: "" }])}>
              <Plus size={12} /> Add link
            </button>
          </div>
        </details>

        <details className="person-section" open={(person?.fields?.length ?? 0) > 0}>
          <summary>Custom fields</summary>
          <div className="field">
            {fields.map((f, i) => (
              <div key={i} className="person-date-row">
                <input
                  className="input person-link-platform"
                  value={f.key}
                  placeholder="shoe size"
                  aria-label="Field name"
                  onChange={(e) => setFields(fields.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)))}
                />
                <input
                  className="input"
                  value={f.value}
                  placeholder="38"
                  aria-label="Field value"
                  onChange={(e) => setFields(fields.map((x, j) => (j === i ? { ...x, value: e.target.value } : x)))}
                />
                <button type="button" className="icon-btn" aria-label="Remove field" onClick={() => setFields(fields.filter((_, j) => j !== i))}>
                  <X size={13} />
                </button>
              </div>
            ))}
            <button type="button" className="btn btn-secondary btn-xs" onClick={() => setFields([...fields, { key: "", value: "" }])}>
              <Plus size={12} /> Add field
            </button>
          </div>
        </details>

        <div className="field-row">
          <label className="field">
            <span>Tags</span>
            <input className="input" value={tags} onChange={(e) => setTags(e.target.value)} placeholder="family, vip" />
          </label>
        </div>
        <div className="field-row person-dialog-foot">
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
        </div>
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


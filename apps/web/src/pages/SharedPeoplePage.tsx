// SharedPeoplePage — public birthday/date list at /people/shared/:token.
// No auth; the token is the capability. Shows only names and dates.

import type { SharedPerson } from "@cal/api-client";
import { CalApi } from "@cal/api-client";
import { Cake, Heart } from "lucide-react";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { iso, todayIso } from "../lib/date";

const api = new CalApi("/api");

interface Occasion {
  name: string;
  label: string;
  date: string; // next occurrence YYYY-MM-DD
  daysUntil: number;
  turning?: number;
  color: string;
}

// nextOccurrence maps a stored YYYY-MM-DD onto the coming instance —
// Feb 29 falls back to Feb 28 on non-leap years.
function nextOccurrence(stored: string): { date: string; daysUntil: number; turning?: number } | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(stored);
  if (!m) return null;
  const year0 = +m[1], month = +m[2], day = +m[3];
  const today = todayIso();
  for (const year of [new Date().getFullYear(), new Date().getFullYear() + 1]) {
    const d = new Date(year, month - 1, day);
    const occ = d.getMonth() === month - 1 ? d : new Date(year, month - 1, day - 1);
    const occIso = iso(occ);
    if (occIso >= today) {
      const daysUntil = Math.round((occ.getTime() - new Date(today + "T00:00:00").getTime()) / 86400000);
      return { date: occIso, daysUntil, turning: year - year0 > 0 ? year - year0 : undefined };
    }
  }
  return null;
}

function formatDate(iso: string): string {
  return new Date(iso + "T00:00:00").toLocaleDateString(undefined, { month: "long", day: "numeric" });
}

function daysLabel(n: number): string {
  if (n === 0) return "today";
  if (n === 1) return "tomorrow";
  return `in ${n} days`;
}

export function SharedPeoplePage() {
  const { token } = useParams();
  const [people, setPeople] = useState<SharedPerson[]>();
  const [err, setErr] = useState(false);

  useEffect(() => {
    if (token) void api.sharedPeople(token).then((r) => setPeople(r.people)).catch(() => setErr(true));
  }, [token]);

  if (err) return <div className="shared-board"><h1>Not found</h1><p>This link is revoked or wrong.</p></div>;
  if (!people) return <div className="shared-board"><p>Loading…</p></div>;

  const occasions: Occasion[] = [];
  for (const p of people) {
    if (p.birthday) {
      const next = nextOccurrence(p.birthday);
      if (next) occasions.push({ name: p.name, label: "birthday", color: p.color ?? "", ...next });
    }
    for (const d of p.dates) {
      const next = nextOccurrence(d.date);
      if (next) occasions.push({ name: p.name, label: d.label, color: p.color ?? "", ...next });
    }
  }
  occasions.sort((a, b) => a.daysUntil - b.daysUntil);

  return (
    <div className="shared-board">
      <header className="shared-head">
        <h1>Special days</h1>
        <p className="shared-desc">Birthdays and dates worth remembering — live list, always current.</p>
        <p className="shared-desc">
          <a href={`/api/shared/people/${token}/calendar.ics`}>Subscribe in your calendar (ICS)</a>
        </p>
      </header>
      <ul className="link-list" style={{ maxWidth: 560, margin: "0 auto" }}>
        {occasions.length === 0 && <li className="link-row"><span className="row-title">Nothing on the calendar yet.</span></li>}
        {occasions.map((o, i) => (
          <li key={`${o.name}-${o.label}-${i}`} className={`link-row ${o.color ? `color-${o.color}` : ""}`}>
            <span className="link-mark">{o.label === "birthday" ? <Cake size={14} /> : <Heart size={14} />}</span>
            <span className="row-title">
              {o.name}
              <span className="link-domain">{o.label}{o.turning ? ` · turns ${o.turning}` : ""}</span>
            </span>
            <span className="meta-chip date">{formatDate(o.date)}</span>
            <span className={`meta-chip ${o.daysUntil <= 7 ? "soon" : ""}`}>{daysLabel(o.daysUntil)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

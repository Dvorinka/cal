import type { Person, PersonInput } from "@cal/api-client";
import { addDays, fromIso, iso, todayIso } from "./date";

/** personToInput copies a stored person into the dialog/save input shape —
 *  the API replaces the whole record, so partial edits send everything. */
export function personToInput(p: Person): PersonInput {
  return {
    name: p.name,
    nickname: p.nickname ?? "",
    relation: p.relation,
    birthday: p.birthday ?? "",
    birthdayRemind: p.birthdayRemind ?? null,
    dates: p.dates ?? [],
    notes: p.notes,
    color: p.color,
    workspaceId: p.workspaceId ?? "",
    avatar: p.avatar ?? "",
    phone: p.phone ?? "",
    email: p.email ?? "",
    address: p.address ?? "",
    giftIdeas: p.giftIdeas ?? "",
    interests: p.interests ?? "",
    isFavorite: p.isFavorite,
    fields: p.fields ?? [],
    links: p.links ?? [],
    tags: p.tags ?? [],
  };
}

/** A person date rendered on a calendar day — a birthday or a named yearly
 *  date (anniversary, nameday…), expanded into a concrete occurrence. */
export interface PersonOccurrence {
  /** person.id + date key + occurrence date — stable per render. */
  id: string;
  personId: string;
  name: string;
  /** "birthday" or the custom date's label. */
  label: string;
  date: string; // occurrence YYYY-MM-DD inside the requested range
  color: string;
  /** Years since the stored date, when its year is known and past. */
  turning?: number;
}

// yearlyOn maps a stored YYYY-MM-DD onto `year`, keeping month+day.
// Feb 29 falls back to Feb 28 on non-leap years.
function yearlyOn(year: number, month: number, day: number): string {
  const d = new Date(year, month - 1, day);
  if (d.getMonth() !== month - 1) return iso(new Date(year, month - 1, day - 1));
  return iso(d);
}

/** personDatesInRange expands every person's birthday + named dates into
 *  occurrences within [from, to] (YYYY-MM-DD, inclusive). */
export function personDatesInRange(people: Person[], from: string, to: string): PersonOccurrence[] {
  const fromYear = fromIso(from).getFullYear();
  const toYear = fromIso(to).getFullYear();
  const out: PersonOccurrence[] = [];
  for (const p of people) {
    const named: { key: string; label: string; date: string }[] = [];
    if (p.birthday) named.push({ key: "birthday", label: "birthday", date: p.birthday });
    p.dates?.forEach((d, i) => named.push({ key: `date${i}`, label: d.label, date: d.date }));
    for (const item of named) {
      const [y0, month, day] = item.date.split("-").map(Number);
      if (!month || !day) continue;
      for (let year = fromYear; year <= toYear; year++) {
        const date = yearlyOn(year, month, day);
        if (date < from || date > to) continue;
        out.push({
          id: `${p.id}:${item.key}:${date}`,
          personId: p.id,
          name: p.name,
          label: item.label,
          date,
          color: p.color || "slate",
          turning: y0 && y0 < year ? year - y0 : undefined,
        });
      }
    }
  }
  return out.sort((a, b) => a.date.localeCompare(b.date));
}

export interface UpcomingDate extends PersonOccurrence {
  daysUntil: number;
}

/** upcomingDates returns occurrences in the next `days` days, soonest first. */
export function upcomingDates(people: Person[], days = 30): UpcomingDate[] {
  const today = todayIso();
  const to = iso(addDays(fromIso(today), days));
  const midnight = fromIso(today).getTime();
  return personDatesInRange(people, today, to).map((o) => ({
    ...o,
    daysUntil: Math.round((fromIso(o.date).getTime() - midnight) / 86400000),
  }));
}

/** nextByPerson maps personId → their soonest occurrence within `days`. */
export function nextByPerson(people: Person[], days = 366): Map<string, UpcomingDate> {
  const map = new Map<string, UpcomingDate>();
  for (const o of upcomingDates(people, days)) {
    if (!map.has(o.personId)) map.set(o.personId, o);
  }
  return map;
}

export function daysLabel(daysUntil: number): string {
  if (daysUntil <= 0) return "today";
  if (daysUntil === 1) return "tomorrow";
  return `in ${daysUntil}d`;
}

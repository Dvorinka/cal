export type CalendarView = "month" | "week" | "day";

export function iso(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function todayIso(): string {
  return iso(new Date());
}

export function addDays(date: Date, days: number): Date {
  const next = new Date(date);
  next.setDate(next.getDate() + days);
  return next;
}

export function startOfWeek(date: Date): Date {
  const next = new Date(date);
  const day = (next.getDay() + 6) % 7;
  next.setDate(next.getDate() - day);
  next.setHours(0, 0, 0, 0);
  return next;
}

export function monthMatrix(anchor: Date): Date[] {
  const first = new Date(anchor.getFullYear(), anchor.getMonth(), 1);
  const start = startOfWeek(first);
  return Array.from({ length: 42 }, (_, index) => addDays(start, index));
}

export function weekDays(anchor: Date): Date[] {
  const start = startOfWeek(anchor);
  return Array.from({ length: 7 }, (_, index) => addDays(start, index));
}

export function rangeFor(view: CalendarView, anchor: Date): { from: string; to: string } {
  if (view === "day") return { from: iso(anchor), to: iso(anchor) };
  const days = view === "week" ? weekDays(anchor) : monthMatrix(anchor);
  return { from: iso(days[0]), to: iso(days[days.length - 1]) };
}

export function formatHeading(view: CalendarView, anchor: Date): string {
  if (view === "day") return anchor.toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });
  if (view === "week") {
    const days = weekDays(anchor);
    return `${days[0].toLocaleDateString(undefined, { month: "short", day: "numeric" })} - ${days[6].toLocaleDateString(undefined, { month: "short", day: "numeric" })}`;
  }
  return anchor.toLocaleDateString(undefined, { month: "long", year: "numeric" });
}

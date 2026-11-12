import type { Entry, Habit } from "@cal/api-client";

/** Open occurrence a habit can be checked off: same title, tagged habit, not done, dated today or earlier. Latest date wins. */
export function openHabitOccurrence(entries: Entry[], habit: Pick<Habit, "title">, today: string): Entry | undefined {
  let best: Entry | undefined;
  for (const entry of entries) {
    if (entry.type !== "task" || entry.completed || entry.title !== habit.title) continue;
    if (!entry.tags.includes("habit")) continue;
    if (entry.date > today) continue;
    if (!best || entry.date > best.date) best = entry;
  }
  return best;
}

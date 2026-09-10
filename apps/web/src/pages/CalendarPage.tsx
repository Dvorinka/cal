import { useEffect, useMemo } from "react";
import { MonthView } from "../components/MonthView";
import { TimeGridView } from "../components/TimeGridView";
import { TopBar } from "../components/TopBar";
import { rangeFor } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function CalendarPage() {
  const { user, entries, holidays, settings, loadEntries, updateEntry, loadHolidays } = usePlanner();
  const { view, anchor, hiddenTypes } = useUi();

  const range = useMemo(() => rangeFor(view, anchor, settings.weekStart), [view, anchor, settings.weekStart]);

  useEffect(() => {
    if (!user) return;
    void loadEntries(range);
  }, [user, range.from, range.to, loadEntries]);

  useEffect(() => {
    if (!user || !settings.showHolidays) return;
    void loadHolidays(settings.country, anchor.getFullYear());
  }, [user, settings.country, settings.showHolidays, anchor, loadHolidays]);

  const visible = entries.filter((entry) => !hiddenTypes.has(entry.type));
  const shownHolidays = settings.showHolidays ? holidays : [];

  return (
    <>
      <TopBar weekStart={settings.weekStart} />
      {view === "month" ? (
        <MonthView
          anchor={anchor}
          entries={visible}
          holidays={shownHolidays}
          weekStart={settings.weekStart}
          onMoveEntry={(id, date) => void updateEntry(id, { date })}
        />
      ) : (
        <TimeGridView
          view={view}
          anchor={anchor}
          entries={visible}
          holidays={shownHolidays}
          weekStart={settings.weekStart}
          onMoveEntry={(id, patch) => void updateEntry(id, patch)}
        />
      )}
    </>
  );
}

import type { Entry, Holiday } from "@cal/api-client";
import { Plus } from "lucide-react";
import { useMemo, useState } from "react";
import { draggedEntryId, dropTargetProps } from "../lib/dnd";
import { iso, monthMatrix, timeToMinutes, todayIso, weekdayNames, type WeekStartPref } from "../lib/date";
import { useUi } from "../stores/ui";
import { EntryChip } from "./EntryChip";

interface Props {
  anchor: Date;
  entries: Entry[];
  holidays: Holiday[];
  weekStart: WeekStartPref;
  onMoveEntry: (id: string, date: string) => void;
}

const MAX_VISIBLE = 4;

export function MonthView({ anchor, entries, holidays, weekStart, onMoveEntry }: Props) {
  const selectDate = useUi((state) => state.selectDate);
  const selectedDate = useUi((state) => state.selectedDate);
  const openCreate = useUi((state) => state.openCreate);
  const setView = useUi((state) => state.setView);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const today = todayIso();

  const days = useMemo(() => monthMatrix(anchor, weekStart), [anchor, weekStart]);
  const names = useMemo(() => weekdayNames(weekStart), [weekStart]);

  const byDate = useMemo(() => {
    const map = new Map<string, Entry[]>();
    for (const entry of entries) {
      const list = map.get(entry.date) ?? [];
      list.push(entry);
      map.set(entry.date, list);
    }
    for (const list of map.values()) {
      list.sort((a, b) => (timeToMinutes(a.startTime) ?? 9999) - (timeToMinutes(b.startTime) ?? 9999) || a.createdAt.localeCompare(b.createdAt));
    }
    return map;
  }, [entries]);

  const holidayByDate = useMemo(() => {
    const map = new Map<string, string>();
    for (const h of holidays) map.set(h.date, map.has(h.date) ? `${map.get(h.date)}, ${h.name}` : h.name);
    return map;
  }, [holidays]);

  function toggleExpanded(date: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(date)) next.delete(date);
      else next.add(date);
      return next;
    });
  }

  return (
    <div className="month-wrap">
      <div className="dow-row">
        {names.map((name) => (
          <span key={name}>{name}</span>
        ))}
      </div>
      <div className="month-grid">
        {days.map((day) => {
          const date = iso(day);
          const dayEntries = byDate.get(date) ?? [];
          const isOutside = day.getMonth() !== anchor.getMonth();
          const isExpanded = expanded.has(date);
          const visible = isExpanded ? dayEntries : dayEntries.slice(0, MAX_VISIBLE);
          const hidden = dayEntries.length - visible.length;
          return (
            <div
              key={date}
              className={`day-cell ${date === today ? "today" : ""} ${date === selectedDate ? "selected" : ""} ${isOutside ? "outside" : ""}`}
              onClick={() => selectDate(date)}
              onDoubleClick={() => openCreate(date)}
              {...dropTargetProps((event) => {
                const id = draggedEntryId(event);
                if (id) onMoveEntry(id, date);
              })}
            >
              <div className="day-head">
                <span className="day-num">{day.getDate()}</span>
                {holidayByDate.has(date) && <span className="holiday-label">{holidayByDate.get(date)}</span>}
                <button
                  type="button"
                  className="icon-btn day-add"
                  aria-label={`Add entry on ${date}`}
                  onClick={(event) => {
                    event.stopPropagation();
                    openCreate(date);
                  }}
                >
                  <Plus size={13} />
                </button>
              </div>
              {visible.map((entry) => (
                <EntryChip key={entry.id} entry={entry} />
              ))}
              {hidden > 0 && (
                <button
                  type="button"
                  className="more-link"
                  onClick={(event) => {
                    event.stopPropagation();
                    toggleExpanded(date);
                    selectDate(date);
                  }}
                >
                  +{hidden} more
                </button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

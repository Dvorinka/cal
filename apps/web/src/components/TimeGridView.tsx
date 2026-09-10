import type { Entry, Holiday } from "@cal/api-client";
import { useEffect, useMemo, useRef, useState } from "react";
import { draggedEntryId, dropTargetProps, startEntryDrag } from "../lib/dnd";
import {
  formatTime,
  iso,
  minutesToTime,
  timeToMinutes,
  todayIso,
  weekDays,
  type CalendarView,
  type WeekStartPref,
} from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";
import { EntryChip } from "./EntryChip";

const HOUR_H = 52;
const HOURS = Array.from({ length: 24 }, (_, i) => i);

interface Props {
  view: CalendarView;
  anchor: Date;
  entries: Entry[];
  holidays: Holiday[];
  weekStart: WeekStartPref;
  onMoveEntry: (id: string, patch: { date: string; startTime?: string; endTime?: string }) => void;
}

/** Assigns overlapping timed events to lanes so they render side by side. */
function lanes(events: { entry: Entry; start: number; end: number }[]) {
  const sorted = [...events].sort((a, b) => a.start - b.start || b.end - a.end);
  const result: { entry: Entry; start: number; end: number; lane: number; lanes: number }[] = [];
  let group: typeof sorted = [];
  let groupEnd = -1;
  const flush = () => {
    const laneEnds: number[] = [];
    for (const item of group) {
      let lane = laneEnds.findIndex((end) => end <= item.start);
      if (lane === -1) {
        lane = laneEnds.length;
        laneEnds.push(item.end);
      } else {
        laneEnds[lane] = item.end;
      }
      result.push({ ...item, lane, lanes: 0 });
    }
    const lanes = laneEnds.length;
    for (let i = result.length - group.length; i < result.length; i++) result[i].lanes = lanes;
    group = [];
    groupEnd = -1;
  };
  for (const item of sorted) {
    if (item.start >= groupEnd && group.length) flush();
    group.push(item);
    groupEnd = Math.max(groupEnd, item.end);
  }
  if (group.length) flush();
  return result;
}

export function TimeGridView({ view, anchor, entries, holidays, weekStart, onMoveEntry }: Props) {
  const selectDate = useUi((state) => state.selectDate);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const setContextMenu = useUi((state) => state.setContextMenu);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [now, setNow] = useState(() => new Date());
  const today = todayIso();

  const days = useMemo(() => (view === "day" ? [anchor] : weekDays(anchor, weekStart)), [view, anchor, weekStart]);
  const cols = days.length;
  const dates = days.map(iso);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = Math.max(0, (now.getHours() - 2) * HOUR_H);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view, anchor]);

  const timed = useMemo(() => entries.filter((entry) => timeToMinutes(entry.startTime) !== undefined), [entries]);
  const untimed = useMemo(() => entries.filter((entry) => timeToMinutes(entry.startTime) === undefined), [entries]);
  const nowMinutes = now.getHours() * 60 + now.getMinutes();

  return (
    <div className="timegrid" style={{ "--cols": cols, "--hour-h": `${HOUR_H}px` } as React.CSSProperties}>
      <div className="tg-header">
        <div className="tg-corner" />
        {days.map((day, i) => (
          <div
            key={dates[i]}
            className={`tg-day-head ${dates[i] === today ? "today" : ""}`}
            onClick={() => selectDate(dates[i])}
            role="button"
            tabIndex={0}
          >
            <span className="dow">{day.toLocaleDateString(undefined, { weekday: "short" })}</span>
            <span className="num">{day.getDate()}</span>
          </div>
        ))}
      </div>

      <div className="tg-allday">
        <div className="tg-allday-label">All day</div>
        {days.map((day, i) => {
          const date = dates[i];
          const dayHolidays = holidays.filter((h) => h.date === date);
          return (
            <div
              key={date}
              className="tg-allday-cell"
              {...dropTargetProps((event) => {
                const id = draggedEntryId(event);
                if (id) onMoveEntry(id, { date, startTime: "", endTime: "" });
              })}
            >
              {dayHolidays.map((h) => (
                <div key={h.id} className="tg-holiday">
                  {h.name}
                </div>
              ))}
              {untimed
                .filter((entry) => entry.date === date)
                .map((entry) => (
                  <EntryChip key={entry.id} entry={entry} showTime={false} />
                ))}
            </div>
          );
        })}
      </div>

      <div className="tg-scroll" ref={scrollRef}>
        <div className="tg-body">
          <div className="tg-gutter">
            {HOURS.map((hour) => (
              <span key={hour} className="tg-hour-label" style={{ top: hour * HOUR_H }}>
                {hour === 0 ? "" : minutesToTime(hour * 60)}
              </span>
            ))}
          </div>
          {days.map((day, i) => {
            const date = dates[i];
            const dayEvents = lanes(
              timed
                .filter((entry) => entry.date === date)
                .map((entry) => {
                  const start = timeToMinutes(entry.startTime) ?? 0;
                  const end = timeToMinutes(entry.endTime) ?? start + 60;
                  return { entry, start, end: Math.max(end, start + 25) };
                }),
            );
            return (
              <div
                key={date}
                className={`tg-col ${date === today ? "today" : ""}`}
                onClick={(event) => {
                  const rect = event.currentTarget.getBoundingClientRect();
                  const minutes = Math.round(((event.clientY - rect.top) / HOUR_H) * 60 / 30) * 30;
                  selectDate(date);
                  openCreate(date, minutesToTime(Math.max(0, Math.min(1410, minutes))));
                }}
                {...dropTargetProps((event) => {
                  const id = draggedEntryId(event);
                  if (!id) return;
                  const rect = event.currentTarget.getBoundingClientRect();
                  const minutes = Math.round(((event.clientY - rect.top) / HOUR_H) * 60 / 30) * 30;
                  const start = minutesToTime(Math.max(0, Math.min(1410, minutes)));
                  const moved = entries.find((e) => e.id === id);
                  const duration =
                    moved && moved.startTime && moved.endTime
                      ? (timeToMinutes(moved.endTime) ?? 60) - (timeToMinutes(moved.startTime) ?? 0)
                      : 60;
                  const end = minutesToTime(Math.min(1439, Math.max(0, Math.min(1410, minutes)) + duration));
                  onMoveEntry(id, { date, startTime: start, endTime: end });
                })}
              >
                {HOURS.map((hour) => (
                  <div key={hour} className="tg-hour-line" style={{ top: hour * HOUR_H }} />
                ))}
                {dayEvents.map(({ entry, start, end, lane, lanes: laneCount }) => (
                  <div
                    key={entry.id}
                    className={`tg-event color-${entry.color} ${entry.completed ? "done" : ""}`}
                    style={{
                      top: (start / 60) * HOUR_H,
                      height: Math.max(20, ((end - start) / 60) * HOUR_H - 2),
                      left: `calc(${(lane / laneCount) * 100}% + 3px)`,
                      right: `calc(${((laneCount - lane - 1) / laneCount) * 100}% + 3px)`,
                    }}
                    draggable
                    onDragStart={(event) => {
                      event.stopPropagation();
                      startEntryDrag(event, entry.id);
                      event.currentTarget.classList.add("dragging");
                    }}
                    onDragEnd={(event) => event.currentTarget.classList.remove("dragging")}
                    onClick={(event) => {
                      event.stopPropagation();
                      openEdit(entry);
                    }}
                    onContextMenu={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      setContextMenu({ x: event.clientX, y: event.clientY, entry });
                    }}
                    title={entry.title}
                  >
                    <div className="t">{entry.title}</div>
                    {(end - start) * (HOUR_H / 60) > 34 && (
                      <div className="time">
                        {formatTime(entry.startTime!)} – {formatTime(entry.endTime ?? minutesToTime(start + 60))}
                      </div>
                    )}
                  </div>
                ))}
                {date === today && <div className="tg-now" style={{ top: (nowMinutes / 60) * HOUR_H }} />}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

import type { Entry, Holiday } from "@cal/api-client";
import { motion } from "framer-motion";
import { iso, monthMatrix, type CalendarView, weekDays } from "../lib/date";
import { EntryCard } from "./EntryCard";

interface Props {
  view: CalendarView;
  anchor: Date;
  entries: Entry[];
  holidays: Holiday[];
  selectedDate: string;
  onSelectDate: (date: string) => void;
  onMoveEntry: (id: string, date: string) => void;
  onToggleEntry: (id: string, completed: boolean) => void;
  onDeleteEntry: (id: string) => void;
}

export function CalendarGrid(props: Props) {
  const days = props.view === "day" ? [props.anchor] : props.view === "week" ? weekDays(props.anchor) : monthMatrix(props.anchor);
  const dragged = { current: "" };

  return (
    <div className={`calendar-grid ${props.view}`}>
      {days.map((day) => {
        const date = iso(day);
        const dayEntries = props.entries.filter((entry) => entry.date === date);
        const dayHolidays = props.holidays.filter((holiday) => holiday.date === date);
        const outsideMonth = props.view === "month" && day.getMonth() !== props.anchor.getMonth();
        return (
          <motion.section
            layout
            key={date}
            className={`day-cell ${props.selectedDate === date ? "selected" : ""} ${outsideMonth ? "muted" : ""}`}
            onClick={() => props.onSelectDate(date)}
            onDragOver={(event) => event.preventDefault()}
            onDrop={() => dragged.current && props.onMoveEntry(dragged.current, date)}
          >
            <header>
              <span>{day.toLocaleDateString(undefined, { weekday: "short" })}</span>
              <strong>{day.getDate()}</strong>
            </header>
            <div className="holiday-stack">
              {dayHolidays.map((holiday) => (
                <p key={holiday.id} className="holiday">
                  {holiday.name}
                </p>
              ))}
            </div>
            <div className="entry-stack">
              {dayEntries.map((entry) => (
                <EntryCard
                  key={entry.id}
                  entry={entry}
                  onToggle={props.onToggleEntry}
                  onDelete={props.onDeleteEntry}
                  onDragStart={() => {
                    dragged.current = entry.id;
                  }}
                />
              ))}
            </div>
          </motion.section>
        );
      })}
    </div>
  );
}

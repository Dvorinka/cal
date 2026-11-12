import type { WeekReview } from "@cal/api-client";
import { NotebookPen } from "lucide-react";
import { useEffect, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { addDays, formatDayShort, fromIso, iso, todayIso } from "../lib/date";
import { reportErr, usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

function rangeLabel(from: string, to: string): string {
  const fmt = { month: "short", day: "numeric" } as const;
  return `${fromIso(from).toLocaleDateString(undefined, fmt)} – ${fromIso(to).toLocaleDateString(undefined, fmt)}`;
}

function daysBetween(from: string, to: string): string[] {
  const out: string[] = [];
  let cursor = new Date(`${from}T12:00:00`);
  const end = new Date(`${to}T12:00:00`);
  while (cursor <= end && out.length < 14) {
    out.push(iso(cursor));
    cursor = addDays(cursor, 1);
  }
  return out;
}

export function ReviewPage() {
  const api = usePlanner((state) => state.api);
  const createEntry = usePlanner((state) => state.createEntry);
  const toast = usePlanner((state) => state.toast);
  const openEdit = useUi((state) => state.openEdit);
  const [review, setReview] = useState<WeekReview | null>(null);
  const today = todayIso();

  useEffect(() => {
    void api.weeklyReview().then(setReview).catch(reportErr("Could not load review"));
  }, [api]);

  async function saveAsNote() {
    if (!review) return;
    const lines = [
      `## Week of ${review.from}`,
      "",
      `- **${review.tasksDone}** tasks done, **${review.tasksSlipped}** slipped`,
      `- **${review.notesWritten}** notes written`,
      `- Completion streak: **${review.streak}** day${review.streak === 1 ? "" : "s"}`,
      review.busiestDay ? `- Busiest: ${formatDayShort(review.busiestDay)} (${review.busiestCount} done)` : "",
      "",
      "## Three focuses for next week",
      "",
      "1. ",
      "2. ",
      "3. ",
    ].join("\n");
    const entry = await createEntry({ title: `Weekly review ${review.from}`, type: "note", date: today, content: lines });
    if (entry) {
      toast("Review saved as a note");
      openEdit(entry);
    }
  }

  const days = review ? daysBetween(review.from, review.to) : [];
  const peak = days.reduce((max, day) => Math.max(max, review?.perDay[day] ?? 0), 0);

  return (
    <>
      <PageHeader title="Weekly review" sub={review ? rangeLabel(review.from, review.to) : "Last 7 days"}>
        <button type="button" className="btn btn-secondary" disabled={!review} onClick={() => void saveAsNote()}>
          <NotebookPen size={14} /> Save as note
        </button>
      </PageHeader>
      <div className="page-scroll review-page" data-testid="review-page">
        {!review && <p className="panel-empty">Loading the last 7 days…</p>}
        {review && (
          <>
            <div className="review-grid">
              <div className="review-stat"><b>{review.tasksDone}</b><span>done</span></div>
              <div className="review-stat"><b>{review.tasksSlipped}</b><span>slipped</span></div>
              <div className="review-stat"><b>{review.notesWritten}</b><span>notes</span></div>
              <div className="review-stat"><b>{review.streak}</b><span>day streak</span></div>
            </div>
            <section className="panel">
              <h3>Completed by day</h3>
              <ul className="review-bars">
                {days.map((day) => {
                  const count = review.perDay[day] ?? 0;
                  const width = peak === 0 ? 0 : Math.round((count / peak) * 100);
                  const label = new Date(`${day}T12:00:00`).toLocaleDateString(undefined, { weekday: "short", month: "short", day: "numeric" });
                  return (
                    <li key={day}>
                      <span>{label}</span>
                      <i aria-hidden><b style={{ width: `${width}%` }} /></i>
                      <em>{count}</em>
                    </li>
                  );
                })}
              </ul>
              {review.busiestDay && (
                <p className="panel-note">Busiest day: {formatDayShort(review.busiestDay)}, {review.busiestCount} completed.</p>
              )}
            </section>
          </>
        )}
      </div>
    </>
  );
}

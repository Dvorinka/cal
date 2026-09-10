// TimerPill — solidtime-style running clock. One timer per user server-side;
// this polls for it and ticks locally. Start from a task's context menu.

import { Pause, Timer } from "lucide-react";
import { useEffect, useState } from "react";
import { usePlanner } from "../stores/planner";

function fmtElapsed(since: string): string {
  const s = Math.max(0, Math.floor((Date.now() - new Date(since).getTime()) / 1000));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  return h > 0
    ? `${h}:${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`
    : `${m}:${String(sec).padStart(2, "0")}`;
}

export function TimerPill() {
  const api = usePlanner((s) => s.api);
  const [timer, setTimer] = useState<{ title: string; startAt: string } | null>(null);
  const [, tick] = useState(0);

  useEffect(() => {
    const load = () => void api.currentTimer().then(setTimer).catch(() => setTimer(null));
    load();
    const poll = window.setInterval(load, 30_000);
    const tock = window.setInterval(() => tick((n) => n + 1), 1_000);
    return () => {
      window.clearInterval(poll);
      window.clearInterval(tock);
    };
  }, [api]);

  if (!timer) return null;

  return (
    <div className="timer-pill" role="status">
      <Timer size={12} className="timer-icon" />
      <span className="timer-title">{timer.title || "Focus"}</span>
      <span className="timer-clock">{fmtElapsed(timer.startAt)}</span>
      <button
        type="button"
        className="timer-stop"
        aria-label="Stop timer"
        onClick={() => void api.stopTimer().then(() => setTimer(null))}
      >
        <Pause size={11} />
      </button>
    </div>
  );
}

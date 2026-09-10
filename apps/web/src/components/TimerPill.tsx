// TimerPill — solidtime-style running clock. One timer per user server-side;
// this polls for it and ticks locally. With `planned` minutes set it becomes a
// pomodoro countdown and fires a notification at zero.

import { Pause, Timer } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { usePlanner } from "../stores/planner";

function fmtElapsed(since: string, planned?: number): string {
  const s = Math.floor((Date.now() - new Date(since).getTime()) / 1000);
  const left = planned ? planned * 60 - s : s;
  const abs = Math.abs(Math.max(planned ? left : s, 0));
  const h = Math.floor(abs / 3600);
  const m = Math.floor((abs % 3600) / 60);
  const sec = abs % 60;
  const t = h > 0
    ? `${h}:${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`
    : `${m}:${String(sec).padStart(2, "0")}`;
  return planned && left < 0 ? `+${t}` : t;
}

export function TimerPill() {
  const api = usePlanner((s) => s.api);
  const [timer, setTimer] = useState<{ title: string; startAt: string; planned?: number } | null>(null);
  const [, tick] = useState(0);
  const fired = useRef(false);

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

  // Pomodoro done → notify once.
  useEffect(() => {
    if (!timer?.planned || fired.current) return;
    const left = timer.planned * 60 * 1000 - (Date.now() - new Date(timer.startAt).getTime());
    if (left <= 0) {
      fired.current = true;
      if (Notification.permission === "granted") {
        new Notification("Focus done", { body: `${timer.planned}m on ${timer.title || "the task"}`, tag: "pomodoro" });
      }
    }
  }, [timer]);

  if (!timer) return null;

  return (
    <div className="timer-pill" role="status">
      <Timer size={12} className="timer-icon" />
      <span className="timer-title">{timer.title || "Focus"}</span>
      <span className="timer-clock">{fmtElapsed(timer.startAt, timer.planned)}</span>
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

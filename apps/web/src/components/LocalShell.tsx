import { CalendarDays, PlugZap } from "lucide-react";
import { MailPage } from "../pages/MailPage";
import { usePlanner } from "../stores/planner";
import { Toasts } from "./Toasts";

// LocalShell — the whole app in local mode. No sidebar, no calendar, no
// server: just Mail talking straight to the provider via the CalMail plugin,
// plus the door back to a real server when one exists.
export function LocalShell() {
  const exitLocal = usePlanner((s) => s.exitLocal);
  return (
    <main className="app-shell local-shell">
      <section className="workspace">
        <MailPage />
        <footer className="local-foot">
          <span className="local-chip">
            <CalendarDays size={12} /> Local mode — nothing leaves this device except your mail
          </span>
          <button type="button" className="btn btn-secondary btn-xs" onClick={exitLocal}>
            <PlugZap size={12} /> Connect a server
          </button>
        </footer>
      </section>
      <Toasts />
    </main>
  );
}

import { CalendarDays } from "lucide-react";
import { useState } from "react";
import { usePlanner } from "../stores/planner";

// Capacitor webviews run on capacitor://localhost (iOS) or http://localhost
// (Android) — the API lives elsewhere, so native builds must ask for a server.
const isNative = typeof window !== "undefined" &&
  (window.location.protocol === "capacitor:" ||
    window.location.protocol === "ionic:" ||
    (window as { Capacitor?: { isNativePlatform?: () => boolean } }).Capacitor?.isNativePlatform?.() === true);

export function AuthPanel() {
  const login = usePlanner((state) => state.login);
  const register = usePlanner((state) => state.register);
  const api = usePlanner((state) => state.api);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [server, setServer] = useState(api.remote || "");
  const [showServer, setShowServer] = useState(isNative || !!api.remote);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (isNative && !server.trim()) {
      setError("Enter your Cal server address (e.g. https://cal.example.com)");
      return;
    }
    setBusy(true);
    try {
      const target = server.trim().replace(/\/+$/, "");
      if (mode === "login") await login(email, password, target);
      else await register(email, password, target);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Authentication failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-head">
          <div className="mark">
            <CalendarDays size={21} strokeWidth={2.2} />
          </div>
          <h1 id="auth-title">{mode === "login" ? "Sign in to Cal" : "Create your account"}</h1>
          <p>Your self-hosted planner. Tasks, notes, links and holidays in one quiet calendar.</p>
        </div>
        <form onSubmit={submit} className="auth-form">
          {showServer && (
            <label className="field">
              <span>Server</span>
              <input
                className="input"
                type="url"
                inputMode="url"
                autoCapitalize="none"
                autoCorrect="off"
                placeholder="https://cal.example.com"
                value={server}
                onChange={(event) => setServer(event.target.value)}
                required={isNative}
              />
            </label>
          )}
          <label className="field">
            <span>Email</span>
            <input
              className="input"
              type="email"
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </label>
          <label className="field">
            <span>Password</span>
            <input
              className="input"
              type="password"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              minLength={8}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          {error && <p className="auth-error">{error}</p>}
          <button type="submit" className="btn btn-primary" disabled={busy} style={{ minHeight: 36 }}>
            {mode === "login" ? "Sign in" : "Create account"}
          </button>
        </form>
        <p className="auth-swap">
          {mode === "login" ? "New here? " : "Already have an account? "}
          <button type="button" onClick={() => setMode(mode === "login" ? "register" : "login")}>
            {mode === "login" ? "Create an account" : "Sign in"}
          </button>
          {!isNative && !showServer && (
            <>
              {" · "}
              <button type="button" onClick={() => setShowServer(true)}>other server</button>
            </>
          )}
        </p>
      </section>
    </main>
  );
}

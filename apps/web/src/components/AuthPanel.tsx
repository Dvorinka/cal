import { CalendarDays } from "lucide-react";
import { useState } from "react";
import { canUseLocal } from "../lib/local";
import { usePlanner } from "../stores/planner";

// Capacitor webviews run on capacitor://localhost (iOS) or http://localhost
// (Android) — the API lives elsewhere, so native builds must ask for a server.
const isNative = typeof window !== "undefined" &&
  (window.location.protocol === "capacitor:" ||
    window.location.protocol === "ionic:" ||
    (window as { Capacitor?: { isNativePlatform?: () => boolean } }).Capacitor?.isNativePlatform?.() === true);

// Password-reset links mail to /reset-password?token=… — the shell renders
// AuthPanel for every unauthenticated path, so the token is picked up here.
const resetToken = (() => {
  if (typeof window === "undefined" || window.location.pathname !== "/reset-password") return "";
  return new URLSearchParams(window.location.search).get("token") ?? "";
})();

type Mode = "login" | "register" | "forgot" | "reset";

export function AuthPanel() {
  const login = usePlanner((state) => state.login);
  const register = usePlanner((state) => state.register);
  const enterLocal = usePlanner((state) => state.enterLocal);
  const api = usePlanner((state) => state.api);
  const [mode, setMode] = useState<Mode>(resetToken ? "reset" : "login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [server, setServer] = useState(api.remote || "");
  const [showServer, setShowServer] = useState(isNative || !!api.remote);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);

  function applyServer() {
    const target = server.trim().replace(/\/+$/, "");
    if (target && target !== api.remote) api.setServer(target);
    return target;
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setNotice("");
    if (isNative && !server.trim()) {
      setError("Enter your Cal server address (e.g. https://cal.example.com)");
      return;
    }
    setBusy(true);
    try {
      const target = applyServer();
      if (mode === "forgot") {
        await api.forgotPassword(email);
        // Server always 204s — never reveals whether the account exists.
        setNotice("If that account exists, a reset link is on its way. It works once and expires in one hour.");
      } else if (mode === "reset") {
        if (password !== confirm) {
          setError("Passwords don't match");
          return;
        }
        await api.resetPassword(resetToken, password);
        window.history.replaceState(null, "", "/");
        setMode("login");
        setNotice("Password updated — sign in with the new one.");
        setPassword("");
        setConfirm("");
      } else if (mode === "login") await login(email, password, target);
      else await register(email, password, target);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Authentication failed");
    } finally {
      setBusy(false);
    }
  }

  const title =
    mode === "login" ? "Sign in to Cal"
    : mode === "register" ? "Create your account"
    : mode === "forgot" ? "Reset your password"
    : "Choose a new password";

  return (
    <main className="auth-shell">
      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-head">
          <div className="mark">
            <CalendarDays size={21} strokeWidth={2.2} />
          </div>
          <h1 id="auth-title">{title}</h1>
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
          {mode !== "reset" && (
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
          )}
          {(mode === "login" || mode === "register" || mode === "reset") && (
            <label className="field">
              <span>{mode === "reset" ? "New password" : "Password"}</span>
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
          )}
          {mode === "reset" && (
            <label className="field">
              <span>Confirm password</span>
              <input
                className="input"
                type="password"
                autoComplete="new-password"
                minLength={8}
                value={confirm}
                onChange={(event) => setConfirm(event.target.value)}
                required
              />
            </label>
          )}
          {error && <p className="auth-error">{error}</p>}
          {notice && <p className="auth-notice">{notice}</p>}
          <button type="submit" className="btn btn-primary" disabled={busy} style={{ minHeight: 36 }}>
            {mode === "login" ? "Sign in"
            : mode === "register" ? "Create account"
            : mode === "forgot" ? "Send reset link"
            : "Set new password"}
          </button>
        </form>
        <p className="auth-swap">
          {mode === "forgot" || mode === "reset" ? (
            <button type="button" onClick={() => { setMode("login"); setError(""); }}>
              Back to sign in
            </button>
          ) : (
            <>
              {mode === "login" ? "New here? " : "Already have an account? "}
              <button type="button" onClick={() => setMode(mode === "login" ? "register" : "login")}>
                {mode === "login" ? "Create an account" : "Sign in"}
              </button>
              {mode === "login" && (
                <>
                  {" · "}
                  <button type="button" onClick={() => { setMode("forgot"); setError(""); }}>
                    Forgot password?
                  </button>
                </>
              )}
              {!isNative && !showServer && (
                <>
                  {" · "}
                  <button type="button" onClick={() => setShowServer(true)}>other server</button>
                </>
              )}
            </>
          )}
        </p>
        {canUseLocal() && mode === "login" && (
          <p className="auth-swap">
            No server?{" "}
            <button type="button" onClick={enterLocal}>
              Use on this device — mail only
            </button>
          </p>
        )}
      </section>
    </main>
  );
}

import { useState } from "react";
import { usePlanner } from "../stores/planner";

export function AuthPanel() {
  const login = usePlanner((state) => state.login);
  const register = usePlanner((state) => state.register);
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("demo@cal.local");
  const [password, setPassword] = useState("password123");
  const [error, setError] = useState("");

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    try {
      if (mode === "login") await login(email, password);
      else await register(email, password);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Authentication failed");
    }
  }

  return (
    <main className="auth-shell">
      <section className="auth-panel" aria-labelledby="auth-title">
        <div>
          <p className="eyebrow">Self-hosted planner</p>
          <h1 id="auth-title">Cal</h1>
          <p className="lede">Daily calendar, tasks, notes, links, holidays, offline cache.</p>
        </div>
        <form onSubmit={submit} className="auth-form">
          <label>
            Email
            <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" minLength={8} value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>
          {error && <p className="error">{error}</p>}
          <button type="submit">{mode === "login" ? "Log in" : "Create account"}</button>
        </form>
        <button className="ghost" type="button" onClick={() => setMode(mode === "login" ? "register" : "login")}>
          {mode === "login" ? "Need account?" : "Have account?"}
        </button>
      </section>
    </main>
  );
}

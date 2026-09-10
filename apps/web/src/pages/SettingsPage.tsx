import { Download, LogOut } from "lucide-react";
import { useEffect, useMemo } from "react";
import { PageHeader } from "../components/PageHeader";
import { usePlanner } from "../stores/planner";

export function SettingsPage() {
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const countries = usePlanner((state) => state.countries);
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const user = usePlanner((state) => state.user);
  const logout = usePlanner((state) => state.logout);

  useEffect(() => {
    void loadEntries({});
  }, [loadEntries]);

  const stats = useMemo(() => {
    const counts = { task: 0, note: 0, link: 0, done: 0 };
    for (const entry of entries) {
      if (entry.type === "task") {
        counts.task++;
        if (entry.completed) counts.done++;
      } else if (entry.type === "note") counts.note++;
      else counts.link++;
    }
    return counts;
  }, [entries]);

  const set = (patch: Partial<typeof settings>) => void updateSettings({ ...settings, ...patch });

  return (
    <>
      <PageHeader title="Settings" />
      <div className="page-scroll settings-page">
        <section className="panel">
          <h3>Appearance</h3>
          <div className="settings-grid">
            <label className="field">
              <span>Theme</span>
              <select
                className="select"
                value={settings.theme}
                onChange={(e) => set({ theme: e.target.value as typeof settings.theme })}
              >
                <option value="system">System</option>
                <option value="dark">Dark</option>
                <option value="light">Light</option>
              </select>
            </label>
            <label className="field">
              <span>Week starts on</span>
              <select
                className="select"
                value={settings.weekStart}
                onChange={(e) => set({ weekStart: e.target.value as typeof settings.weekStart })}
              >
                <option value="monday">Monday</option>
                <option value="sunday">Sunday</option>
              </select>
            </label>
          </div>
        </section>

        <section className="panel">
          <h3>Holidays</h3>
          <div className="settings-grid">
            <label className="switch-row">
              Show holidays
              <span className="switch">
                <input
                  type="checkbox"
                  checked={settings.showHolidays}
                  onChange={(e) => set({ showHolidays: e.target.checked })}
                />
                <i />
              </span>
            </label>
            <label className="field">
              <span>Region</span>
              <select className="select" value={settings.country} onChange={(e) => set({ country: e.target.value })}>
                {countries.length === 0 && <option value={settings.country}>{settings.country}</option>}
                {countries.map((country) => (
                  <option key={country.code} value={country.code}>
                    {country.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </section>

        <section className="panel">
          <h3>Your data</h3>
          <p className="panel-note">
            {stats.task} tasks ({stats.done} done), {stats.note} notes, {stats.link} links — stored in your own
            database. Take them with you any time.
          </p>
          <a className="btn btn-secondary" href="/api/export" download>
            <Download size={14} /> Export everything (JSON)
          </a>
        </section>

        <section className="panel">
          <h3>Account</h3>
          <p className="panel-note">{user?.email}</p>
          <button type="button" className="btn btn-secondary" onClick={() => void logout()}>
            <LogOut size={14} /> Sign out
          </button>
        </section>
      </div>
    </>
  );
}

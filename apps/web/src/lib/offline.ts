import type { Entry, Settings, User } from "@cal/api-client";

const prefix = "cal:";

export function readCachedEntries(): Entry[] {
  return read<Entry[]>("entries") ?? [];
}

export function cacheEntries(entries: Entry[]): void {
  write("entries", entries);
}

export function readCachedUser(): User | undefined {
  return read<User>("user");
}

export function cacheUser(user: User): void {
  write("user", user);
}

export function clearCachedUser(): void {
  try {
    localStorage.removeItem(prefix + "user");
  } catch {
    // no storage
  }
}

export function readCachedSettings(): Settings {
  return (
    read<Settings>("settings") ?? {
      country: "US",
      showHolidays: true,
      theme: "system",
      weekStart: "monday",
      accent: "green",
      widgetToken: "",
      apiToken: "",
    }
  );
}

export function cacheSettings(settings: Settings): void {
  write("settings", settings);
}

function read<T>(key: string): T | undefined {
  try {
    const raw = localStorage.getItem(prefix + key);
    return raw ? (JSON.parse(raw) as T) : undefined;
  } catch {
    return undefined;
  }
}

function write(key: string, value: unknown): void {
  try {
    localStorage.setItem(prefix + key, JSON.stringify(value));
  } catch {
    // no storage (tests, SSR, private mode)
  }
}

import type { Entry, Settings } from "@cal/api-client";

const prefix = "cal:";

export function readCachedEntries(): Entry[] {
  return read<Entry[]>("entries") ?? [];
}

export function cacheEntries(entries: Entry[]): void {
  write("entries", entries);
}

export function readCachedSettings(): Settings {
  return read<Settings>("settings") ?? { country: "US", showHolidays: true, theme: "system" };
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
  localStorage.setItem(prefix + key, JSON.stringify(value));
}

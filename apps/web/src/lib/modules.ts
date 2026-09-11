import type { Settings } from "@cal/api-client";

// Feature modules — each maps a settings.modules key to the routes and
// nav entries it gates. Calendar and Today are always on; everything
// else can be switched off in Settings → Modules.
export const MODULES = [
  { key: "tasks", label: "Tasks", hint: "Task lists, due dates, habits" },
  { key: "boards", label: "Boards", hint: "Kanban boards and cards" },
  { key: "notes", label: "Notes", hint: "Journal, notes and templates" },
  { key: "links", label: "Links", hint: "Bookmarks and YouTube saves" },
  { key: "files", label: "Files", hint: "Uploads, documents, attachments" },
  { key: "time", label: "Time tracking", hint: "Timers, timesheet, billing" },
  { key: "github", label: "GitHub", hint: "Issue/PR inbox and activity" },
  { key: "mail", label: "Mail", hint: "IMAP/SMTP inbox and compose" },
  { key: "tags", label: "Tags", hint: "Tag browser and counts" },
  { key: "people", label: "People", hint: "Birthdays, anniversaries, important dates" },
] as const;

export type ModuleKey = (typeof MODULES)[number]["key"];

/** Module is on unless explicitly disabled; missing settings = all on. */
export function moduleOn(settings: Settings | undefined, key: ModuleKey): boolean {
  return settings?.modules?.[key] !== false;
}

/** Route → module key. Used for access gating and nav filtering. */
export function moduleForPath(path: string): ModuleKey | null {
  if (path.startsWith("/tasks")) return "tasks";
  if (path.startsWith("/boards")) return "boards";
  if (path.startsWith("/notes")) return "notes";
  if (path.startsWith("/links")) return "links";
  if (path.startsWith("/files")) return "files";
  if (path.startsWith("/time")) return "time";
  if (path.startsWith("/github")) return "github";
  if (path.startsWith("/mail")) return "mail";
  if (path.startsWith("/tags")) return "tags";
  if (path.startsWith("/people")) return "people";
  return null;
}

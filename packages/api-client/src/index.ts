export type EntryType = "task" | "note" | "link" | "event";
export type Theme = "light" | "dark" | "system";
export type WeekStart = "monday" | "sunday";
export type Recur = "none" | "daily" | "weekly" | "monthly" | "yearly";
export type Accent = "green" | "blue" | "violet" | "amber" | "rose";

export interface User {
  id: string;
  email: string;
}

export interface Entry {
  id: string;
  title: string;
  content?: string;
  type: EntryType;
  linkUrl?: string;
  date: string;
  /** HH:MM 24h; absent means all-day / untimed */
  startTime?: string;
  endTime?: string;
  completed: boolean;
  color: string;
  tags: string[];
  recur: Recur;
  /** Minutes before startTime to fire a reminder; absent = none. */
  remind?: number;
  accountId?: string;
  createdAt: string;
}

export interface EntryInput {
  title: string;
  content?: string;
  type: EntryType;
  linkUrl?: string;
  date: string;
  startTime?: string;
  endTime?: string;
  color?: string;
  tags?: string[];
  recur?: Recur;
  remind?: number | null;
  accountId?: string;
}

export type EntryPatch = Partial<EntryInput & { completed: boolean }>;

export interface Revision {
  id: string;
  entryId: string;
  title: string;
  content: string;
  type: EntryType;
  date: string;
  startTime?: string;
  endTime?: string;
  completed: boolean;
  color: string;
  tags: string[];
  recur: Recur;
  remind?: number;
  savedAt: string;
}

export interface Settings {
  country: string;
  showHolidays: boolean;
  theme: Theme;
  weekStart: WeekStart;
  accent: Accent;
  timezone: string;
  widgetToken: string;
  apiToken: string;
}

export interface SessionInfo {
  id: string;
  userAgent?: string;
  createdAt: string;
  lastSeen?: string;
  current: boolean;
}

export interface CaldavAccount {
  id: string;
  name: string;
  url: string;
  username: string;
  color: string;
  lastSynced?: string;
}

export interface Feed {
  id: string;
  name: string;
  url: string;
  color: string;
  fetchedAt?: string;
}

export interface FeedEvent {
  id: string;
  feedId: string;
  feedName: string;
  title: string;
  date: string;
  startTime?: string;
  endTime?: string;
  color: string;
  location?: string;
  url?: string;
  details?: string;
}

export interface Holiday {
  id: string;
  name: string;
  date: string;
  country: string;
}

export interface Country {
  code: string;
  name: string;
}

export interface AuthRequest {
  email: string;
  password: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
  }
}

export class CalApi {
  constructor(private readonly baseUrl = "/api") {}

  async register(input: AuthRequest): Promise<User> {
    return this.request<User>("/auth/register", { method: "POST", body: input });
  }

  async login(input: AuthRequest): Promise<User> {
    return this.request<User>("/auth/login", { method: "POST", body: input });
  }

  async logout(): Promise<void> {
    await this.request<void>("/auth/logout", { method: "POST" });
  }

  async me(): Promise<User> {
    return this.request<User>("/me");
  }

  async entries(params: { from?: string; to?: string; q?: string } = {}): Promise<Entry[]> {
    const search = new URLSearchParams();
    if (params.from) search.set("from", params.from);
    if (params.to) search.set("to", params.to);
    if (params.q) search.set("q", params.q);
    const suffix = search.size ? `?${search.toString()}` : "";
    return this.request<Entry[]>(`/entries${suffix}`);
  }

  async createEntry(input: EntryInput): Promise<Entry> {
    return this.request<Entry>("/entries", { method: "POST", body: input });
  }

  async updateEntry(id: string, patch: EntryPatch): Promise<Entry> {
    return this.request<Entry>(`/entries/${id}`, { method: "PATCH", body: patch });
  }

  async deleteEntry(id: string): Promise<void> {
    await this.request<void>(`/entries/${id}`, { method: "DELETE" });
  }

  async settings(): Promise<Settings> {
    return this.request<Settings>("/settings");
  }

  async updateSettings(settings: Settings): Promise<Settings> {
    return this.request<Settings>("/settings", { method: "PUT", body: settings });
  }

  async holidays(country: string, year: number): Promise<Holiday[]> {
    return this.request<Holiday[]>(`/holidays?country=${encodeURIComponent(country)}&year=${year}`);
  }

  async countries(): Promise<Country[]> {
    return this.request<Country[]>("/holidays/countries");
  }

  async feeds(): Promise<Feed[]> {
    return this.request<Feed[]>("/feeds");
  }

  async createFeed(input: { name: string; url: string; color?: string }): Promise<Feed> {
    return this.request<Feed>("/feeds", { method: "POST", body: input });
  }

  async deleteFeed(id: string): Promise<void> {
    await this.request<void>(`/feeds/${id}`, { method: "DELETE" });
  }

  async refreshFeed(id: string): Promise<void> {
    await this.request<void>(`/feeds/${id}/refresh`, { method: "POST" });
  }

  async feedEvents(params: { from: string; to: string }): Promise<FeedEvent[]> {
    const search = new URLSearchParams({ from: params.from, to: params.to });
    return this.request<FeedEvent[]>(`/feed-events?${search.toString()}`);
  }

  async sessions(): Promise<SessionInfo[]> {
    return this.request<SessionInfo[]>("/sessions");
  }

  async revokeSession(id: string): Promise<void> {
    await this.request<void>(`/sessions/${id}`, { method: "DELETE" });
  }

  async changePassword(current: string, password: string): Promise<void> {
    await this.request<void>("/password", { method: "POST", body: { current, password } });
  }

  async caldavAccounts(): Promise<CaldavAccount[]> {
    return this.request<CaldavAccount[]>("/caldav");
  }

  async addCaldavAccount(input: { name?: string; url: string; username: string; password: string; color?: string }): Promise<CaldavAccount> {
    return this.request<CaldavAccount>("/caldav", { method: "POST", body: input });
  }

  async deleteCaldavAccount(id: string): Promise<void> {
    await this.request<void>(`/caldav/${id}`, { method: "DELETE" });
  }

  async syncCaldav(id: string): Promise<void> {
    await this.request<void>(`/caldav/${id}/sync`, { method: "POST" });
  }

  async entryRevisions(id: string): Promise<Revision[]> {
    return this.request<Revision[]>(`/entries/${id}/revisions`);
  }

  async restoreRevision(entryId: string, revId: string): Promise<void> {
    await this.request<void>(`/entries/${entryId}/restore/${revId}`, { method: "POST" });
  }

  async restore(file: File): Promise<{ restored: number }> {
    const response = await fetch(`${this.baseUrl}/restore`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: file,
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Request failed: ${response.status}`);
    }
    return response.json();
  }

  async importIcs(file: File): Promise<{ imported: number }> {
    const response = await fetch(`${this.baseUrl}/import`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "text/calendar" },
      body: file,
    });
    if (!response.ok) {
      const text = await response.text();
      throw new ApiError(text || response.statusText, response.status);
    }
    return response.json() as Promise<{ imported: number }>;
  }

  async rotateWidgetToken(): Promise<string> {
    const out = await this.request<{ widgetToken: string }>("/settings/widget-token", { method: "POST" });
    return out.widgetToken;
  }

  async pushVapid(): Promise<string> {
    const out = await this.request<{ publicKey: string }>("/push/vapid");
    return out.publicKey;
  }

  async pushSubscribe(sub: PushSubscriptionJSON): Promise<void> {
    await this.request<void>("/push/subscribe", { method: "POST", body: sub });
  }

  async pushUnsubscribe(endpoint: string): Promise<void> {
    await this.request<void>("/push/unsubscribe", { method: "POST", body: { endpoint } });
  }

  async rotateApiToken(): Promise<string> {
    const out = await this.request<{ apiToken: string }>("/settings/api-token", { method: "POST" });
    return out.apiToken;
  }

  private async request<T>(path: string, init: { method?: string; body?: unknown } = {}): Promise<T> {
    const response = await fetch(`${this.baseUrl}${path}`, {
      method: init.method ?? "GET",
      credentials: "include",
      headers: init.body ? { "Content-Type": "application/json" } : undefined,
      body: init.body ? JSON.stringify(init.body) : undefined,
    });

    if (!response.ok) {
      const text = await response.text();
      throw new ApiError(text || response.statusText, response.status);
    }

    if (response.status === 204) return undefined as T;
    return response.json() as Promise<T>;
  }
}

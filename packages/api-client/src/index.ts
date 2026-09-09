export type EntryType = "task" | "note" | "link";
export type Theme = "light" | "dark" | "system";

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
  completed: boolean;
  color: string;
  tags: string[];
  createdAt: string;
}

export interface EntryInput {
  title: string;
  content?: string;
  type: EntryType;
  linkUrl?: string;
  date: string;
  color?: string;
  tags?: string[];
}

export type EntryPatch = Partial<EntryInput & { completed: boolean }>;

export interface Settings {
  country: string;
  showHolidays: boolean;
  theme: Theme;
}

export interface Holiday {
  id: string;
  name: string;
  date: string;
  country: string;
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

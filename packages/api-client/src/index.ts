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
  pinned: boolean;
  watched?: boolean;
  linkImage?: string;
  linkDesc?: string;
  linkFavicon?: string;
  linkVideoId?: string;
  color: string;
  boardId?: string;
  columnId?: string;
  position?: number;
  tags: string[];
  recur: Recur;
  /** Minutes before startTime to fire a reminder; absent = none. */
  remind?: number;
  accountId?: string;
  workspaceId?: string;
  blockedBy?: string;
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
  boardId?: string;
  columnId?: string;
  workspaceId?: string;
  blockedBy?: string;
}

export type EntryPatch = Partial<EntryInput & { completed: boolean; pinned: boolean; watched: boolean }>;

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
  city: string;
  quotaMb?: number;
  digestTime?: string;
  defaultRate?: number;
  githubToken?: string;
  widgetToken: string;
  apiToken: string;
  /** Feature modules on/off; absent key = enabled. */
  modules?: Record<string, boolean>;
  /** Active workspace id; undefined/"none"/"" semantics handled client-side. */
  activeWorkspace?: string;
}

export interface WeekReview {
  from: string;
  to: string;
  tasksDone: number;
  tasksSlipped: number;
  notesWritten: number;
  streak: number;
  busiestDay: string;
  busiestCount: number;
  perDay: Record<string, number>;
}

export interface Habit {
  id: string;
  title: string;
  recur: string;
  streak: number;
  lastDone?: string;
}

export interface Webhook {
  id: string;
  url: string;
  secret: string;
  createdAt: string;
}

export interface Unfurl {
  title: string;
  favicon: string;
  description: string;
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

export interface FileRec {
  id: string;
  name: string;
  origName: string;
  size: number;
  mime: string;
  shareToken?: string;
  total: number;
  done: number;
  createdAt: string;
  tags: string[];
  workspaceId?: string;
}

export interface Feed {
  id: string;
  name: string;
  url: string;
  color: string;
  kind: string; // "calendar" | "links" — links feeds create bookmark entries
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
  image?: string;
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
  // Native apps (Capacitor) can't use cookie auth cross-origin — they store
  // a server URL + Bearer session. Same-origin web uses neither.
  private server = "";
  private session = "";

  constructor(private readonly baseUrl = "/api") {
    try {
      this.server = localStorage.getItem("cal:server") ?? "";
      this.session = localStorage.getItem("cal:session") ?? "";
    } catch {
      // storage unavailable
    }
  }

  get remote(): string {
    return this.server;
  }

  setServer(url: string) {
    this.server = url.replace(/\/+$/, "");
    try {
      if (this.server) localStorage.setItem("cal:server", this.server);
      else localStorage.removeItem("cal:server");
    } catch { /* ignore */ }
  }

  setSession(token: string) {
    this.session = token;
    try {
      if (token) localStorage.setItem("cal:session", token);
      else localStorage.removeItem("cal:session");
    } catch { /* ignore */ }
  }

  // assetUrl resolves a /api/… path for <img src> and friends — absolute on
  // native, with the session as a query param since media tags send no headers.
  assetUrl(path: string): string {
    const base = this.server ? `${this.server}${path}` : path;
    return this.session ? `${base}${path.includes("?") ? "&" : "?"}session=${encodeURIComponent(this.session)}` : base;
  }

  private get root(): string {
    return this.server ? `${this.server}${this.baseUrl}` : this.baseUrl;
  }

  async register(input: AuthRequest): Promise<{ user: User; session: string }> {
    const out = await this.request<{ user: User; session: string }>("/auth/register", { method: "POST", body: input });
    if (out.session) this.setSession(out.session);
    return out;
  }

  async login(input: AuthRequest): Promise<{ user: User; session: string }> {
    const out = await this.request<{ user: User; session: string }>("/auth/login", { method: "POST", body: input });
    if (out.session) this.setSession(out.session);
    return out;
  }

  async logout(): Promise<void> {
    await this.request<void>("/auth/logout", { method: "POST" });
  }

  async me(): Promise<User> {
    return this.request<User>("/me");
  }

  async entries(params: { from?: string; to?: string; q?: string; workspace?: string } = {}): Promise<Entry[]> {
    const search = new URLSearchParams();
    if (params.from) search.set("from", params.from);
    if (params.to) search.set("to", params.to);
    if (params.q) search.set("q", params.q);
    if (params.workspace) search.set("workspace", params.workspace);
    const suffix = search.size ? `?${search.toString()}` : "";
    return this.request<Entry[]>(`/entries${suffix}`);
  }

  async refreshLink(id: string): Promise<void> {
    await this.request(`/entries/${id}/refresh-link`, { method: "POST" });
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

  async createFeed(input: { name: string; url: string; color?: string; kind?: string }): Promise<Feed> {
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

  async weeklyReview(): Promise<WeekReview> {
    return this.request<WeekReview>("/review/week");
  }

  async habits(): Promise<Habit[]> {
    return this.request<Habit[]>("/habits");
  }

  async unfurl(url: string): Promise<Unfurl> {
    return this.request<Unfurl>(`/unfurl?url=${encodeURIComponent(url)}`);
  }

  async webhooks(): Promise<Webhook[]> {
    return this.request("/webhooks");
  }

  async addWebhook(url: string): Promise<Webhook> {
    return this.request("/webhooks", { method: "POST", body: JSON.stringify({ url }) });
  }

  async deleteWebhook(id: string): Promise<void> {
    return this.request(`/webhooks/${id}`, { method: "DELETE" });
  }

  async discoverCaldav(input: { url: string; username: string; password: string }): Promise<{ href: string; name: string }[]> {
    return this.request("/caldav/discover", { method: "POST", body: JSON.stringify(input) });
  }

  async connectCarddav(input: { name?: string; url: string; username: string; password: string }): Promise<{ imported: number; found: number }> {
    return this.request("/carddav", { method: "POST", body: JSON.stringify(input) });
  }

  async googleStatus(): Promise<{ connected: boolean }> {
    return this.request("/google/status");
  }

  async googleConnect(): Promise<{ url: string }> {
    return this.request("/google/connect");
  }

  async googleSync(): Promise<void> {
    return this.request("/google/sync", { method: "POST" });
  }

  async googleDisconnect(): Promise<void> {
    return this.request("/google", { method: "DELETE" });
  }

  async storage(): Promise<{ usedBytes: number; quotaBytes: number }> {
    return this.request("/storage");
  }

  async files(): Promise<FileRec[]> {
    return this.request("/files");
  }

  async deleteFile(id: string): Promise<void> {
    return this.request(`/files/${id}`, { method: "DELETE" });
  }

  async shareFile(id: string, on: boolean): Promise<{ shareToken: string | null }> {
    return this.request(`/files/${id}/share`, { method: "POST", body: JSON.stringify({ on }) });
  }

  async boards(): Promise<Board[]> {
    return this.request("/boards");
  }

  async createBoard(name: string, color?: string, template?: string): Promise<Board> {
    return this.request("/boards", { method: "POST", body: JSON.stringify({ name, color, template }) });
  }

  async deleteBoard(id: string): Promise<void> {
    return this.request(`/boards/${id}`, { method: "DELETE" });
  }

  async boardView(id: string): Promise<{ columns: BoardColumn[]; cards: Entry[] }> {
    return this.request(`/boards/${id}/view`);
  }

  async createColumn(boardId: string, name: string): Promise<BoardColumn> {
    return this.request(`/boards/${boardId}/columns`, { method: "POST", body: JSON.stringify({ name }) });
  }

  async updateColumn(id: string, patch: { name?: string; wipLimit?: number }): Promise<void> {
    return this.request(`/columns/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
  }

  async deleteColumn(id: string): Promise<void> {
    return this.request(`/columns/${id}`, { method: "DELETE" });
  }

  async moveCard(entryId: string, columnId: string | null, position: number): Promise<void> {
    return this.request(`/cards/${entryId}/move`, { method: "POST", body: JSON.stringify({ columnId, position }) });
  }

  async startTimer(opts: { entryId?: string; note?: string; planned?: number; billable?: boolean; rate?: number; projectId?: string } = {}): Promise<TimeEntry> {
    return this.request("/timer/start", { method: "POST", body: JSON.stringify(opts) });
  }
  async stopTimer(): Promise<TimeEntry> {
    return this.request("/timer/stop", { method: "POST", body: "{}" });
  }
  async currentTimer(): Promise<TimeEntry | null> {
    const r = await fetch(`${this.root}/timer/current`, { credentials: "include", headers: this.headers(false) });
    return r.status === 204 ? null : r.json();
  }
  async timeSummary(): Promise<TimeSummary> {
    return this.request("/time/summary");
  }
  async entryActivity(id: string): Promise<ActivityItem[]> {
    return this.request(`/entries/${id}/activity`);
  }
  async trash(): Promise<Entry[]> {
    return this.request("/trash");
  }
  async restoreEntry(id: string): Promise<void> {
    return this.request(`/trash/${id}/restore`, { method: "POST", body: "{}" });
  }
  async purgeEntry(id: string): Promise<void> {
    return this.request(`/trash/${id}`, { method: "DELETE" });
  }
  async updateBoard(id: string, patch: { description?: string; targetDate?: string }): Promise<void> {
    return this.request(`/boards/${id}`, { method: "PATCH", body: JSON.stringify(patch) });
  }
  async shareBoard(id: string, on: boolean, edit = false): Promise<{ shareToken: string; edit: boolean }> {
    return this.request(`/boards/${id}/share`, { method: "POST", body: JSON.stringify({ on, edit }) });
  }
  async sharedBoard(token: string): Promise<{ name: string; description: string; targetDate?: string; edit: boolean; columns: { id: string; name: string }[]; cards: { id: string; columnId?: string; title: string; completed: boolean; date: string; tags: string[] }[] }> {
    const r = await fetch(`${this.root}/shared/boards/${token}`, { headers: this.headers(false) });
    if (!r.ok) throw new Error("not found");
    return r.json();
  }
  async sharedBoardMove(token: string, cardId: string, columnId: string, position: number): Promise<void> {
    const r = await fetch(`${this.root}/shared/boards/${token}/cards/${cardId}/move`, {
      method: "POST",
      headers: this.headers(true),
      body: JSON.stringify({ columnId, position }),
    });
    if (!r.ok) throw new Error(await r.text());
  }
  async tags(): Promise<Record<string, number>> {
    return this.request("/tags");
  }
  async timeLog(from?: string, to?: string): Promise<TimeEntry[]> {
    const q = from && to ? `?from=${from}&to=${to}` : "";
    return this.request(`/time/log${q}`);
  }
  async deleteTimeEntry(id: string): Promise<void> {
    return this.request(`/time/log/${id}`, { method: "DELETE" });
  }

  async createTimeEntry(input: TimeEntryInput): Promise<TimeEntry> {
    return this.request("/time/log", { method: "POST", body: input });
  }

  async updateTimeEntry(id: string, patch: TimeEntryPatch): Promise<TimeEntry> {
    return this.request(`/time/log/${id}`, { method: "PATCH", body: patch });
  }

  async dashboard(workspace?: string): Promise<DashboardStats> {
    const q = workspace ? `?workspace=${encodeURIComponent(workspace)}` : "";
    return this.request(`/dashboard${q}`);
  }

  async workspaces(): Promise<Workspace[]> {
    return this.request("/workspaces");
  }
  async createWorkspace(input: { name: string; color?: string; icon?: string }): Promise<Workspace> {
    return this.request("/workspaces", { method: "POST", body: input });
  }
  async updateWorkspace(id: string, patch: { name?: string; color?: string; icon?: string; position?: number }): Promise<void> {
    return this.request(`/workspaces/${id}`, { method: "PATCH", body: patch });
  }
  async deleteWorkspace(id: string): Promise<void> {
    return this.request(`/workspaces/${id}`, { method: "DELETE" });
  }

  async updateFile(id: string, patch: { tags?: string[]; workspaceId?: string }): Promise<FileRec> {
    return this.request(`/files/${id}`, { method: "PATCH", body: patch });
  }

  async filters(): Promise<SavedFilter[]> {
    return this.request("/filters");
  }
  async createFilter(name: string, filter: Record<string, unknown>): Promise<SavedFilter> {
    return this.request("/filters", { method: "POST", body: { name, filter } });
  }
  async deleteFilter(id: string): Promise<void> {
    return this.request(`/filters/${id}`, { method: "DELETE" });
  }

  async mailAccounts(): Promise<MailAccount[]> {
    return this.request("/mail/accounts");
  }
  async createMailAccount(input: {
    name?: string; email: string; imapHost: string; imapPort?: number;
    smtpHost: string; smtpPort?: number; username?: string; password: string;
  }): Promise<MailAccount> {
    return this.request("/mail/accounts", { method: "POST", body: input });
  }
  async deleteMailAccount(id: string): Promise<void> {
    return this.request(`/mail/accounts/${id}`, { method: "DELETE" });
  }
  async testMailAccount(id: string): Promise<{ ok: boolean }> {
    return this.request(`/mail/accounts/${id}/test`, { method: "POST", body: "{}" });
  }
  async mailMailboxes(id: string): Promise<{ name: string; delimiter: string }[]> {
    return this.request(`/mail/${id}/mailboxes`);
  }
  async mailMessages(id: string, mailbox = "INBOX", page = 0): Promise<{ total: number; messages: MailSummary[] }> {
    return this.request(`/mail/${id}/messages?mailbox=${encodeURIComponent(mailbox)}&page=${page}`);
  }
  async mailMessage(id: string, uid: number, mailbox = "INBOX"): Promise<MailMessage> {
    return this.request(`/mail/${id}/message/${uid}?mailbox=${encodeURIComponent(mailbox)}`);
  }
  async mailFlag(id: string, uid: number, seen: boolean, mailbox = "INBOX"): Promise<void> {
    return this.request(`/mail/${id}/message/${uid}/flag?mailbox=${encodeURIComponent(mailbox)}`, { method: "POST", body: { seen } });
  }
  async mailDelete(id: string, uid: number, mailbox = "INBOX"): Promise<void> {
    return this.request(`/mail/${id}/message/${uid}?mailbox=${encodeURIComponent(mailbox)}`, { method: "DELETE" });
  }
  async mailSend(id: string, input: { to: string; cc?: string; subject: string; text: string; attachments?: string[] }): Promise<void> {
    return this.request(`/mail/${id}/send`, { method: "POST", body: input });
  }

  async githubInbox(): Promise<{ number: number; title: string; state: string; url: string; repo: string; isPR: boolean; labels: string[] }[]> {
    return this.request("/github/inbox");
  }
  async githubActivity(): Promise<{ login: string; eventsThisWeek: number }> {
    return this.request("/github/activity");
  }
  async githubImport(url: string, boardId?: string, columnId?: string): Promise<Entry> {
    return this.request("/github/import", { method: "POST", body: JSON.stringify({ url, boardId, columnId }) });
  }

  async search(q: string): Promise<{ entries: Entry[]; files: { id: string; name: string; origName: string }[]; boards: Board[] }> {
    return this.request(`/search?q=${encodeURIComponent(q)}`);
  }
  async agenda(days = 7): Promise<string> {
    const r = await fetch(`${this.root}/agenda?days=${days}`, { credentials: "include", headers: this.headers(false) });
    return r.text();
  }

  async activity(): Promise<Record<string, number>> {
    return this.request("/activity");
  }

  async pushSubscriptions(): Promise<{ id: string; label: string; endpoint: string }[]> {
    return this.request("/push/subscriptions");
  }

  async deletePushSubscription(id: string): Promise<void> {
    return this.request(`/push/subscriptions/${id}`, { method: "DELETE" });
  }

  async upload(file: File, opts: { tags?: string[]; workspaceId?: string } = {}): Promise<{ url: string; name: string; markdown: string }> {
    const form = new FormData();
    form.append("file", file);
    if (opts.tags?.length) form.append("tags", opts.tags.join(","));
    if (opts.workspaceId) form.append("workspaceId", opts.workspaceId);
    const response = await fetch(`${this.root}/files`, {
      method: "POST",
      credentials: "include",
      headers: this.headers(false),
      body: form,
    });
    if (!response.ok) throw new Error(`Upload failed: ${response.status}`);
    return response.json();
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
    const response = await fetch(`${this.root}/restore`, {
      method: "POST",
      credentials: "include",
      headers: this.headers(true),
      body: file,
    });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `Request failed: ${response.status}`);
    }
    return response.json();
  }

  async importIcs(file: File): Promise<{ imported: number }> {
    const response = await fetch(`${this.root}/import`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "text/calendar", ...(this.session ? { Authorization: `Bearer ${this.session}` } : {}) },
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

  private headers(withJson: boolean): Record<string, string> {
    const h: Record<string, string> = {};
    if (withJson) h["Content-Type"] = "application/json";
    if (this.session) h["Authorization"] = `Bearer ${this.session}`;
    return h;
  }

  private async request<T>(path: string, init: { method?: string; body?: unknown } = {}): Promise<T> {
    const response = await fetch(`${this.root}${path}`, {
      method: init.method ?? "GET",
      credentials: "include",
      headers: this.headers(!!init.body),
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

export interface Board {
  id: string;
  name: string;
  color: string;
  description: string;
  targetDate?: string;
  shareToken?: string;
  total: number;
  done: number;
  minutes: number; // tracked time against this board
  createdAt: string;
}

export interface BoardColumn {
  id: string;
  boardId: string;
  name: string;
  position: number;
  wipLimit?: number;
}

export interface TimeEntry {
  id: string;
  entryId?: string;
  title: string;
  startAt: string;
  endAt?: string;
  note: string;
  planned?: number;
  billable?: boolean;
  rate?: number;
  projectId?: string;
  project?: string;
  tags: string[];
}

export interface Workspace {
  id: string;
  name: string;
  color: string;
  icon: string;
  position: number;
  createdAt: string;
}

export interface FeedItem {
  kind: "entry" | "file" | "card";
  action: string;
  title: string;
  entryId?: string;
  at: string;
}

export interface DashboardStats {
  tasksTotal: number;
  tasksDone: number;
  doneThisWeek: number;
  weekActivity: Record<string, number>;
  deadlines: Entry[];
  feed: FeedItem[];
  timeTodayMin: number;
  timeWeekMin: number;
  running?: TimeEntry;
}

export interface SavedFilter {
  id: string;
  name: string;
  filter: Record<string, unknown>;
  createdAt: string;
}

export interface MailAccount {
  id: string;
  name: string;
  email: string;
  imapHost: string;
  imapPort: number;
  smtpHost: string;
  smtpPort: number;
  username: string;
  createdAt: string;
}

export interface MailSummary {
  uid: number;
  from: string;
  to?: string[];
  subject: string;
  date: string;
  seen: boolean;
  size: number;
}

export interface MailMessage {
  uid: number;
  from: string;
  to: string[];
  subject: string;
  text: string;
  html: string;
}

export interface TimeEntryInput {
  entryId?: string;
  note?: string;
  startAt?: string;
  endAt?: string;
  planned?: number;
  billable?: boolean;
  rate?: number;
  projectId?: string;
  tags?: string[];
}

export type TimeEntryPatch = Partial<{
  startAt: string | null;
  endAt: string | null;
  note: string;
  tags: string[];
  billable: boolean;
  rate: number | null;
  projectId: string | null;
  entryId: string | null;
  planned: number | null;
}>;

export interface ActivityItem {
  id: string;
  action: string;
  detail: string;
  createdAt: string;
}

export interface TimeSummary {
  todayMinutes: number;
  weekMinutes: number;
  billableAmount: number;
  perEntry: { entryId: string; title: string; minutes: number }[];
}

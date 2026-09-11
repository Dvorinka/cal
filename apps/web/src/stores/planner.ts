import {
  CalApi,
  type CaldavAccount,
  type Country,
  type Entry,
  type EntryInput,
  type EntryPatch,
  type Feed,
  type FeedEvent,
  type Holiday,
  type Settings,
  type User,
  type Workspace,
} from "@cal/api-client";
import { create } from "zustand";
import {
  cacheEntries,
  cacheSettings,
  cacheUser,
  clearCachedUser,
  readCachedEntries,
  readCachedSettings,
  readCachedUser,
} from "../lib/offline";
import { ApiError } from "@cal/api-client";
import { enqueue, isOfflineError, newTempId, readQueue, writeQueue } from "../lib/opqueue";

export interface Toast {
  id: number;
  message: string;
  action?: { label: string; run: () => void };
}

let toastSeq = 0;

// pushWidgetConfig hands the Android home widget the server origin + the
// read-only widget token. No-ops outside the Capacitor shell.
function pushWidgetConfig(settings: Settings) {
  try {
    const cap = (window as unknown as { Capacitor?: { Plugins?: Record<string, { set?: (v: unknown) => Promise<void> }> } }).Capacitor;
    void cap?.Plugins?.WidgetConfig?.set?.({ server: location.origin, widgetToken: settings.widgetToken });
  } catch {
    /* not in the app */
  }
}

interface PlannerState {
  api: CalApi;
  user?: User;
  booted: boolean;
  entries: Entry[];
  feedEvents: FeedEvent[];
  feeds: Feed[];
  holidays: Holiday[];
  countries: Country[];
  settings: Settings;
  loading: boolean;
  offline: boolean;
  pendingOps: number;
  error?: string;
  toasts: Toast[];
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  loadEntries: (params: { from?: string; to?: string; q?: string; workspace?: string }) => Promise<void>;
  workspaces: Workspace[];
  loadWorkspaces: () => Promise<void>;
  addWorkspace: (input: { name: string; color?: string; icon?: string }) => Promise<Workspace | undefined>;
  removeWorkspace: (id: string) => Promise<void>;
  /** undefined/"" = all spaces, "none" = Personal only, uuid = that space. */
  setWorkspace: (workspace: string) => Promise<void>;
  createEntry: (input: EntryInput) => Promise<Entry | undefined>;
  updateEntry: (id: string, patch: EntryPatch) => Promise<void>;
  deleteEntry: (id: string) => Promise<void>;
  loadHolidays: (country: string, year: number) => Promise<void>;
  loadCountries: () => Promise<void>;
  loadFeeds: () => Promise<void>;
  accounts: CaldavAccount[];
  loadAccounts: () => Promise<void>;
  addAccount: (input: { name?: string; url: string; username: string; password: string }) => Promise<boolean>;
  removeAccount: (id: string) => Promise<void>;
  syncAccount: (id: string) => Promise<void>;
  loadFeedEvents: (params: { from: string; to: string }) => Promise<void>;
  addFeed: (input: { name: string; url: string; color?: string; kind?: string }) => Promise<boolean>;
  removeFeed: (id: string) => Promise<void>;
  refreshFeed: (id: string) => Promise<void>;
  importIcs: (file: File) => Promise<void>;
  restore: (file: File) => Promise<void>;
  rotateToken: (kind: "widget" | "api") => Promise<void>;
  updateSettings: (settings: Settings) => Promise<void>;
  toast: (message: string, action?: Toast["action"]) => void;
  dismissToast: (id: number) => void;
  flushQueue: () => Promise<void>;
}

export const usePlanner = create<PlannerState>((set, get) => ({
  api: new CalApi(),
  entries: readCachedEntries(),
  feedEvents: [],
  feeds: [],
  accounts: [],
  holidays: [],
  countries: [],
  workspaces: [],
  settings: readCachedSettings(),
  loading: false,
  booted: false,
  offline: false,
  pendingOps: readQueue().length,
  toasts: [],

  async bootstrap() {
    try {
      const user = await get().api.me();
      const settings = await get().api.settings();
      cacheSettings(settings);
      cacheUser(user);
      pushWidgetConfig(settings);
      set({ user, settings, booted: true, offline: false });
      void get().loadCountries();
      void get().loadWorkspaces();
      void get().flushQueue();
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        set({ user: undefined, booted: true });
        return;
      }
      // Network/server failure: fall back to the cached session if present.
      const user = readCachedUser();
      set({ user, booted: true, offline: user !== undefined });
    }
  },

  async login(email, password) {
    const user = await get().api.login({ email, password });
    const settings = await get().api.settings();
    cacheSettings(settings);
    cacheUser(user);
    pushWidgetConfig(settings);
    set({ user, settings, error: undefined, offline: false });
    void get().loadCountries();
  },

  async register(email, password) {
    const user = await get().api.register({ email, password });
    const settings = await get().api.settings();
    cacheSettings(settings);
    cacheUser(user);
    pushWidgetConfig(settings);
    set({ user, settings, error: undefined, offline: false });
    void get().loadCountries();
  },

  async logout() {
    try {
      await get().api.logout();
    } finally {
      clearCachedUser();
      set({ user: undefined, entries: [], holidays: [] });
    }
  },

  async loadEntries(params) {
    set({ loading: true });
    try {
      // Workspace scope applies globally unless a caller overrides it.
      const workspace = params.workspace ?? get().settings.activeWorkspace ?? "";
      const entries = await get().api.entries({ ...params, workspace });
      cacheEntries(entries);
      set({ entries, loading: false, offline: false, error: undefined });
      void get().flushQueue();
    } catch (error) {
      set({
        entries: readCachedEntries(),
        loading: false,
        offline: true,
        error: error instanceof Error ? error.message : "Offline cache loaded",
      });
    }
  },

  async loadWorkspaces() {
    try {
      const workspaces = await get().api.workspaces();
      set({ workspaces });
    } catch {
      set({ workspaces: [] });
    }
  },

  async addWorkspace(input) {
    try {
      const w = await get().api.createWorkspace(input);
      set({ workspaces: [...get().workspaces, w] });
      return w;
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to create space");
      return undefined;
    }
  },

  async removeWorkspace(id) {
    try {
      await get().api.deleteWorkspace(id);
      set({ workspaces: get().workspaces.filter((w) => w.id !== id) });
      // If the deleted space was active, fall back to All.
      if (get().settings.activeWorkspace === id) await get().setWorkspace("");
      await get().loadEntries({});
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to delete space");
    }
  },

  async setWorkspace(workspace) {
    const settings = { ...get().settings, activeWorkspace: workspace || undefined };
    await get().updateSettings(settings);
    await get().loadEntries({});
  },

  async createEntry(input) {
    try {
      // New entries inherit the active workspace (not "none" — that's Personal).
      const ws = get().settings.activeWorkspace;
      if (!input.workspaceId && ws && ws !== "none") input = { ...input, workspaceId: ws };
      const entry = await get().api.createEntry(input);
      const entries = [...get().entries, entry];
      cacheEntries(entries);
      set({ entries, offline: false });
      return entry;
    } catch (error) {
      if (!isOfflineError(error)) {
        get().toast(error instanceof Error ? error.message : "Failed to create entry");
        return undefined;
      }
      // Offline: queue the create, show a temp entry so the UI stays live.
      const tempId = newTempId();
      enqueue({ kind: "create", id: tempId, input });
      const tmp: Entry = {
        id: tempId,
        title: input.title,
        content: input.content ?? "",
        type: input.type ?? "task",
        linkUrl: input.linkUrl,
        date: input.date,
        startTime: input.startTime || undefined,
        endTime: input.endTime || undefined,
        completed: false,
        pinned: false,
        color: input.color ?? "slate",
        tags: input.tags ?? [],
        recur: input.recur ?? "none",
        remind: input.remind ?? undefined,
        createdAt: new Date().toISOString(),
      };
      const entries = [...get().entries, tmp];
      cacheEntries(entries);
      set({ entries, offline: true, pendingOps: readQueue().length });
      get().toast("Saved offline — will sync when back");
      return tmp;
    }
  },

  async updateEntry(id, patch) {
    const before = get().entries;
    const optimistic = before.map((entry) =>
      entry.id === id ? { ...entry, ...patch, remind: patch.remind === null ? undefined : (patch.remind ?? entry.remind) } : entry,
    );
    set({ entries: optimistic });
    cacheEntries(optimistic);
    try {
      const updated = await get().api.updateEntry(id, patch);
      // Completing a recurring task spawns the next occurrence server-side; refetch.
      if (patch.completed && updated.type === "task" && updated.recur !== "none") {
        const fresh = await get().api.entries({});
        const merged = new Map(fresh.map((entry) => [entry.id, entry]));
        for (const entry of get().entries) if (!merged.has(entry.id)) merged.set(entry.id, entry);
        const entries = [...merged.values()];
        cacheEntries(entries);
        set({ entries });
      } else {
        const entries = get().entries.map((entry) => (entry.id === id ? updated : entry));
        cacheEntries(entries);
        set({ entries });
      }
    } catch (error) {
      if (isOfflineError(error)) {
        // Keep the optimistic state; queue the patch for replay.
        enqueue({ kind: "update", id, patch });
        set({ offline: true, pendingOps: readQueue().length });
        get().toast("Saved offline — will sync when back");
        return;
      }
      set({ entries: before });
      cacheEntries(before);
      get().toast(error instanceof Error ? error.message : "Failed to save entry");
    }
  },

  async deleteEntry(id) {
    const snapshot = get().entries;
    const removed = snapshot.find((entry) => entry.id === id);
    const entries = snapshot.filter((entry) => entry.id !== id);
    set({ entries });
    cacheEntries(entries);
    try {
      await get().api.deleteEntry(id);
      if (removed) {
        get().toast("Entry deleted", {
          label: "Undo",
          run: () => {
            void get().createEntry({
              title: removed.title,
              content: removed.content,
              type: removed.type,
              linkUrl: removed.linkUrl,
              date: removed.date,
              startTime: removed.startTime,
              endTime: removed.endTime,
              color: removed.color,
              tags: removed.tags,
              recur: removed.recur,
            });
          },
        });
      }
    } catch (error) {
      if (isOfflineError(error)) {
        enqueue({ kind: "delete", id });
        set({ offline: true, pendingOps: readQueue().length });
        get().toast("Deleted offline — will sync when back");
        return;
      }
      set({ entries: snapshot });
      cacheEntries(snapshot);
      get().toast(error instanceof Error ? error.message : "Failed to delete entry");
    }
  },

  // flushQueue replays pending mutations in order; creates swap tmp ids.
  async flushQueue() {
    const ops = readQueue();
    if (ops.length === 0) return;
    const idMap = new Map<string, string>();
    for (const op of ops) {
      const id = idMap.get(op.id) ?? op.id;
      try {
        if (op.kind === "create") {
          const entry = await get().api.createEntry(op.input);
          idMap.set(op.id, entry.id);
        } else if (op.kind === "update") {
          if (!id.startsWith("tmp-")) await get().api.updateEntry(id, op.patch);
        } else if (op.kind === "delete") {
          if (!id.startsWith("tmp-")) await get().api.deleteEntry(id);
        }
      } catch (error) {
        if (isOfflineError(error)) {
          // Still offline — keep this op and everything after it.
          const idx = ops.indexOf(op);
          writeQueue(ops.slice(idx));
          set({ pendingOps: ops.length - idx });
          return;
        }
        // Real error (e.g. 4xx): drop the op, keep flushing the rest.
      }
    }
    writeQueue([]);
    set({ pendingOps: 0 });
    await get().loadEntries({});
    get().toast("Offline changes synced");
  },

  async loadHolidays(country, year) {
    try {
      const holidays = await get().api.holidays(country, year);
      set({ holidays });
    } catch {
      set({ holidays: [] });
    }
  },

  async loadCountries() {
    try {
      const countries = await get().api.countries();
      set({ countries });
    } catch {
      set({ countries: [] });
    }
  },

  async loadFeeds() {
    try {
      const feeds = await get().api.feeds();
      set({ feeds });
    } catch {
      set({ feeds: [] });
    }
  },

  async loadFeedEvents(params) {
    if (get().feeds.length === 0) {
      set({ feedEvents: [] });
      return;
    }
    try {
      const feedEvents = await get().api.feedEvents(params);
      set({ feedEvents });
    } catch {
      // Feeds are a cache mirror — offline just shows fewer external events.
    }
  },

  async addFeed(input) {
    try {
      await get().api.createFeed(input);
      await get().loadFeeds();
      get().toast("Feed added");
      return true;
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to add feed");
      return false;
    }
  },

  async removeFeed(id) {
    try {
      await get().api.deleteFeed(id);
      set({ feeds: get().feeds.filter((f) => f.id !== id), feedEvents: get().feedEvents.filter((e) => e.feedId !== id) });
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to delete feed");
    }
  },

  async refreshFeed(id) {
    try {
      await get().api.refreshFeed(id);
      await get().loadFeeds();
      get().toast("Feed refreshed");
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to refresh feed");
    }
  },

  async loadAccounts() {
    try {
      const accounts = await get().api.caldavAccounts();
      set({ accounts });
    } catch {
      set({ accounts: [] });
    }
  },

  async addAccount(input) {
    try {
      await get().api.addCaldavAccount(input);
      await get().loadAccounts();
      get().toast("Calendar connected — syncing");
      return true;
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to connect");
      return false;
    }
  },

  async removeAccount(id) {
    try {
      await get().api.deleteCaldavAccount(id);
      set({ accounts: get().accounts.filter((a) => a.id !== id), entries: get().entries.filter((e) => e.accountId !== id) });
      get().toast("Calendar removed — its events stay deleted locally");
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to remove");
    }
  },

  async syncAccount(id) {
    try {
      await get().api.syncCaldav(id);
      await get().loadAccounts();
      await get().loadEntries({});
      get().toast("Synced");
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Sync failed");
    }
  },

  async restore(file) {
    try {
      const out = await get().api.restore(file);
      await get().loadEntries({});
      get().toast(`Restored ${out.restored} entr${out.restored === 1 ? "y" : "ies"}`);
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Restore failed");
    }
  },

  async importIcs(file) {
    try {
      const out = await get().api.importIcs(file);
      get().toast(`Imported ${out.imported} event${out.imported === 1 ? "" : "s"}`);
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Import failed");
    }
  },

  async rotateToken(kind) {
    try {
      const token = kind === "widget" ? await get().api.rotateWidgetToken() : await get().api.rotateApiToken();
      const settings = { ...get().settings };
      if (kind === "widget") settings.widgetToken = token;
      else settings.apiToken = token;
      cacheSettings(settings);
      set({ settings });
      get().toast("Token rotated");
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to rotate token");
    }
  },

  async updateSettings(settings) {
    try {
      const saved = await get().api.updateSettings(settings);
      cacheSettings(saved);
      set({ settings: saved });
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to save settings");
    }
  },

  toast(message, action) {
    const id = ++toastSeq;
    set({ toasts: [...get().toasts, { id, message, action }] });
    window.setTimeout(() => get().dismissToast(id), 6000);
  },

  dismissToast(id) {
    set({ toasts: get().toasts.filter((toast) => toast.id !== id) });
  },
}));

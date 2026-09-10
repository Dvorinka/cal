import {
  CalApi,
  type Country,
  type Entry,
  type EntryInput,
  type EntryPatch,
  type Holiday,
  type Settings,
  type User,
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

export interface Toast {
  id: number;
  message: string;
  action?: { label: string; run: () => void };
}

let toastSeq = 0;

interface PlannerState {
  api: CalApi;
  user?: User;
  booted: boolean;
  entries: Entry[];
  holidays: Holiday[];
  countries: Country[];
  settings: Settings;
  loading: boolean;
  offline: boolean;
  error?: string;
  toasts: Toast[];
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  loadEntries: (params: { from: string; to: string; q?: string }) => Promise<void>;
  createEntry: (input: EntryInput) => Promise<Entry | undefined>;
  updateEntry: (id: string, patch: EntryPatch) => Promise<void>;
  deleteEntry: (id: string) => Promise<void>;
  loadHolidays: (country: string, year: number) => Promise<void>;
  loadCountries: () => Promise<void>;
  updateSettings: (settings: Settings) => Promise<void>;
  toast: (message: string, action?: Toast["action"]) => void;
  dismissToast: (id: number) => void;
}

export const usePlanner = create<PlannerState>((set, get) => ({
  api: new CalApi(),
  entries: readCachedEntries(),
  holidays: [],
  countries: [],
  settings: readCachedSettings(),
  loading: false,
  booted: false,
  offline: false,
  toasts: [],

  async bootstrap() {
    try {
      const user = await get().api.me();
      const settings = await get().api.settings();
      cacheSettings(settings);
      cacheUser(user);
      set({ user, settings, booted: true, offline: false });
      void get().loadCountries();
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
    set({ user, settings, error: undefined, offline: false });
    void get().loadCountries();
  },

  async register(email, password) {
    const user = await get().api.register({ email, password });
    const settings = await get().api.settings();
    cacheSettings(settings);
    cacheUser(user);
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
      const entries = await get().api.entries(params);
      cacheEntries(entries);
      set({ entries, loading: false, offline: false, error: undefined });
    } catch (error) {
      set({
        entries: readCachedEntries(),
        loading: false,
        offline: true,
        error: error instanceof Error ? error.message : "Offline cache loaded",
      });
    }
  },

  async createEntry(input) {
    try {
      const entry = await get().api.createEntry(input);
      const entries = [...get().entries, entry];
      cacheEntries(entries);
      set({ entries, offline: false });
      return entry;
    } catch (error) {
      get().toast(error instanceof Error ? error.message : "Failed to create entry");
      return undefined;
    }
  },

  async updateEntry(id, patch) {
    const optimistic = get().entries.map((entry) => (entry.id === id ? { ...entry, ...patch } : entry));
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
      const entries = get().entries;
      set({ entries });
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
      set({ entries: snapshot });
      cacheEntries(snapshot);
      get().toast(error instanceof Error ? error.message : "Failed to delete entry");
    }
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

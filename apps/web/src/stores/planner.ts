import { CalApi, type Entry, type EntryInput, type EntryPatch, type Holiday, type Settings, type User } from "@cal/api-client";
import { create } from "zustand";
import { cacheEntries, cacheSettings, readCachedEntries, readCachedSettings } from "../lib/offline";

interface PlannerState {
  api: CalApi;
  user?: User;
  entries: Entry[];
  holidays: Holiday[];
  settings: Settings;
  loading: boolean;
  error?: string;
  bootstrap: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  loadEntries: (params: { from: string; to: string; q?: string }) => Promise<void>;
  createEntry: (input: EntryInput) => Promise<void>;
  updateEntry: (id: string, patch: EntryPatch) => Promise<void>;
  deleteEntry: (id: string) => Promise<void>;
  loadHolidays: (country: string, year: number) => Promise<void>;
  updateSettings: (settings: Settings) => Promise<void>;
}

export const usePlanner = create<PlannerState>((set, get) => ({
  api: new CalApi(),
  entries: readCachedEntries(),
  holidays: [],
  settings: readCachedSettings(),
  loading: false,
  async bootstrap() {
    try {
      const user = await get().api.me();
      const settings = await get().api.settings();
      cacheSettings(settings);
      set({ user, settings });
    } catch {
      set({ user: undefined });
    }
  },
  async login(email, password) {
    const user = await get().api.login({ email, password });
    const settings = await get().api.settings();
    cacheSettings(settings);
    set({ user, settings, error: undefined });
  },
  async register(email, password) {
    const user = await get().api.register({ email, password });
    const settings = await get().api.settings();
    cacheSettings(settings);
    set({ user, settings, error: undefined });
  },
  async logout() {
    await get().api.logout();
    set({ user: undefined, entries: [] });
  },
  async loadEntries(params) {
    set({ loading: true });
    try {
      const entries = await get().api.entries(params);
      cacheEntries(entries);
      set({ entries, loading: false, error: undefined });
    } catch (error) {
      set({ entries: readCachedEntries(), loading: false, error: error instanceof Error ? error.message : "Offline cache loaded" });
    }
  },
  async createEntry(input) {
    const entry = await get().api.createEntry(input);
    const entries = [...get().entries, entry];
    cacheEntries(entries);
    set({ entries });
  },
  async updateEntry(id, patch) {
    const optimistic = get().entries.map((entry) => (entry.id === id ? { ...entry, ...patch } : entry));
    set({ entries: optimistic });
    cacheEntries(optimistic);
    const updated = await get().api.updateEntry(id, patch);
    const entries = get().entries.map((entry) => (entry.id === id ? updated : entry));
    cacheEntries(entries);
    set({ entries });
  },
  async deleteEntry(id) {
    await get().api.deleteEntry(id);
    const entries = get().entries.filter((entry) => entry.id !== id);
    cacheEntries(entries);
    set({ entries });
  },
  async loadHolidays(country, year) {
    const holidays = await get().api.holidays(country, year);
    set({ holidays });
  },
  async updateSettings(settings) {
    const saved = await get().api.updateSettings(settings);
    cacheSettings(saved);
    set({ settings: saved });
  },
}));

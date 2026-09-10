import type { Entry } from "@cal/api-client";
import { create } from "zustand";
import { addDays, addMonths, todayIso, type CalendarView } from "../lib/date";

export type EditorState =
  | { mode: "closed" }
  | { mode: "create"; date: string; startTime?: string }
  | { mode: "edit"; entry: Entry };

export interface ContextMenuState {
  x: number;
  y: number;
  entry: Entry;
}

interface UiState {
  view: CalendarView;
  anchor: Date;
  selectedDate: string;
  paletteOpen: boolean;
  editor: EditorState;
  contextMenu?: ContextMenuState;
  sidebarOpen: boolean;
  hiddenTypes: Set<string>;
  setView: (view: CalendarView) => void;
  setAnchor: (date: Date) => void;
  shift: (direction: -1 | 1) => void;
  goToday: () => void;
  selectDate: (date: string) => void;
  openPalette: () => void;
  closePalette: () => void;
  openCreate: (date?: string, startTime?: string) => void;
  openEdit: (entry: Entry) => void;
  closeEditor: () => void;
  setContextMenu: (menu?: ContextMenuState) => void;
  toggleSidebar: () => void;
  closeSidebar: () => void;
  toggleType: (type: string) => void;
}

const persisted = (() => {
  try {
    return JSON.parse(localStorage.getItem("cal:ui") ?? "{}") as { view?: CalendarView; hiddenTypes?: string[] };
  } catch {
    return {};
  }
})();

function persist(state: Pick<UiState, "view" | "hiddenTypes">) {
  localStorage.setItem("cal:ui", JSON.stringify({ view: state.view, hiddenTypes: [...state.hiddenTypes] }));
}

export const useUi = create<UiState>((set, get) => ({
  view: persisted.view === "week" || persisted.view === "day" ? persisted.view : "month",
  anchor: new Date(),
  selectedDate: todayIso(),
  paletteOpen: false,
  editor: { mode: "closed" },
  contextMenu: undefined,
  sidebarOpen: false,
  hiddenTypes: new Set(persisted.hiddenTypes ?? []),

  setView(view) {
    set({ view });
    persist(get());
  },
  setAnchor(anchor) {
    set({ anchor });
  },
  shift(direction) {
    const { view, anchor } = get();
    if (view === "day") set({ anchor: addDays(anchor, direction) });
    else if (view === "week") set({ anchor: addDays(anchor, direction * 7) });
    else set({ anchor: addMonths(anchor, direction) });
  },
  goToday() {
    set({ anchor: new Date(), selectedDate: todayIso() });
  },
  selectDate(selectedDate) {
    set({ selectedDate, anchor: new Date(`${selectedDate}T12:00:00`) });
  },
  openPalette() {
    set({ paletteOpen: true, contextMenu: undefined });
  },
  closePalette() {
    set({ paletteOpen: false });
  },
  openCreate(date, startTime) {
    set({ editor: { mode: "create", date: date ?? get().selectedDate, startTime }, contextMenu: undefined });
  },
  openEdit(entry) {
    set({ editor: { mode: "edit", entry }, contextMenu: undefined });
  },
  closeEditor() {
    set({ editor: { mode: "closed" } });
  },
  setContextMenu(contextMenu) {
    set({ contextMenu });
  },
  toggleSidebar() {
    set({ sidebarOpen: !get().sidebarOpen });
  },
  closeSidebar() {
    set({ sidebarOpen: false });
  },
  toggleType(type) {
    const hiddenTypes = new Set(get().hiddenTypes);
    if (hiddenTypes.has(type)) hiddenTypes.delete(type);
    else hiddenTypes.add(type);
    set({ hiddenTypes });
    persist(get());
  },
}));

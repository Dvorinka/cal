import type { Entry } from "@cal/api-client";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight, CalendarDays, Check, Link2, Moon, Plus, Search, StickyNote, Sun } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { formatDayShort } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

interface Action {
  id: string;
  label: string;
  icon: React.ReactNode;
  run: () => void;
}

export function CommandPalette() {
  const open = useUi((state) => state.paletteOpen);
  const close = useUi((state) => state.closePalette);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const selectDate = useUi((state) => state.selectDate);
  const setView = useUi((state) => state.setView);
  const goToday = useUi((state) => state.goToday);
  const api = usePlanner((state) => state.api);
  const settings = usePlanner((state) => state.settings);
  const updateSettings = usePlanner((state) => state.updateSettings);
  const selectedDate = useUi((state) => state.selectedDate);
  const navigate = useNavigate();

  const [query, setQuery] = useState("");
  const [results, setResults] = useState<Entry[]>([]);
  const [cursor, setCursor] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const actions = useMemo<Action[]>(
    () => [
      {
        id: "new",
        label: "New entry",
        icon: <Plus size={15} />,
        run: () => {
          close();
          openCreate(selectedDate);
        },
      },
      {
        id: "today",
        label: "Go to today",
        icon: <CalendarDays size={15} />,
        run: () => {
          close();
          goToday();
          navigate("/today");
        },
      },
      {
        id: "calendar",
        label: "Go to calendar",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/");
        },
      },
      {
        id: "tasks",
        label: "Go to tasks",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/tasks");
        },
      },
      {
        id: "notes",
        label: "Go to notes",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/notes");
        },
      },
      {
        id: "links",
        label: "Go to links",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/links");
        },
      },
      {
        id: "month",
        label: "Month view",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/");
          setView("month");
        },
      },
      {
        id: "week",
        label: "Week view",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/");
          setView("week");
        },
      },
      {
        id: "day",
        label: "Day view",
        icon: <ArrowRight size={15} />,
        run: () => {
          close();
          navigate("/");
          setView("day");
        },
      },
      {
        id: "theme",
        label: settings.theme === "dark" ? "Switch to light theme" : "Switch to dark theme",
        icon: settings.theme === "dark" ? <Sun size={15} /> : <Moon size={15} />,
        run: () => {
          close();
          void updateSettings({ ...settings, theme: settings.theme === "dark" ? "light" : "dark" });
        },
      },
    ],
    [close, goToday, openCreate, selectedDate, setView, settings, updateSettings, navigate],
  );

  useEffect(() => {
    if (open) {
      setQuery("");
      setResults([]);
      setCursor(0);
      window.setTimeout(() => inputRef.current?.focus(), 20);
    }
  }, [open]);

  useEffect(() => {
    if (!open || !query.trim()) {
      setResults([]);
      return;
    }
    const timer = window.setTimeout(async () => {
      try {
        setResults(await api.entries({ q: query.trim() }));
      } catch {
        // Offline: fall back to the locally cached entries.
        const q = query.trim().toLowerCase();
        setResults(
          usePlanner
            .getState()
            .entries.filter(
              (entry) =>
                entry.title.toLowerCase().includes(q) ||
                (entry.content ?? "").toLowerCase().includes(q) ||
                entry.tags.some((tag) => tag.toLowerCase() === q),
            ),
        );
      }
    }, 180);
    return () => window.clearTimeout(timer);
  }, [query, open, api]);

  const filteredActions = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return actions;
    return actions.filter((action) => action.label.toLowerCase().includes(q));
  }, [actions, query]);

  const items = useMemo(
    () => [
      ...filteredActions.map((action) => ({ kind: "action" as const, action })),
      ...results.map((entry) => ({ kind: "entry" as const, entry })),
    ],
    [filteredActions, results],
  );

  useEffect(() => setCursor(0), [items.length]);

  useEffect(() => {
    listRef.current?.querySelector(".palette-item.active")?.scrollIntoView({ block: "nearest" });
  }, [cursor]);

  function choose(index: number) {
    const item = items[index];
    if (!item) return;
    if (item.kind === "action") {
      item.action.run();
    } else {
      close();
      selectDate(item.entry.date);
      openEdit(item.entry);
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="scrim"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.12 }}
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) close();
          }}
        >
          <motion.div
            className="palette"
            role="dialog"
            aria-label="Command palette"
            initial={{ opacity: 0, scale: 0.98, y: -8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.98, y: -4 }}
            transition={{ type: "spring", duration: 0.3, bounce: 0 }}
          >
            <div className="palette-input">
              <Search size={16} style={{ color: "var(--text-3)", flex: "none" }} />
              <input
                ref={inputRef}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder="Search entries or run a command…"
                onKeyDown={(event) => {
                  if (event.key === "ArrowDown") {
                    event.preventDefault();
                    setCursor((c) => Math.min(c + 1, items.length - 1));
                  } else if (event.key === "ArrowUp") {
                    event.preventDefault();
                    setCursor((c) => Math.max(c - 1, 0));
                  } else if (event.key === "Enter") {
                    event.preventDefault();
                    choose(cursor);
                  } else if (event.key === "Escape") {
                    close();
                  }
                }}
              />
            </div>
            <div className="palette-list" ref={listRef}>
              {filteredActions.length > 0 && <div className="palette-group">Actions</div>}
              {filteredActions.map((action) => {
                const index = items.findIndex((item) => item.kind === "action" && item.action.id === action.id);
                return (
                  <button
                    key={action.id}
                    type="button"
                    className={`palette-item ${cursor === index ? "active" : ""}`}
                    onMouseEnter={() => setCursor(index)}
                    onClick={() => choose(index)}
                  >
                    <span className="icon">{action.icon}</span>
                    {action.label}
                  </button>
                );
              })}
              {results.length > 0 && <div className="palette-group">Entries</div>}
              {results.map((entry) => {
                const index = filteredActions.length + results.indexOf(entry);
                const Icon = entry.type === "note" ? StickyNote : entry.type === "link" ? Link2 : Check;
                return (
                  <button
                    key={entry.id}
                    type="button"
                    className={`palette-item ${cursor === index ? "active" : ""}`}
                    onMouseEnter={() => setCursor(index)}
                    onClick={() => choose(index)}
                  >
                    <span className="swatch" style={{ "--swatch": `var(--c-${entry.color})` } as React.CSSProperties} />
                    {entry.title}
                    <span className="meta">
                      <Icon size={11} style={{ verticalAlign: "-1px" }} /> {formatDayShort(entry.date)}
                    </span>
                  </button>
                );
              })}
              {query.trim() && filteredActions.length === 0 && results.length === 0 && (
                <div className="palette-empty">No matches for “{query.trim()}”</div>
              )}
            </div>
            <div className="palette-foot">
              <span><kbd>↑↓</kbd> navigate</span>
              <span><kbd>↵</kbd> select</span>
              <span><kbd>esc</kbd> close</span>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

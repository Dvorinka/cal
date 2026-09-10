import { AnimatePresence, motion } from "framer-motion";
import { Check, Link2, Pencil, RotateCcw, Timer, Trash2 } from "lucide-react";
import { useEffect, useRef } from "react";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function ContextMenu() {
  const menu = useUi((state) => state.contextMenu);
  const setMenu = useUi((state) => state.setContextMenu);
  const openEdit = useUi((state) => state.openEdit);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const deleteEntry = usePlanner((state) => state.deleteEntry);
  const api = usePlanner((state) => state.api);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menu) return;
    const close = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setMenu(undefined);
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMenu(undefined);
    };
    window.addEventListener("mousedown", close);
    window.addEventListener("keydown", escape);
    window.addEventListener("blur", () => setMenu(undefined), { once: true });
    return () => {
      window.removeEventListener("mousedown", close);
      window.removeEventListener("keydown", escape);
    };
  }, [menu, setMenu]);

  const entry = menu?.entry;
  const isLink = entry?.type === "link" && Boolean(entry.linkUrl);

  return (
    <AnimatePresence>
      {menu && entry && (
        <motion.div
          ref={ref}
          className="context-menu"
          style={{
            left: Math.min(menu.x, window.innerWidth - 200),
            top: Math.min(menu.y, window.innerHeight - 180),
          }}
          initial={{ opacity: 0, scale: 0.96, y: -4 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.97 }}
          transition={{ duration: 0.12, ease: "easeOut" }}
        >
          <button
            type="button"
            onClick={() => {
              setMenu(undefined);
              openEdit(entry);
            }}
          >
            <Pencil size={14} /> Edit
          </button>
          {entry.type === "task" && (
            <button
              type="button"
              onClick={() => {
                void updateEntry(entry.id, { completed: !entry.completed });
                setMenu(undefined);
              }}
            >
              {entry.completed ? <RotateCcw size={14} /> : <Check size={14} />}
              {entry.completed ? "Reopen" : "Mark complete"}
            </button>
          )}
          {entry.type === "task" && (
            <button
              type="button"
              onClick={() => {
                void api.startTimer(entry.id).then(() => setMenu(undefined)).catch(() => setMenu(undefined));
              }}
            >
              <Timer size={14} /> Start timer
            </button>
          )}
          {isLink && (
            <button
              type="button"
              onClick={() => {
                window.open(entry.linkUrl, "_blank", "noopener,noreferrer");
                setMenu(undefined);
              }}
            >
              <Link2 size={14} /> Open link
            </button>
          )}
          <div className="sep" />
          <button
            type="button"
            className="danger"
            onClick={() => {
              void deleteEntry(entry.id);
              setMenu(undefined);
            }}
          >
            <Trash2 size={14} /> Delete
          </button>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

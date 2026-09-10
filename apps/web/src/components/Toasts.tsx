import { AnimatePresence, motion } from "framer-motion";
import { usePlanner } from "../stores/planner";

export function Toasts() {
  const toasts = usePlanner((state) => state.toasts);
  const dismiss = usePlanner((state) => state.dismissToast);

  return (
    <div className="toasts" role="status" aria-live="polite">
      <AnimatePresence initial={false}>
        {toasts.map((toast) => (
          <motion.div
            key={toast.id}
            className="toast"
            initial={{ opacity: 0, y: 12, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.97 }}
            transition={{ type: "spring", duration: 0.35, bounce: 0 }}
          >
            <span>{toast.message}</span>
            {toast.action && (
              <button
                type="button"
                onClick={() => {
                  toast.action?.run();
                  dismiss(toast.id);
                }}
              >
                {toast.action.label}
              </button>
            )}
            <button type="button" onClick={() => dismiss(toast.id)} aria-label="Dismiss">
              ✕
            </button>
          </motion.div>
        ))}
      </AnimatePresence>
    </div>
  );
}

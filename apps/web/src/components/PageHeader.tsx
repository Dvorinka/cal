import { Menu } from "lucide-react";
import type { ReactNode } from "react";
import { useUi } from "../stores/ui";

export function PageHeader({ title, sub, children }: { title: string; sub?: string; children?: ReactNode }) {
  const toggleSidebar = useUi((state) => state.toggleSidebar);
  return (
    <header className="topbar page-head">
      <button type="button" className="icon-btn menu-btn" aria-label="Menu" onClick={toggleSidebar}>
        <Menu size={18} />
      </button>
      <div>
        <h1>{title}</h1>
        {sub && <span className="page-sub">{sub}</span>}
      </div>
      <span className="spacer" />
      {children}
    </header>
  );
}

import { ExternalLink, Link2, Plus } from "lucide-react";
import { useEffect, useMemo } from "react";
import { PageHeader } from "../components/PageHeader";
import { linkDomain } from "../lib/entries";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function LinksPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);

  useEffect(() => {
    void loadEntries({});
  }, [loadEntries]);

  const links = useMemo(
    () =>
      entries
        .filter((e) => e.type === "link")
        .sort((a, b) => b.date.localeCompare(a.date) || b.createdAt.localeCompare(a.createdAt)),
    [entries],
  );

  return (
    <>
      <PageHeader title="Links" sub={`${links.length} saved`}>
        <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
          <Plus size={14} strokeWidth={2.5} /> Save link
        </button>
      </PageHeader>
      <div className="page-scroll">
        {links.length === 0 ? (
          <div className="empty-hint">
            <strong>No links saved</strong>
            <span>Save a link and it lands on that day — a quiet reading list.</span>
            <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
              <Plus size={14} /> Save one
            </button>
          </div>
        ) : (
          <ul className="link-list">
            {links.map((link) => {
              const domain = linkDomain(link.linkUrl);
              return (
                <li key={link.id} className={`link-row color-${link.color}`}>
                  <span className="link-mark">{domain ? domain[0].toUpperCase() : <Link2 size={14} />}</span>
                  <button type="button" className="row-title" onClick={() => openEdit(link)}>
                    {link.title}
                    {domain && <span className="link-domain">{domain}</span>}
                  </button>
                  <span className="meta-chip date">{formatDayShort(link.date)}</span>
                  {link.linkUrl && (
                    <a
                      className="icon-btn"
                      href={link.linkUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      aria-label="Open link"
                    >
                      <ExternalLink size={15} />
                    </a>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </>
  );
}

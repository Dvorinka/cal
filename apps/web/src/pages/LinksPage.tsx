import { Check, ExternalLink, Link2, Play, Plus, Square } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { linkDomain } from "../lib/entries";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function LinksPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const [grid, setGrid] = useState(true);

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
        <button
          type="button"
          className="btn btn-secondary btn-xs"
          onClick={() => setGrid((v) => !v)}
        >
          {grid ? "List" : "Cards"}
        </button>
        <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
          <Plus size={14} strokeWidth={2.5} /> Save link
        </button>
      </PageHeader>
      <div className="page-scroll">
        {links.length === 0 ? (
          <div className="empty-hint">
            <strong>No links saved</strong>
            <span>Save a link — previews and YouTube thumbnails fetch themselves.</span>
            <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
              <Plus size={14} /> Save one
            </button>
          </div>
        ) : grid ? (
          <div className="link-grid">
            {links.map((link) => {
              const domain = linkDomain(link.linkUrl);
              return (
                <article key={link.id} className={`link-card ${link.watched ? "watched" : ""}`}>
                  {link.linkImage ? (
                    <a href={link.linkUrl} target="_blank" rel="noopener noreferrer" className="link-thumb">
                      <img src={link.linkImage} alt="" loading="lazy" />
                      {link.linkVideoId && <span className="link-play"><Play size={20} fill="currentColor" /></span>}
                    </a>
                  ) : (
                    <a href={link.linkUrl} target="_blank" rel="noopener noreferrer" className="link-thumb link-thumb-plain">
                      {link.linkFavicon ? <img src={link.linkFavicon} alt="" className="link-favicon" /> : <Link2 size={20} />}
                    </a>
                  )}
                  <div className="link-card-body">
                    <button type="button" className="link-card-title" onClick={() => openEdit(link)}>
                      {link.title}
                    </button>
                    {link.linkDesc && <p className="link-desc">{link.linkDesc}</p>}
                    <div className="link-card-meta">
                      <span className="link-domain">{domain}</span>
                      <span className="meta-chip date">{formatDayShort(link.date)}</span>
                      {link.linkVideoId && (
                        <button
                          type="button"
                          className="icon-btn"
                          aria-label={link.watched ? "Mark unwatched" : "Mark watched"}
                          onClick={() => void updateEntry(link.id, { watched: !link.watched })}
                        >
                          {link.watched ? <Check size={14} /> : <Square size={14} />}
                        </button>
                      )}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        ) : (
          <ul className="link-list">
            {links.map((link) => {
              const domain = linkDomain(link.linkUrl);
              return (
                <li key={link.id} className={`link-row color-${link.color}`}>
                  {link.linkFavicon ? (
                    <img src={link.linkFavicon} alt="" className="link-favicon" />
                  ) : (
                    <span className="link-mark">{domain ? domain[0].toUpperCase() : <Link2 size={14} />}</span>
                  )}
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

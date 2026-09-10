import { Check, ExternalLink, Link2, Play, Plus, Youtube } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { linkDomain } from "../lib/entries";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

type Filter = "all" | "videos" | "articles";

export function LinksPage() {
  const entries = usePlanner((state) => state.entries);
  const loadEntries = usePlanner((state) => state.loadEntries);
  const updateEntry = usePlanner((state) => state.updateEntry);
  const openCreate = useUi((state) => state.openCreate);
  const openEdit = useUi((state) => state.openEdit);
  const [grid, setGrid] = useState(true);
  const [filter, setFilter] = useState<Filter>("all");

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

  const visible = useMemo(() => {
    if (filter === "videos") return links.filter((l) => l.linkVideoId);
    if (filter === "articles") return links.filter((l) => !l.linkVideoId);
    return links;
  }, [links, filter]);

  const videoCount = links.filter((l) => l.linkVideoId).length;

  return (
    <>
      <PageHeader title="Links" sub={`${links.length} saved · ${videoCount} video${videoCount === 1 ? "" : "s"}`}>
        {(["all", "videos", "articles"] as Filter[]).map((f) => (
          <button
            key={f}
            type="button"
            className={`btn btn-secondary btn-xs ${filter === f ? "on" : ""}`}
            onClick={() => setFilter(f)}
          >
            {f === "videos" ? "Videos" : f === "articles" ? "Articles" : "All"}
          </button>
        ))}
        <button type="button" className="btn btn-secondary btn-xs" onClick={() => setGrid((v) => !v)}>
          {grid ? "List" : "Cards"}
        </button>
        <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
          <Plus size={14} strokeWidth={2.5} /> Save link
        </button>
      </PageHeader>
      <div className="page-scroll">
        {visible.length === 0 ? (
          <div className="empty-hint">
            <strong>{filter === "all" ? "No links saved" : `No ${filter}`}</strong>
            <span>Save a link — previews and YouTube thumbnails fetch themselves.</span>
            <button type="button" className="btn btn-primary" onClick={() => openCreate(todayIso())}>
              <Plus size={14} /> Save one
            </button>
          </div>
        ) : grid ? (
          <div className="link-grid">
            {visible.map((link) => {
              const domain = linkDomain(link.linkUrl);
              const isVideo = !!link.linkVideoId;
              return (
                <article key={link.id} className={`link-card ${link.watched ? "watched" : ""}`}>
                  <a href={link.linkUrl} target="_blank" rel="noopener noreferrer" className="link-thumb">
                    {link.linkImage ? (
                      <img src={link.linkImage} alt="" loading="lazy" />
                    ) : (
                      <span className="link-thumb-plain">
                        {link.linkFavicon ? <img src={link.linkFavicon} alt="" className="link-favicon" /> : <Link2 size={20} />}
                      </span>
                    )}
                    {isVideo && (
                      <span className="link-play">
                        <span className="link-play-btn"><Play size={18} fill="currentColor" /></span>
                      </span>
                    )}
                    {isVideo && (
                      <button
                        type="button"
                        className={`link-watched ${link.watched ? "on" : ""}`}
                        aria-label={link.watched ? "Mark unwatched" : "Mark watched"}
                        onClick={(e) => {
                          e.preventDefault();
                          void updateEntry(link.id, { watched: !link.watched });
                        }}
                      >
                        <Check size={13} />
                        {link.watched && <span>Watched</span>}
                      </button>
                    )}
                  </a>
                  <div className="link-card-body">
                    <button type="button" className="link-card-title" onClick={() => openEdit(link)}>
                      {link.title}
                    </button>
                    {link.linkDesc && (
                      <p className="link-desc">
                        {isVideo && <Youtube size={12} className="link-yt" />}
                        {link.linkDesc}
                      </p>
                    )}
                    <div className="link-card-meta">
                      <span className="link-domain">{domain}</span>
                      <span className="meta-chip date">{formatDayShort(link.date)}</span>
                      {link.tags?.slice(0, 2).map((t) => (
                        <span key={t} className="meta-chip">#{t}</span>
                      ))}
                    </div>
                  </div>
                </article>
              );
            })}
          </div>
        ) : (
          <ul className="link-list">
            {visible.map((link) => {
              const domain = linkDomain(link.linkUrl);
              return (
                <li key={link.id} className={`link-row color-${link.color} ${link.watched ? "watched" : ""}`}>
                  {link.linkFavicon ? (
                    <img src={link.linkFavicon} alt="" className="link-favicon" />
                  ) : (
                    <span className="link-mark">{domain ? domain[0].toUpperCase() : <Link2 size={14} />}</span>
                  )}
                  <button type="button" className="row-title" onClick={() => openEdit(link)}>
                    {link.title}
                    {domain && <span className="link-domain">{domain}</span>}
                  </button>
                  {link.linkVideoId && <Youtube size={13} style={{ color: "var(--text-3)" }} />}
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

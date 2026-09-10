// SharedBoardPage — public read-only board at /board/:token. No auth.

import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { CalApi } from "@cal/api-client";

const api = new CalApi("/api");

export function SharedBoardPage() {
  const { token } = useParams();
  const [board, setBoard] = useState<Awaited<ReturnType<typeof api.sharedBoard>>>();
  const [err, setErr] = useState(false);

  useEffect(() => {
    if (token) void api.sharedBoard(token).then(setBoard).catch(() => setErr(true));
  }, [token]);

  if (err) return <div className="shared-board"><h1>Not found</h1><p>This board link is revoked or wrong.</p></div>;
  if (!board) return <div className="shared-board"><p>Loading…</p></div>;

  const byCol = new Map<string, typeof board.cards>();
  for (const card of board.cards) {
    const k = card.columnId ?? "";
    byCol.set(k, [...(byCol.get(k) ?? []), card]);
  }

  return (
    <div className="shared-board">
      <header className="shared-head">
        <h1>{board.name}</h1>
        {board.description && <p className="shared-desc">{board.description}</p>}
        {board.targetDate && <p className="shared-target">Target: {board.targetDate}</p>}
      </header>
      <div className="kanban shared-kanban">
        {board.columns.map((col) => (
          <section key={col.id} className="kanban-col">
            <header className="kanban-head">
              <span className="kanban-name">{col.name}</span>
              <span className="kanban-count">{(byCol.get(col.id) ?? []).length}</span>
            </header>
            <div className="kanban-cards">
              {(byCol.get(col.id) ?? []).map((card, i) => (
                <div key={i} className={`kanban-card ${card.completed ? "done" : ""}`}>
                  <span className="kanban-card-title">{card.title}</span>
                  <div className="kanban-card-meta">
                    {card.tags.slice(0, 3).map((t) => (
                      <span key={t} className="stream-tag">#{t}</span>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

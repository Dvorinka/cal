// Boards — kanban project management. Cards are ordinary task entries
// (board_id/column_id/position): they complete via the editor, appear on the
// calendar, and in Today. Drag between columns, add cards inline.

import type { Board, BoardColumn, Entry } from "@cal/api-client";
import { Check, Link2, ListOrdered, Plus, Trash2, Trello } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { formatDayShort, todayIso } from "../lib/date";
import { usePlanner } from "../stores/planner";
import { useUi } from "../stores/ui";

export function BoardsPage() {
  const { boardId } = useParams();
  const api = usePlanner((s) => s.api);
  const toast = usePlanner((s) => s.toast);
  const openEdit = useUi((s) => s.openEdit);
  const createEntry = usePlanner((s) => s.createEntry);

  const [boards, setBoards] = useState<Board[]>([]);
  const [board, setBoard] = useState<Board>();
  const [columns, setColumns] = useState<BoardColumn[]>([]);
  const [cards, setCards] = useState<Entry[]>([]);
  const [newBoard, setNewBoard] = useState("");
  const [template, setTemplate] = useState("kanban");
  const [newCol, setNewCol] = useState("");
  const [newCard, setNewCard] = useState<Record<string, string>>({});
  const [dragId, setDragId] = useState<string>();
  const [overCol, setOverCol] = useState<string>();

  const loadBoards = useCallback(() => void api.boards().then(setBoards).catch(() => {}), [api]);
  const loadView = useCallback(
    (id: string) => {
      void api.boardView(id).then((v) => {
        setColumns(v.columns);
        setCards(v.cards);
      }).catch(() => {});
    },
    [api],
  );

  useEffect(loadBoards, [loadBoards]);
  useEffect(() => {
    if (boardId) {
      void loadView(boardId);
      void api.boards().then((bs) => setBoard(bs.find((b) => b.id === boardId))).catch(() => {});
    } else {
      setBoard(undefined);
      setColumns([]);
      setCards([]);
    }
  }, [boardId, api, loadView]);

  const byColumn = useMemo(() => {
    const m = new Map<string, Entry[]>();
    for (const card of cards) {
      const key = card.columnId ?? "";
      m.set(key, [...(m.get(key) ?? []), card]);
    }
    for (const list of m.values()) list.sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
    return m;
  }, [cards]);

  async function addBoard() {
    const name = newBoard.trim();
    if (!name) return;
    await api.createBoard(name, undefined, template).then((b) => {
      setBoards((bs) => [...bs, b]);
      setNewBoard("");
      toast("Board created");
    }).catch(() => toast("Failed"));
  }

  async function addColumn() {
    const name = newCol.trim();
    if (!name || !boardId) return;
    await api.createColumn(boardId, name).then(() => {
      setNewCol("");
      loadView(boardId);
    }).catch(() => toast("Failed"));
  }

  async function addCard(columnId: string) {
    const title = (newCard[columnId] ?? "").trim();
    if (!title || !boardId) return;
    const created = await createEntry({ title, type: "task", date: todayIso(), boardId, columnId });
    if (created) {
      setNewCard((m) => ({ ...m, [columnId]: "" }));
      loadView(boardId);
    }
  }

  // Fractional position: drop between neighbours without rewriting the column.
  function drop(cardId: string, columnId: string, beforeCard?: Entry) {
    const list = byColumn.get(columnId) ?? [];
    const idx = beforeCard ? list.findIndex((c) => c.id === beforeCard.id) : list.length;
    const prev = idx > 0 ? list[idx - 1].position : undefined;
    const next = idx < list.length ? list[idx].position : undefined;
    const position = prev !== undefined && next !== undefined ? (prev + next) / 2 : prev !== undefined ? prev + 1024 : next !== undefined ? next - 1024 : 1024;
    void api.moveCard(cardId, columnId, position).then(() => boardId && loadView(boardId)).catch(() => toast("Move failed"));
  }

  if (!boardId) {
    return (
      <>
        <PageHeader title="Boards" sub="Kanban for projects — cards are real tasks, so they show up everywhere" />
        <div className="page-scroll">
          <div className="board-add">
            <input
              className="input"
              placeholder="New board name…"
              value={newBoard}
              onChange={(e) => setNewBoard(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && void addBoard()}
              aria-label="New board name"
            />
            <select
              className="select"
              value={template}
              onChange={(e) => setTemplate(e.target.value)}
              aria-label="Board template"
            >
              <option value="blank">Blank</option>
              <option value="kanban">Kanban</option>
              <option value="sprint">Sprint</option>
              <option value="bugs">Bug tracker</option>
            </select>
            <button type="button" className="btn btn-primary" onClick={() => void addBoard()}>
              <Plus size={14} /> Create board
            </button>
          </div>
          {boards.length === 0 ? (
            <div className="empty-hint">
              <strong>No boards yet</strong>
              <span>A board groups tasks into columns — Todo, Doing, Done or your own flow.</span>
            </div>
          ) : (
            <div className="board-list">
              {boards.map((b) => (
                <Link key={b.id} to={`/boards/${b.id}`} className="board-tile">
                  <Trello size={18} strokeWidth={1.8} />
                  <span className="board-name">{b.name}</span>
                  <button
                    type="button"
                    className="icon-btn"
                    aria-label={`Delete ${b.name}`}
                    onClick={(e) => {
                      e.preventDefault();
                      void api.deleteBoard(b.id).then(loadBoards);
                    }}
                  >
                    <Trash2 size={14} />
                  </button>
                </Link>
              ))}
            </div>
          )}
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader
        title={board?.name ?? "Board"}
        sub={board?.targetDate ? `target ${board.targetDate} — drag cards between columns` : "drag cards between columns — complete them like any task"}
      >
        <button
          type="button"
          className="btn btn-secondary btn-xs"
          onClick={() => {
            const d = window.prompt("Board description", board?.description ?? "");
            if (d === null) return;
            const t = window.prompt("Target date (YYYY-MM-DD, empty clears)", board?.targetDate ?? "");
            if (t === null) return;
            void api.updateBoard(boardId, { description: d, targetDate: t.trim() })
              .then(() => api.boards().then((bs) => setBoard(bs.find((b) => b.id === boardId))));
          }}
        >
          Edit board
        </button>
        <button
          type="button"
          className="btn btn-secondary btn-xs"
          onClick={() =>
            void api.shareBoard(boardId, true).then(({ shareToken }) => {
              const url = `${window.location.origin}/board/${shareToken}`;
              void navigator.clipboard.writeText(url);
              toast("Public link copied — anyone with it can view the board");
            })
          }
        >
          <Link2 size={12} /> Share
        </button>
        <Link to="/boards" className="btn btn-secondary btn-xs">All boards</Link>
      </PageHeader>
      {board?.description && <p className="board-desc">{board.description}</p>}
      <div className="kanban">
        {columns.map((col) => (
          <section
            key={col.id}
            className={`kanban-col ${overCol === col.id ? "over" : ""}`}
            onDragOver={(e) => {
              e.preventDefault();
              setOverCol(col.id);
            }}
            onDragLeave={() => setOverCol((c) => (c === col.id ? undefined : c))}
            onDrop={(e) => {
              e.preventDefault();
              setOverCol(undefined);
              if (dragId) drop(dragId, col.id);
              setDragId(undefined);
            }}
            aria-label={col.name}
          >
            <header className="kanban-head">
              <span className="kanban-name">{col.name}</span>
              <span
                className={`kanban-count ${col.wipLimit && (byColumn.get(col.id) ?? []).length > col.wipLimit ? "over" : ""}`}
                title={col.wipLimit ? `WIP limit ${col.wipLimit}` : undefined}
              >
                {(byColumn.get(col.id) ?? []).length}
                {col.wipLimit ? `/${col.wipLimit}` : ""}
              </span>
              <button
                type="button"
                className="icon-btn"
                aria-label={`WIP limit for ${col.name}`}
                title="Set WIP limit (0 clears)"
                onClick={() => {
                  const v = window.prompt(`WIP limit for "${col.name}" (empty clears)`, String(col.wipLimit ?? ""));
                  if (v === null) return;
                  const n = v.trim() === "" ? 0 : parseInt(v, 10);
                  if (Number.isNaN(n) || n < 0) return;
                  void api.updateColumn(col.id, { wipLimit: n }).then(() => boardId && loadView(boardId));
                }}
              >
                <ListOrdered size={12} />
              </button>
              <button
                type="button"
                className="icon-btn"
                aria-label={`Delete column ${col.name}`}
                onClick={() => void api.deleteColumn(col.id).then(() => boardId && loadView(boardId))}
              >
                <Trash2 size={12} />
              </button>
            </header>
            <div className="kanban-cards">
              {(byColumn.get(col.id) ?? []).map((card) => (
                <div
                  key={card.id}
                  className={`kanban-card color-${card.color} ${card.completed ? "done" : ""}`}
                  draggable
                  onDragStart={() => setDragId(card.id)}
                  onDragEnd={() => setDragId(undefined)}
                  onDragOver={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                  onDrop={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    if (dragId && dragId !== card.id) drop(dragId, col.id, card);
                    setDragId(undefined);
                  }}
                >
                  <button type="button" className="kanban-card-title" onClick={() => openEdit(card)}>
                    {card.title}
                  </button>
                  <div className="kanban-card-meta">
                    <CardDue date={card.date} done={card.completed} />
                    <CardChecklist content={card.content} />
                    {card.tags.slice(0, 3).map((t) => (
                      <span key={t} className="stream-tag">#{t}</span>
                    ))}
                    {card.completed && <Check size={12} className="kanban-done" aria-label="Done" />}
                  </div>
                </div>
              ))}
            </div>
            <div className="kanban-add">
              <input
                className="kanban-add-input"
                placeholder="+ card"
                value={newCard[col.id] ?? ""}
                onChange={(e) => setNewCard((m) => ({ ...m, [col.id]: e.target.value }))}
                onKeyDown={(e) => e.key === "Enter" && void addCard(col.id)}
                aria-label={`Add card to ${col.name}`}
              />
            </div>
          </section>
        ))}
        <div className="kanban-col kanban-col-new">
          <input
            className="input"
            placeholder="+ Column"
            value={newCol}
            onChange={(e) => setNewCol(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && void addColumn()}
            aria-label="New column name"
          />
        </div>
      </div>
    </>
  );
}

// CardDue — date chip on the card face; red when past due and not done.
function CardDue({ date, done }: { date: string; done: boolean }) {
  if (!date) return null;
  const overdue = !done && date < todayIso();
  const today = date === todayIso();
  return (
    <span className={`card-due ${overdue ? "over" : ""} ${today ? "today" : ""}`}>
      {formatDayShort(date)}
    </span>
  );
}

// CardChecklist — "2/5" chip counting `- [ ]` / `- [x]` lines in content.
function CardChecklist({ content }: { content?: string }) {
  if (!content) return null;
  const total = (content.match(/^\s*[-*]\s+\[[ x]\]/gim) ?? []).length;
  const done = (content.match(/^\s*[-*]\s+\[x\]/gim) ?? []).length;
  if (!total) return null;
  return (
    <span className={`card-checks ${done === total ? "all" : ""}`}>
      <Check size={10} /> {done}/{total}
    </span>
  );
}

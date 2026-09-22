import type { ShoppingItem, ShoppingList, ShoppingSection, ShoppingSuggestion } from "@cal/api-client";
import { Check, CheckCheck, CircleHelp, Pencil, Plus, ShoppingBasket, Trash2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { reportErr, usePlanner } from "../stores/planner";

// Shopping — lists → sections → items, inspired by Koffan. Page-local state;
// the detail payload (sections + items) reloads after mutations that would
// reorder rows (check-all, clear), while single-item edits patch in place.

export function ShoppingPage() {
  const api = usePlanner((s) => s.api);
  const toast = usePlanner((s) => s.toast);

  const [lists, setLists] = useState<ShoppingList[]>([]);
  const [activeId, setActiveId] = useState("");
  const [sections, setSections] = useState<ShoppingSection[]>([]);
  const [items, setItems] = useState<ShoppingItem[]>([]);
  const [newList, setNewList] = useState("");
  const [newItem, setNewItem] = useState("");
  const [newQty, setNewQty] = useState("");
  const [newSection, setNewSection] = useState("");
  const [addingSection, setAddingSection] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const [renameVal, setRenameVal] = useState("");
  const [suggestions, setSuggestions] = useState<ShoppingSuggestion[]>([]);
  const [showSug, setShowSug] = useState(false);
  const sugTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  // Section remembered from the picked suggestion — applied on submit.
  const pendingSection = useRef<string | undefined>(undefined);

  const active = lists.find((l) => l.id === activeId);

  const loadLists = useCallback(async () => {
    const ls = await api.shoppingLists();
    setLists(ls);
    setActiveId((cur) => (cur && ls.some((l) => l.id === cur) ? cur : ls[0]?.id ?? ""));
  }, [api]);

  const loadDetail = useCallback(async (id: string) => {
    if (!id) { setSections([]); setItems([]); return; }
    const d = await api.shoppingList(id);
    setSections(d.sections);
    setItems(d.items);
  }, [api]);

  useEffect(() => { void loadLists().catch(reportErr("Could not load lists")); }, [loadLists]);
  useEffect(() => { void loadDetail(activeId).catch(reportErr("Could not load list")); }, [activeId, loadDetail]);

  // Suggestions debounce — history autocomplete on the add row.
  useEffect(() => {
    clearTimeout(sugTimer.current);
    if (newItem.trim().length < 2) { setSuggestions([]); return; }
    sugTimer.current = setTimeout(() => {
      void api.shoppingSuggest(newItem.trim()).then(setSuggestions).catch(() => setSuggestions([]));
    }, 180);
    return () => clearTimeout(sugTimer.current);
  }, [newItem, api]);

  async function addList() {
    const name = newList.trim();
    if (!name) return;
    try {
      const l = await api.shoppingCreateList({ name });
      setLists((cur) => [...cur, l]);
      setActiveId(l.id);
      setNewList("");
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function addItem(sectionId?: string) {
    const name = newItem.trim();
    if (!name || !activeId) return;
    try {
      const it = await api.shoppingCreateItem(activeId, { name, quantity: newQty.trim(), sectionId });
      setItems((cur) => [...cur, it]);
      setLists((cur) => cur.map((l) => l.id === activeId ? { ...l, items: l.items + 1, open: l.open + 1 } : l));
      setNewItem("");
      setNewQty("");
      setSuggestions([]);
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function patchItem(it: ShoppingItem, patch: Parameters<typeof api.shoppingUpdateItem>[1]) {
    try {
      const next = await api.shoppingUpdateItem(it.id, patch);
      setItems((cur) => cur.map((x) => (x.id === it.id ? next : x)));
      if (patch.checked !== undefined) {
        setLists((cur) => cur.map((l) => l.id === it.listId
          ? { ...l, open: l.open + (patch.checked ? -1 : 1) } : l));
      }
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function removeItem(it: ShoppingItem) {
    try {
      await api.shoppingDeleteItem(it.id);
      setItems((cur) => cur.filter((x) => x.id !== it.id));
      setLists((cur) => cur.map((l) => l.id === it.listId
        ? { ...l, items: l.items - 1, open: l.open - (it.checked ? 0 : 1) } : l));
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function addSection() {
    const name = newSection.trim();
    if (!name || !activeId) return;
    try {
      const sc = await api.shoppingCreateSection(activeId, name);
      setSections((cur) => [...cur, sc]);
      setNewSection("");
      setAddingSection(false);
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function renameSection(sc: ShoppingSection) {
    const name = window.prompt("Section name", sc.name)?.trim();
    if (!name || name === sc.name) return;
    try {
      await api.shoppingUpdateSection(sc.id, name);
      setSections((cur) => cur.map((x) => (x.id === sc.id ? { ...x, name } : x)));
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function deleteSection(sc: ShoppingSection) {
    if (!window.confirm(`Delete section "${sc.name}"? Its items move to the ungrouped block.`)) return;
    try {
      await api.shoppingDeleteSection(sc.id);
      await loadDetail(activeId);
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function checkSection(sc: ShoppingSection, checked: boolean) {
    try {
      await api.shoppingCheckSection(sc.id, checked);
      const next = items.map((x) => (x.sectionId === sc.id ? { ...x, checked } : x));
      setItems(next);
      setLists((cur) => cur.map((l) => l.id === activeId
        ? { ...l, open: next.filter((i) => !i.checked).length } : l));
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function clearPurchased() {
    if (!activeId) return;
    try {
      const { removed } = await api.shoppingClearPurchased(activeId);
      if (removed > 0) {
        setItems((cur) => cur.filter((x) => !x.checked));
        await loadLists();
      }
      toast(removed ? `Cleared ${removed} item${removed === 1 ? "" : "s"}` : "Nothing to clear");
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function renameList() {
    if (!active) return;
    const name = renameVal.trim();
    setRenaming(false);
    if (!name || name === active.name) return;
    try {
      const next = await api.shoppingUpdateList(active.id, { name, icon: active.icon });
      setLists((cur) => cur.map((l) => (l.id === next.id ? next : l)));
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  async function deleteList() {
    if (!active || !window.confirm(`Delete "${active.name}" and its ${active.items} items?`)) return;
    try {
      await api.shoppingDeleteList(active.id);
      setActiveId("");
      await loadLists();
    } catch (e) { toast(e instanceof Error ? e.message : "Failed"); }
  }

  function pickSuggestion(sg: ShoppingSuggestion) {
    setNewItem(sg.name);
    setShowSug(false);
    // The suggestion's last-used section is applied on submit if it still
    // exists on this list.
    const sc = sections.find((s) => s.name === sg.section);
    if (sc) pendingSection.current = sc.id;
  }

  function renderItems(list: ShoppingItem[]) {
    return list.map((it) => (
      <div key={it.id} className={`shop-item ${it.checked ? "done" : ""}`}>
        <button
          type="button"
          className={`shop-check ${it.checked ? "on" : ""}`}
          aria-label={it.checked ? `Uncheck ${it.name}` : `Check ${it.name}`}
          onClick={() => void patchItem(it, { checked: !it.checked })}
        >
          {it.checked && <Check size={13} />}
        </button>
        <div className="shop-item-main">
          <span className="shop-item-name">
            {it.name}
            {it.quantity && <span className="shop-qty">{it.quantity}</span>}
          </span>
          {it.note && <span className="shop-item-note">{it.note}</span>}
        </div>
        <button
          type="button"
          className={`icon-btn ${it.uncertain ? "shop-uncertain" : ""}`}
          title={it.uncertain ? "Marked as can't find" : "Can't find it?"}
          aria-label={`Toggle uncertain on ${it.name}`}
          onClick={() => void patchItem(it, { uncertain: !it.uncertain })}
        >
          <CircleHelp size={15} />
        </button>
        <button type="button" className="icon-btn" aria-label={`Delete ${it.name}`} onClick={() => void removeItem(it)}>
          <Trash2 size={14} />
        </button>
      </div>
    ));
  }

  const unsectioned = items.filter((it) => !it.sectionId);

  return (
    <>
      <PageHeader title="Shopping" sub={active ? `${active.open} to buy · ${active.items} total` : "Shared-style shopping lists"} />
      <div className="panel-wrap">
        <div className="shop-chips">
          {lists.map((l) => (
            <button
              key={l.id}
              type="button"
              className={`shop-chip ${l.id === activeId ? "on" : ""}`}
              onClick={() => setActiveId(l.id)}
            >
              {l.icon && <span>{l.icon}</span>}
              {l.name}
              <i>{l.open}</i>
            </button>
          ))}
          <input
            className="input shop-newlist"
            placeholder="+ New list…"
            aria-label="New list name"
            value={newList}
            onChange={(e) => setNewList(e.target.value)}
            onKeyDown={(e) => { if (e.key === "Enter") void addList(); }}
          />
        </div>

        {!active && lists.length === 0 && (
          <section className="panel">
            <p className="panel-note">
              <ShoppingBasket size={15} style={{ verticalAlign: "-3px" }} /> No lists yet —
              type a name above (Groceries, Pharmacy, Hardware…) and press Enter.
            </p>
          </section>
        )}

        {active && (
          <section className="panel">
            <div className="shop-head">
              {renaming ? (
                <input
                  className="input"
                  style={{ maxWidth: 240 }}
                  value={renameVal}
                  autoFocus
                  onChange={(e) => setRenameVal(e.target.value)}
                  onBlur={() => void renameList()}
                  onKeyDown={(e) => { if (e.key === "Enter") void renameList(); if (e.key === "Escape") setRenaming(false); }}
                />
              ) : (
                <h3>{active.icon} {active.name}</h3>
              )}
              <span className="spacer" />
              <button type="button" className="icon-btn" aria-label="Rename list"
                onClick={() => { setRenameVal(active.name); setRenaming(true); }}>
                <Pencil size={14} />
              </button>
              <button type="button" className="btn btn-secondary" onClick={() => void clearPurchased()}>
                Clear purchased
              </button>
              <button type="button" className="icon-btn" aria-label="Delete list" onClick={() => void deleteList()}>
                <Trash2 size={14} />
              </button>
            </div>

            <div className="shop-add">
              <div className="shop-add-name">
                <input
                  className="input"
                  placeholder="Add an item…"
                  value={newItem}
                  onChange={(e) => { setNewItem(e.target.value); setShowSug(true); }}
                  onFocus={() => setShowSug(true)}
                  onBlur={() => setTimeout(() => setShowSug(false), 150)}
                  onKeyDown={(e) => { if (e.key === "Enter") { void addItem(pendingSection.current); pendingSection.current = undefined; } }}
                />
                {showSug && suggestions.length > 0 && (
                  <div className="shop-suggest">
                    {suggestions.map((sg) => (
                      <button key={sg.name} type="button" onMouseDown={(e) => e.preventDefault()} onClick={() => pickSuggestion(sg)}>
                        {sg.name}
                        {sg.section && <i>{sg.section}</i>}
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <input
                className="input shop-qty-input"
                placeholder="Qty"
                value={newQty}
                onChange={(e) => setNewQty(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") { void addItem(pendingSection.current); pendingSection.current = undefined; } }}
              />
              <button type="button" className="btn btn-primary" disabled={!newItem.trim()}
                onClick={() => { void addItem(pendingSection.current); pendingSection.current = undefined; }}>
                <Plus size={14} /> Add
              </button>
            </div>

            {unsectioned.length > 0 && (
              <div className="shop-group">{renderItems(unsectioned)}</div>
            )}

            {sections.map((sc) => {
              const secItems = items.filter((it) => it.sectionId === sc.id);
              const allChecked = secItems.length > 0 && secItems.every((it) => it.checked);
              return (
                <div key={sc.id} className="shop-group">
                  <div className="shop-section-head">
                    <strong>{sc.name}</strong>
                    <span className="panel-note">{secItems.filter((i) => i.checked).length}/{secItems.length}</span>
                    <span className="spacer" />
                    <button
                      type="button"
                      className="icon-btn"
                      title={allChecked ? "Uncheck all" : "Check all"}
                      aria-label={allChecked ? `Uncheck all in ${sc.name}` : `Check all in ${sc.name}`}
                      onClick={() => void checkSection(sc, !allChecked)}
                    >
                      <CheckCheck size={15} />
                    </button>
                    <button type="button" className="icon-btn" aria-label={`Rename ${sc.name}`} onClick={() => void renameSection(sc)}>
                      <Pencil size={13} />
                    </button>
                    <button type="button" className="icon-btn" aria-label={`Delete ${sc.name}`} onClick={() => void deleteSection(sc)}>
                      <Trash2 size={13} />
                    </button>
                  </div>
                  {renderItems(secItems)}
                </div>
              );
            })}

            {addingSection ? (
              <div className="shop-add" style={{ marginTop: 10 }}>
                <input
                  className="input"
                  placeholder="Section name (Dairy, Produce…)"
                  value={newSection}
                  autoFocus
                  onChange={(e) => setNewSection(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter") void addSection(); if (e.key === "Escape") setAddingSection(false); }}
                />
                <button type="button" className="btn btn-secondary" onClick={() => void addSection()}>Add section</button>
              </div>
            ) : (
              <button type="button" className="btn btn-secondary" style={{ marginTop: 10 }} onClick={() => setAddingSection(true)}>
                <Plus size={14} /> Section
              </button>
            )}
          </section>
        )}
      </div>
    </>
  );
}

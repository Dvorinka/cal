// Family tree — a read-only graph over person_links. parent/child edges fix
// generations; partner/sibling sit side by side; friend/coworker/mentor draw
// dashed. Nodes click through to the profile.

import type { PersonRelation } from "@cal/api-client";
import { ArrowLeft, Users } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { usePlanner } from "../stores/planner";

const NODE_W = 150;
const NODE_H = 44;
const GAP_X = 34;
const GAP_Y = 84;

// Edge direction: "to is from's kind" — so parent edges climb a generation
// and child edges drop one.
const GEN_DELTA: Record<string, number> = { parent: -1, child: 1 };
const FAMILY_KINDS = new Set(["parent", "child", "sibling", "partner"]);

interface Node {
  id: string;
  name: string;
  color: string;
  gen: number;
  x: number;
  y: number;
}

export function FamilyTreePage() {
  const api = usePlanner((s) => s.api);
  const people = usePlanner((s) => s.people);
  const loadPeople = usePlanner((s) => s.loadPeople);
  const navigate = useNavigate();
  const [relations, setRelations] = useState<PersonRelation[]>([]);

  useEffect(() => {
    if (people.length === 0) void loadPeople();
  }, [people.length, loadPeople]);
  useEffect(() => {
    void api.allPersonRelations().then(setRelations).catch(() => setRelations([]));
  }, [api]);

  const { nodes, edges, width, height } = useMemo(() => {
    const byId = new Map(people.map((p) => [p.id, p]));
    // Deduplicate — each link arrives twice (once per endpoint).
    const seen = new Set<string>();
    const links = relations.filter((r) => {
      const key = [r.personId, r.otherId, r.kind].sort().join("|");
      if (seen.has(key)) return false;
      seen.add(key);
      return byId.has(r.personId) && byId.has(r.otherId);
    });

    // Generations: iterate parent/child deltas to a fixpoint. Cycles just
    // settle on an arbitrary but consistent row.
    const gen = new Map<string, number>();
    const seed = links[0]?.personId ?? people[0]?.id;
    if (seed) gen.set(seed, 0);
    for (let pass = 0; pass < people.length + 4; pass++) {
      let changed = false;
      for (const l of links) {
        const d = GEN_DELTA[l.kind];
        if (d === undefined) continue;
        const a = gen.get(l.personId);
        const b = gen.get(l.otherId);
        if (a === undefined && b === undefined) continue;
        if (a !== undefined && gen.get(l.otherId) !== a + d) {
          gen.set(l.otherId, a + d);
          changed = true;
        } else if (b !== undefined && gen.get(l.personId) !== b - d) {
          gen.set(l.personId, b - d);
          changed = true;
        }
      }
      if (!changed) break;
    }

    // Row assignment: unlinked people join gen 0 at the end.
    const rows = new Map<number, string[]>();
    for (const p of people) {
      const g = gen.get(p.id) ?? 0;
      rows.set(g, [...(rows.get(g) ?? []), p.id]);
    }
    const gens = [...rows.keys()].sort((a, b) => a - b);
    const maxCols = Math.max(...[...rows.values()].map((r) => r.length), 1);
    const w = maxCols * (NODE_W + GAP_X) + GAP_X;
    const nodes: Node[] = [];
    gens.forEach((g, rowIdx) => {
      const ids = rows.get(g)!;
      const rowW = ids.length * (NODE_W + GAP_X) - GAP_X;
      const startX = (w - rowW) / 2;
      ids.forEach((id, i) => {
        const p = byId.get(id)!;
        nodes.push({ id, name: p.name, color: p.color || "slate", gen: g, x: startX + i * (NODE_W + GAP_X), y: 20 + rowIdx * (NODE_H + GAP_Y) });
      });
    });
    const pos = new Map(nodes.map((n) => [n.id, n]));
    const edges = links
      .map((l) => ({ a: pos.get(l.personId)!, b: pos.get(l.otherId)!, kind: l.kind }))
      .filter((e) => e.a && e.b);
    return { nodes, edges, width: w, height: 20 + gens.length * (NODE_H + GAP_Y) };
  }, [people, relations]);

  return (
    <>
      <PageHeader title="Family tree" sub={`${people.length} people · ${edges.length} links`}>
        <Link className="btn btn-ghost" to="/people">
          <ArrowLeft size={14} /> People
        </Link>
      </PageHeader>
      <div className="page-scroll">
        {nodes.length === 0 ? (
          <div className="empty-hint">
            <strong>No one here yet</strong>
            <span>Add people and link them — parents, partners, siblings — and the tree draws itself.</span>
            <Link className="btn btn-primary" to="/people">
              <Users size={14} /> Open people
            </Link>
          </div>
        ) : (
          <div className="tree-scroll">
            <svg className="family-tree" width={width} height={height} role="img" aria-label="Family tree">
              {edges.map((e, i) => {
                const ax = e.a.x + NODE_W / 2;
                const ay = e.a.y + NODE_H / 2;
                const bx = e.b.x + NODE_W / 2;
                const by = e.b.y + NODE_H / 2;
                const family = FAMILY_KINDS.has(e.kind);
                // Generational links route through an elbow; same-row links arc.
                const midY = (ay + by) / 2;
                const d =
                  e.a.gen === e.b.gen
                    ? `M ${ax} ${ay} C ${ax} ${ay - 46}, ${bx} ${by - 46}, ${bx} ${by}`
                    : `M ${ax} ${ay} L ${ax} ${midY} L ${bx} ${midY} L ${bx} ${by}`;
                return <path key={i} d={d} className={`tree-edge ${family ? "family" : "other"}`} />;
              })}
              {nodes.map((n) => (
                <g key={n.id} className="tree-node" onClick={() => navigate(`/people/${n.id}`)} role="link" aria-label={n.name}>
                  <rect
                    x={n.x}
                    y={n.y}
                    width={NODE_W}
                    height={NODE_H}
                    rx={10}
                    style={{ "--pc": `var(--c-${n.color})` } as React.CSSProperties}
                  />
                  <text x={n.x + NODE_W / 2} y={n.y + NODE_H / 2 + 4} textAnchor="middle">
                    {n.name.length > 20 ? `${n.name.slice(0, 19)}…` : n.name}
                  </text>
                </g>
              ))}
            </svg>
            <p className="panel-note tree-legend">
              Solid lines are family (parent, child, sibling, partner); dashed are other links
              (friend, coworker, mentor). Click a name to open the profile.
            </p>
          </div>
        )}
      </div>
    </>
  );
}

import { describe, expect, it } from "vitest";
import type { Entry } from "@cal/api-client";
import { allTags, groupTasks, linkDomain } from "./entries";

function task(partial: Partial<Entry>): Entry {
  return {
    id: partial.id ?? Math.random().toString(36).slice(2),
    title: "x",
    type: "task",
    date: "2026-09-10",
    completed: false,
    color: "slate",
    tags: [],
    recur: "none",
    createdAt: "2026-09-01T00:00:00Z",
    ...partial,
  };
}

describe("groupTasks", () => {
  const today = "2026-09-10";
  const entries = [
    task({ id: "o", date: "2026-09-01" }),
    task({ id: "t1", date: today, startTime: "15:00" }),
    task({ id: "t2", date: today, startTime: "09:00" }),
    task({ id: "u", date: "2026-09-20" }),
    task({ id: "d", date: today, completed: true }),
    task({ id: "n", type: "note" }),
  ];

  it("buckets by completion and date relative to today", () => {
    const g = groupTasks(entries, today);
    expect(g.overdue.map((e) => e.id)).toEqual(["o"]);
    expect(g.today.map((e) => e.id)).toEqual(["t2", "t1"]); // time-sorted
    expect(g.upcoming.map((e) => e.id)).toEqual(["u"]);
    expect(g.done.map((e) => e.id)).toEqual(["d"]);
  });

  it("ignores non-task types", () => {
    const g = groupTasks(entries, today);
    expect(Object.values(g).flat().map((e) => e.id)).not.toContain("n");
  });
});

describe("allTags", () => {
  it("ranks by frequency then alphabetically", () => {
    const entries = [
      task({ tags: ["work", "home"] }),
      task({ tags: ["work"] }),
      task({ tags: ["errand"] }),
    ];
    expect(allTags(entries)).toEqual(["work", "errand", "home"]);
  });
});

describe("linkDomain", () => {
  it("strips protocol and www", () => {
    expect(linkDomain("https://www.linear.app/issues")).toBe("linear.app");
    expect(linkDomain("not a url")).toBe("");
    expect(linkDomain(undefined)).toBe("");
  });
});

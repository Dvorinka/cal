import type { Entry } from "@cal/api-client";
import { describe, expect, it } from "vitest";
import { openHabitOccurrence } from "./habits";

function task(patch: Partial<Entry> & Pick<Entry, "id" | "title" | "date">): Entry {
  return {
    content: "",
    type: "task",
    completed: false,
    pinned: false,
    color: "slate",
    tags: ["habit"],
    recur: "daily",
    createdAt: "2026-11-01T00:00:00Z",
    ...patch,
  };
}

describe("openHabitOccurrence", () => {
  const today = "2026-11-12";

  it("picks the latest open occurrence on or before today", () => {
    const entries = [
      task({ id: "old", title: "Run", date: "2026-11-10" }),
      task({ id: "today", title: "Run", date: today }),
      task({ id: "future", title: "Run", date: "2026-11-13" }),
    ];
    expect(openHabitOccurrence(entries, { title: "Run" }, today)?.id).toBe("today");
  });

  it("ignores completed rows, other titles, and untagged tasks", () => {
    const entries = [
      task({ id: "done", title: "Run", date: today, completed: true }),
      task({ id: "other", title: "Read", date: today }),
      task({ id: "plain", title: "Run", date: today, tags: [] }),
    ];
    expect(openHabitOccurrence(entries, { title: "Run" }, today)).toBeUndefined();
  });
});

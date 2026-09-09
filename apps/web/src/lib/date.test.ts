import { describe, expect, it } from "vitest";
import { iso, monthMatrix, rangeFor, weekDays } from "./date";

describe("date helpers", () => {
  it("builds Monday-first weeks", () => {
    const days = weekDays(new Date(2026, 4, 12));
    expect(days).toHaveLength(7);
    expect(iso(days[0])).toBe("2026-05-11");
  });

  it("builds a six-week month matrix", () => {
    expect(monthMatrix(new Date(2026, 4, 12))).toHaveLength(42);
  });

  it("creates day range", () => {
    expect(rangeFor("day", new Date(2026, 4, 12))).toEqual({
      from: "2026-05-12",
      to: "2026-05-12",
    });
  });
});

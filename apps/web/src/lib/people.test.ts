import type { Person } from "@cal/api-client";
import { describe, expect, it } from "vitest";
import { nextByPerson, personDatesInRange } from "./people";

const person = (over: Partial<Person>): Person => ({
  id: "p1",
  name: "Ada",
  relation: "friend",
  dates: [],
  notes: "",
  color: "rose",
  createdAt: "2026-01-01T00:00:00Z",
  ...over,
});

describe("person dates", () => {
  it("expands a birthday into every year of the range", () => {
    const out = personDatesInRange([person({ birthday: "1990-05-10" })], "2026-01-01", "2027-12-31");
    expect(out.map((o) => o.date)).toEqual(["2026-05-10", "2027-05-10"]);
    expect(out[0].label).toBe("birthday");
    expect(out[0].turning).toBe(36);
  });

  it("expands named dates alongside the birthday", () => {
    const out = personDatesInRange(
      [person({ birthday: "1990-03-01", dates: [{ label: "anniversary", date: "2015-06-20" }] })],
      "2026-01-01",
      "2026-12-31",
    );
    expect(out.map((o) => `${o.label}@${o.date}`)).toEqual(["birthday@2026-03-01", "anniversary@2026-06-20"]);
    expect(out[1].turning).toBe(11);
  });

  it("clamps Feb 29 to Feb 28 on non-leap years", () => {
    const out = personDatesInRange([person({ birthday: "2000-02-29" })], "2026-01-01", "2026-12-31");
    expect(out.map((o) => o.date)).toEqual(["2026-02-28"]);
    const leap = personDatesInRange([person({ birthday: "2000-02-29" })], "2028-01-01", "2028-12-31");
    expect(leap.map((o) => o.date)).toEqual(["2028-02-29"]);
  });

  it("skips malformed dates and reports no turning for future years", () => {
    const out = personDatesInRange(
      [person({ dates: [{ label: "bad", date: "nope" }, { label: "someday", date: "2030-07-04" }] })],
      "2026-01-01",
      "2026-12-31",
    );
    expect(out).toHaveLength(1);
    expect(out[0].turning).toBeUndefined();
  });

  it("picks the soonest occurrence per person", () => {
    const people = [
      person({ id: "a", name: "A", birthday: "1990-12-01" }),
      person({ id: "b", name: "B", birthday: "1990-01-05" }),
    ];
    const map = nextByPerson(people);
    expect(map.get("a")?.label).toBe("birthday");
    expect(map.get("b")?.daysUntil).toBeGreaterThanOrEqual(0);
    expect(map.size).toBe(2);
  });
});

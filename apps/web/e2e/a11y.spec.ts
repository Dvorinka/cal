import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";
import { signUp } from "./helpers";

test.describe("accessibility", () => {
  for (const [name, nav] of [
    ["today", "/today"],
    ["tasks", "/tasks"],
    ["settings", "/settings"],
  ] as const) {
    test(`${name} page has no serious axe violations`, async ({ page }) => {
      await signUp(page);
      await page.goto(nav);
      const results = await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa"])
        .analyze();
      const serious = results.violations.filter(
        (v) => v.impact === "serious" || v.impact === "critical",
      );
      expect(serious, JSON.stringify(serious, null, 2)).toEqual([]);
    });
  }
});

import { expect, test } from "@playwright/test";
import { signUp } from "./helpers";

// The dev server ships no service worker, so a hard reload while offline can't
// be exercised here — that's the PWA path in production. What we verify: the
// queue persists ops, shows a pending count, flushes on reconnect, and the
// server-side entry survives a normal reload.
test.describe("offline queue", () => {
  test("create while offline → pending pill → flushes on reconnect", async ({ page, context }) => {
    await signUp(page);

    await context.setOffline(true);
    await page.getByRole("button", { name: "New entry" }).click();
    await page.getByPlaceholder("Task").fill("offline-task");
    await page.getByRole("button", { name: "Create" }).click();
    await expect(page.getByText("offline-task").first()).toBeVisible();
    await expect(page.getByText(/pending|offline/i).first()).toBeVisible();

    // Back online — the queue flushes and the pending pill clears.
    await context.setOffline(false);
    await expect(page.getByText(/\d+ pending/i)).toHaveCount(0, { timeout: 10_000 });
    await page.reload();
    await expect(page.getByText("offline-task").first()).toBeVisible();
  });
});

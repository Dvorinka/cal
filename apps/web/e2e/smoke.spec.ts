import { expect, test } from "@playwright/test";
import { signUp } from "./helpers";

test.describe("core flows", () => {
  test("sign up → create task → complete → persists on reload", async ({ page }) => {
    await signUp(page);

    // Create via the editor
    await page.getByRole("button", { name: "New entry" }).click();
    await page.getByPlaceholder("Task").fill("E2E task");
    await page.getByRole("button", { name: "Create" }).click();
    await expect(page.getByText("E2E task").first()).toBeVisible();

    // Today page shows it
    await page.getByRole("link", { name: "Today" }).click();
    await expect(page.getByText("E2E task").first()).toBeVisible();

    // Complete it
    await page.getByRole("button", { name: "Complete" }).first().click();
    await expect(page.locator(".check-list li.done").first()).toBeVisible();

    // Persists across reload
    await page.reload();
    await expect(page.getByText("E2E task").first()).toBeVisible();
  });

  test("navigation across pages + command palette", async ({ page }) => {
    await signUp(page);
    await page.getByRole("link", { name: "Tasks" }).first().click();
    await expect(page.getByPlaceholder(/dentist fri/i)).toBeVisible();
    await page.getByRole("link", { name: "Notes" }).first().click();
    await expect(page.getByRole("heading", { name: "Notes" })).toBeVisible();
    await page.getByRole("link", { name: "Links" }).first().click();
    await expect(page.getByRole("heading", { name: "Links" })).toBeVisible();
    await page.getByRole("link", { name: "Settings" }).first().click();
    await expect(page.getByRole("heading", { name: "Appearance" })).toBeVisible();
    // Palette opens with ⌘K
    await page.keyboard.press("ControlOrMeta+k");
    await expect(page.getByPlaceholder(/search/i)).toBeVisible();
    await page.keyboard.press("Escape");
  });

  test("settings round-trips a theme change", async ({ page }) => {
    await signUp(page);
    await page.getByRole("link", { name: "Settings" }).click();
    await page.getByLabel("Theme").selectOption("light");
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  });
});

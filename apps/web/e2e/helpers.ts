import { expect, type Page } from "@playwright/test";

// Fresh user per spec — no shared state between runs.
export async function signUp(page: Page) {
  const email = `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 7)}@cal.local`;
  await page.goto("/");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("e2e-password-123");
  await page.getByRole("button", { name: "Create an account" }).click();
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page.getByRole("button", { name: "New entry" })).toBeVisible({ timeout: 10_000 });
  return email;
}

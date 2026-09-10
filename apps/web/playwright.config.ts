import { defineConfig } from "@playwright/test";

// E2E against a running stack: `npm run e2e` with the dev servers up
// (or E2E_BASE_URL pointing at a deployed instance). Each run registers a
// fresh user so specs are order-independent.
export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: 1,
  workers: 2,
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:5173",
    screenshot: "only-on-failure",
  },
  projects: [{ name: "chromium", use: { browserName: "chromium" } }],
});

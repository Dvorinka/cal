// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from "vitest";
import { canUseLocal, clearImport, isLocalMode, markLocalMode, wantsImport } from "./local";

describe("local mode flags", () => {
  beforeEach(() => localStorage.clear());

  it("is never local off the Android shell", () => {
    expect(canUseLocal()).toBe(false);
    markLocalMode(true);
    expect(isLocalMode()).toBe(false); // flag set, but the gate holds
  });

  it("marks the import intent for a later server login", () => {
    markLocalMode(true);
    expect(wantsImport()).toBe(true);
    clearImport();
    expect(wantsImport()).toBe(false);
  });

  it("exit clears the mode but keeps the import offer pending", () => {
    markLocalMode(true);
    markLocalMode(false);
    expect(wantsImport()).toBe(true);
  });
});

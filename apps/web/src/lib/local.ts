import { Capacitor } from "@capacitor/core";

// Local mode: the app runs with no Cal server at all. Mail talks straight to
// the user's own IMAP/SMTP provider through the native CalMail plugin — the
// web platform has no raw TCP/TLS, so this can only exist on Android.
// "cal:mode" persists the choice; "cal:import" marks that local mail accounts
// should be offered for import the next time a server login succeeds.
const MODE_KEY = "cal:mode";
const IMPORT_KEY = "cal:import";

export function canUseLocal(): boolean {
  try {
    return Capacitor.isNativePlatform() && Capacitor.getPlatform() === "android";
  } catch {
    return false;
  }
}

export function isLocalMode(): boolean {
  try {
    return canUseLocal() && localStorage.getItem(MODE_KEY) === "local";
  } catch {
    return false;
  }
}

export function markLocalMode(on: boolean): void {
  try {
    if (on) {
      localStorage.setItem(MODE_KEY, "local");
      localStorage.setItem(IMPORT_KEY, "1");
    } else {
      localStorage.removeItem(MODE_KEY);
    }
  } catch {
    // no storage
  }
}

export function wantsImport(): boolean {
  try {
    return localStorage.getItem(IMPORT_KEY) === "1";
  } catch {
    return false;
  }
}

export function clearImport(): void {
  try {
    localStorage.removeItem(IMPORT_KEY);
  } catch {
    // no storage
  }
}

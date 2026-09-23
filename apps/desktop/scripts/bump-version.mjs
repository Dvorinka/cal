#!/usr/bin/env node
// Usage: bump-version.mjs <latestReleasedVersion>
//
// Single source of truth for "what version is next" — used by
// .github/workflows/auto-release.yml and by hand when staging a release PR.
//
// Decision:
//   - wails.json productVersion already ahead of the latest tag (a human
//     bumped the files in a release PR): keep it, edit nothing.
//   - Otherwise: patch-bump the latest tag version and write it into every
//     versioned file.
//
// Prints GITHUB_OUTPUT-ready lines: "next=X.Y.Z" and "changed=true|false".

import { readFileSync, writeFileSync } from "node:fs";

const latest = process.argv[2] ?? "0.0.0";

const cmp = (a, b) => {
  const pa = a.split(".").map(Number);
  const pb = b.split(".").map(Number);
  for (let i = 0; i < 3; i++) if (pa[i] !== pb[i]) return pa[i] - pb[i];
  return 0;
};

const readJson = (p) => JSON.parse(readFileSync(p, "utf8"));
const writeJson = (p, j) => writeFileSync(p, JSON.stringify(j, null, 2) + "\n");

const wailsPath = "apps/desktop/wails.json";
const wails = readJson(wailsPath);
const repo = wails.info.productVersion;

let next = repo;
let changed = false;

if (cmp(repo, latest) <= 0) {
  const [major, minor, patch] = latest.split(".").map(Number);
  next = `${major}.${minor}.${patch + 1}`;
  changed = true;

  // Desktop installer/app metadata.
  wails.info.productVersion = next;
  writeJson(wailsPath, wails);

  // Web workspace version.
  const webPath = "apps/web/package.json";
  const web = readJson(webPath);
  web.version = next;
  writeJson(webPath, web);

  // Android: versionCode is a monotonic int — Play/sideload installs require
  // each build to be strictly higher than the last.
  const gradlePath = "apps/web/android/app/build.gradle";
  const gradle = readFileSync(gradlePath, "utf8");
  const code = Number(gradle.match(/versionCode (\d+)/)[1]) + 1;
  writeFileSync(
    gradlePath,
    gradle
      .replace(/versionCode \d+/, `versionCode ${code}`)
      .replace(/versionName "[^"]+"/, `versionName "${next}"`),
  );

  // Lockfile mirrors the workspace versions; root package.json stays put —
  // it was never bumped for releases.
  const lockPath = "package-lock.json";
  const lock = readJson(lockPath);
  lock.version = next;
  lock.packages[""].version = next;
  lock.packages["apps/web"].version = next;
  writeJson(lockPath, lock);
}

console.log(`next=${next}\nchanged=${changed}`);

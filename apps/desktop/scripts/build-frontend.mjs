// Build the web app and stage it under frontend/dist for go:embed.
// Node script (not sh) so `wails build` works on Windows runners too.
import { cpSync, rmSync } from "node:fs";
import { execSync } from "node:child_process";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "../../..");
execSync("npm run build -w @cal/web", { cwd: root, stdio: "inherit" });
rmSync(join(root, "apps/desktop/frontend/dist"), { recursive: true, force: true });
cpSync(join(root, "apps/web/dist"), join(root, "apps/desktop/frontend/dist"), { recursive: true });

// Downloads the zonky embedded-Postgres binaries and unpacks them into a
// directory the installers can bundle. Bundled runtime = fully offline first
// launch and no stray console windows (the app spawns postgres itself).
//
//   node scripts/prepare-pg-runtime.mjs [os] [arch] [outDir]
//   node scripts/prepare-pg-runtime.mjs windows amd64 build/windows/installer/pg-runtime
//
// Keep PG_VERSION in sync with pgVersion in postgres.go.
// Zero-dependency: the jar's single .txz entry is pulled out with a minimal
// zip reader (zlib is in Node stdlib), then system tar handles the xz/tar.
import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync, renameSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { inflateRawSync } from "node:zlib";

const PG_VERSION = "18.3.0";
const REPO = "https://repo1.maven.org/maven2";

const [osName = "windows", arch = "amd64", outDir = "build/windows/installer/pg-runtime"] =
  process.argv.slice(2);

const artifact = `embedded-postgres-binaries-${osName}-${arch}`;
const jarURL = `${REPO}/io/zonky/test/postgres/${artifact}/${PG_VERSION}/${artifact}-${PG_VERSION}.jar`;

// Pull the one .txz member out of the jar (zip format, usually stored
// uncompressed or deflated).
function extractTxz(jarPath) {
  const buf = readFileSync(jarPath);
  let eocd = -1;
  for (let i = buf.length - 22; i >= Math.max(0, buf.length - 22 - 65536); i--) {
    if (buf.readUInt32LE(i) === 0x06054b50) { eocd = i; break; }
  }
  if (eocd < 0) throw new Error("not a zip/jar");

  const count = buf.readUInt16LE(eocd + 10);
  let off = buf.readUInt32LE(eocd + 16);
  for (let i = 0; i < count; i++) {
    if (buf.readUInt32LE(off) !== 0x02014b50) break;
    const method = buf.readUInt16LE(off + 10);
    const compSize = buf.readUInt32LE(off + 20);
    const nameLen = buf.readUInt16LE(off + 28);
    const extraLen = buf.readUInt16LE(off + 30);
    const commentLen = buf.readUInt16LE(off + 32);
    const localOff = buf.readUInt32LE(off + 42);
    const name = buf.toString("utf8", off + 46, off + 46 + nameLen);
    if (name.endsWith(".txz")) {
      if (compSize === 0xffffffff) throw new Error("zip64 jar — not supported");
      const lNameLen = buf.readUInt16LE(localOff + 26);
      const lExtraLen = buf.readUInt16LE(localOff + 28);
      const data = buf.subarray(localOff + 30 + lNameLen + lExtraLen,
        localOff + 30 + lNameLen + lExtraLen + compSize);
      if (method === 0) return data;
      if (method === 8) return inflateRawSync(data);
      throw new Error(`unsupported zip compression method ${method}`);
    }
    off += 46 + nameLen + extraLen + commentLen;
  }
  throw new Error("no .txz inside the jar");
}

const tmp = mkdtempSync(join(tmpdir(), "pg-runtime-"));
try {
  process.stdout.write(`downloading ${jarURL}\n`);
  const jarPath = join(tmp, "pg.jar");
  const jar = Buffer.from(await (await fetch(jarURL)).arrayBuffer());
  writeFileSync(jarPath, jar);

  const sha = (await (await fetch(jarURL + ".sha256")).text()).trim();
  const got = createHash("sha256").update(jar).digest("hex");
  if (got !== sha) throw new Error(`sha256 mismatch: ${got} != ${sha}`);

  const txzPath = join(tmp, "pg.txz");
  writeFileSync(txzPath, extractTxz(jarPath));

  // Stage next to the destination so the final rename stays on one filesystem.
  const staging = `${outDir}.tmp-${process.pid}`;
  rmSync(staging, { recursive: true, force: true });
  try {
    mkdirSync(staging, { recursive: true });
    execFileSync("tar", ["-xJf", txzPath, "-C", staging], { stdio: "inherit" });

    const exe = osName === "windows" ? ".exe" : "";
    if (!existsSync(join(staging, "bin", `postgres${exe}`))) {
      throw new Error(`archive missing bin/postgres${exe} — wrong layout?`);
    }
    rmSync(outDir, { recursive: true, force: true });
    renameSync(staging, outDir);
  } finally {
    rmSync(staging, { recursive: true, force: true });
  }
  writeFileSync(join(outDir, "RUNTIME_VERSION.txt"), `postgres ${PG_VERSION} (${osName}/${arch})\n`);
  process.stdout.write(`pg-runtime ready at ${outDir}\n`);
} finally {
  rmSync(tmp, { recursive: true, force: true });
}

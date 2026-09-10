// Offline write queue: mutations that fail on a network error are queued in
// localStorage and replayed in order once connectivity returns. Creates use a
// tmp-* id; later ops against that id are rewritten after the create lands.

import type { EntryInput, EntryPatch } from "@cal/api-client";

export type PendingOp =
  | { kind: "create"; id: string; input: EntryInput }
  | { kind: "update"; id: string; patch: EntryPatch }
  | { kind: "delete"; id: string };

const KEY = "cal:opq";

export function readQueue(): PendingOp[] {
  try {
    if (typeof localStorage === "undefined") return [];
    return JSON.parse(localStorage.getItem(KEY) ?? "[]") as PendingOp[];
  } catch {
    return [];
  }
}

export function writeQueue(ops: PendingOp[]): void {
  try {
    if (typeof localStorage === "undefined") return;
    localStorage.setItem(KEY, JSON.stringify(ops));
  } catch {
    /* storage unavailable */
  }
}

export function enqueue(op: PendingOp): void {
  const ops = readQueue();
  // Coalesce: an update following a create of the same temp id merges into it.
  if (op.kind === "update") {
    const prev = ops.findLast((o) => o.kind === "create" && o.id === op.id);
    if (prev && prev.kind === "create") {
      prev.input = { ...prev.input, ...op.patch } as EntryInput;
      writeQueue(ops);
      return;
    }
    // Two updates to the same id merge, latest wins per field.
    const prior = ops.findLast((o) => o.kind === "update" && o.id === op.id);
    if (prior && prior.kind === "update") {
      prior.patch = { ...prior.patch, ...op.patch };
      writeQueue(ops);
      return;
    }
  }
  // A delete cancels any pending update/create for the same id.
  if (op.kind === "delete") {
    const rest = ops.filter((o) => !(o.id === op.id && o.kind !== "delete"));
    const hadCreate = ops.some((o) => o.kind === "create" && o.id === op.id);
    writeQueue(hadCreate ? rest : [...rest, op]); // never-created: nothing to delete server-side
    return;
  }
  writeQueue([...ops, op]);
}

export function newTempId(): string {
  return `tmp-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

// isOfflineError distinguishes "server unreachable" from a real HTTP error.
export function isOfflineError(error: unknown): boolean {
  if (error instanceof TypeError) return true; // fetch network failure
  const status = (error as { status?: number })?.status;
  return status === 0 || status === 502 || status === 503;
}

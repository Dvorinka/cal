// Files — self-hosted file bin. Drag-drop or click to upload; each file can
// get a public share link. Images preview inline; everything else downloads.

import type { FileRec } from "@cal/api-client";
import { File as FileIcon, FileText, FileImage, FileArchive, FileAudio, FileVideo, Link2, Trash2, Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { usePlanner } from "../stores/planner";

function iconFor(mime: string) {
  if (mime.startsWith("image/")) return FileImage;
  if (mime.startsWith("video/")) return FileVideo;
  if (mime.startsWith("audio/")) return FileAudio;
  if (mime.startsWith("text/") || mime === "application/pdf") return FileText;
  if (/zip|tar|gzip|7z|rar/.test(mime)) return FileArchive;
  return FileIcon;
}

function humanSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1 << 20) return `${(n / 1024).toFixed(0)} KB`;
  return `${(n / (1 << 20)).toFixed(1)} MB`;
}

export function FilesPage() {
  const api = usePlanner((s) => s.api);
  const toast = usePlanner((s) => s.toast);
  const [files, setFiles] = useState<FileRec[]>([]);
  const [dragOver, setDragOver] = useState(false);
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<FileRec>();
  const inputRef = useRef<HTMLInputElement>(null);

  const load = () => void api.files().then(setFiles).catch(() => {});
  useEffect(load, [api]);

  async function upload(list: FileList | null) {
    if (!list?.length) return;
    setBusy(true);
    try {
      for (const f of Array.from(list)) {
        await api.upload(f);
      }
      load();
      toast(list.length === 1 ? "Uploaded" : `${list.length} files uploaded`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  async function toggleShare(f: FileRec) {
    const on = !f.shareToken;
    try {
      const { shareToken } = await api.shareFile(f.id, on);
      setFiles((fs) => fs.map((x) => (x.id === f.id ? { ...x, shareToken: shareToken ?? undefined } : x)));
      if (shareToken) {
        const url = `${location.origin}/api/shared/files/${shareToken}`;
        await navigator.clipboard.writeText(url).catch(() => {});
        toast("Share link copied");
      } else {
        toast("Sharing off");
      }
    } catch {
      toast("Share failed");
    }
  }

  async function remove(f: FileRec) {
    await api.deleteFile(f.id).catch(() => {});
    setFiles((fs) => fs.filter((x) => x.id !== f.id));
    toast("Deleted");
  }

  return (
    <div className="page files-page">
      <header className="page-head">
        <div>
          <h1>Files</h1>
          <p className="page-sub">Drop anything — attach it to notes later, or share a public link.</p>
        </div>
        <button type="button" className="btn btn-primary" disabled={busy} onClick={() => inputRef.current?.click()}>
          <Upload size={14} /> {busy ? "Uploading…" : "Upload"}
        </button>
      </header>

      <input
        ref={inputRef}
        type="file"
        multiple
        hidden
        onChange={(e) => void upload(e.target.files)}
        aria-label="Upload files"
      />

      <div
        className={`files-drop ${dragOver ? "over" : ""}`}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragOver(false);
          void upload(e.dataTransfer.files);
        }}
        role="button"
        tabIndex={0}
        aria-label="Drop files to upload"
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") inputRef.current?.click();
        }}
      >
        {files.length === 0 ? (
          <p className="files-empty">Drop files here, or click Upload — 20 MB each, quota shown in Settings.</p>
        ) : (
          <ul className="files-list">
            {files.map((f) => {
              const Icon = iconFor(f.mime);
              return (
                <li key={f.id} className="files-item">
                  <button
                    type="button"
                    className="files-main"
                    onClick={() => (f.mime.startsWith("image/") ? setPreview(f) : window.open(`/api/files/${f.name}`, "_blank"))}
                    aria-label={`Open ${f.origName}`}
                  >
                    <Icon size={18} strokeWidth={1.8} />
                    <span className="files-name">{f.origName}</span>
                    <span className="files-meta">
                      {humanSize(f.size)} · {new Date(f.createdAt).toLocaleDateString()}
                    </span>
                  </button>
                  <button
                    type="button"
                    className={`icon-btn ${f.shareToken ? "share-on" : ""}`}
                    aria-label={f.shareToken ? "Copy public link / unshare" : "Create public link"}
                    title={f.shareToken ? "Shared — click to copy link, again to unshare via toggle in menu… copying now" : "Create public link"}
                    onClick={() => void toggleShare(f)}
                  >
                    <Link2 size={14} />
                  </button>
                  <button type="button" className="icon-btn" aria-label={`Delete ${f.origName}`} onClick={() => void remove(f)}>
                    <Trash2 size={14} />
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {preview && (
        <div className="files-preview" role="dialog" aria-modal="true" aria-label={preview.origName} onClick={() => setPreview(undefined)}>
          <button type="button" className="files-preview-close" aria-label="Close preview">
            <X size={20} />
          </button>
          <img src={`/api/files/${preview.name}`} alt={preview.origName} onClick={(e) => e.stopPropagation()} />
        </div>
      )}
    </div>
  );
}

import JSZip from "jszip";
import type { NoteListItem } from "../api/notes";

const ATTACHMENT_RE = /(?:!\[[^\]]*\]|]\s*)\(\/attachments\/([^)]+)\)/g;

function extractAttachmentFilenames(markdown: string): string[] {
  const names = new Set<string>();
  for (const m of markdown.matchAll(ATTACHMENT_RE)) {
    const decoded = decodeURIComponent(m[1]);
    names.add(decoded);
  }
  return [...names];
}

/** Rewrite /attachments/x to relative path from note at notePath. */
function rewriteAttachmentsToRelative(markdown: string, notePath: string): string {
  const depth = notePath.split("/").length - 1; // research/projects/note.md → 2
  const prefix = depth > 0 ? "../".repeat(depth) : "";
  return markdown.replaceAll("/attachments/", `${prefix}attachments/`);
}

function rewriteToRelative(markdown: string): string {
  return markdown.replaceAll("/attachments/", "attachments/");
}

async function fetchAttachment(filename: string): Promise<Blob> {
  const res = await fetch(`/attachments/${encodeURIComponent(filename)}`);
  if (!res.ok) throw new Error(`Failed to fetch attachment: ${filename}`);
  return res.blob();
}

function triggerDownload(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * Downloads a note as a plain .md when it has no attachments,
 * or as a .zip bundle (note + attachments/) when it does.
 */
export async function downloadNote(
  content: string,
  notePath: string,
): Promise<void> {
  const baseName = notePath.split("/").pop() ?? "note.md";
  const filenames = extractAttachmentFilenames(content);

  if (filenames.length === 0) {
    const blob = new Blob([content], { type: "text/markdown;charset=utf-8" });
    triggerDownload(blob, baseName);
    return;
  }

  const zip = new JSZip();
  zip.file(baseName, rewriteToRelative(content));

  const attachDir = zip.folder("attachments")!;
  const results = await Promise.allSettled(
    filenames.map(async (name) => {
      const blob = await fetchAttachment(name);
      attachDir.file(name, blob);
    }),
  );

  const failed = results.filter((r) => r.status === "rejected");
  if (failed.length > 0) {
    console.warn(
      `Failed to fetch ${failed.length} attachment(s):`,
      failed.map((r) => (r as PromiseRejectedResult).reason),
    );
  }

  const zipBlob = await zip.generateAsync({ type: "blob" });
  const zipName = baseName.replace(/\.md$/i, "") + ".zip";
  triggerDownload(zipBlob, zipName);
}

/** Fetches note content by path. */
export type FetchNoteContent = (path: string) => Promise<string>;

/**
 * Downloads a directory of notes as a .zip with preserved folder structure
 * and a shared attachments/ folder. Paths in markdown are rewritten to
 * correct relative paths.
 */
export async function downloadDirectory(
  folderPrefix: string,
  notesInFolder: NoteListItem[],
  getNoteContent: FetchNoteContent,
): Promise<void> {
  const notePaths = notesInFolder
    .map((n) => n.path)
    .filter((p) => p.endsWith(".md"));

  if (notePaths.length === 0) {
    throw new Error("No notes in this folder");
  }

  const zip = new JSZip();
  const allAttachments = new Set<string>();

  for (const path of notePaths) {
    const content = await getNoteContent(path);
    const filenames = extractAttachmentFilenames(content);
    filenames.forEach((f) => allAttachments.add(f));
    const rewritten = rewriteAttachmentsToRelative(content, path);
    zip.file(path, rewritten);
  }

  const attachDir = zip.folder("attachments")!;
  const results = await Promise.allSettled(
    [...allAttachments].map(async (name) => {
      const blob = await fetchAttachment(name);
      attachDir.file(name, blob);
    }),
  );

  const failed = results.filter((r) => r.status === "rejected");
  if (failed.length > 0) {
    console.warn(
      `Failed to fetch ${failed.length} attachment(s):`,
      failed.map((r) => (r as PromiseRejectedResult).reason),
    );
  }

  const zipBlob = await zip.generateAsync({ type: "blob" });
  const zipName =
    (folderPrefix || "notes").replace(/\/$/, "").split("/").pop() || "folder";
  triggerDownload(zipBlob, `${zipName}.zip`);
}

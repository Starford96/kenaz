import api from "./client";

export interface NoteListItem {
  path: string;
  title: string;
  checksum: string;
  tags: string[];
  updated_at: string;
}

export interface NoteDetail {
  path: string;
  title: string;
  content: string;
  checksum: string;
  tags: string[];
  frontmatter: Record<string, unknown> | null;
  backlinks: string[];
  updated_at: string;
}

export interface SearchResult {
  path: string;
  title: string;
  snippet: string;
}

export interface GraphData {
  nodes: { id: string; title?: string }[];
  links: { source: string; target: string }[];
}

/** List notes with optional pagination and tag filter. */
export async function listNotes(params?: {
  limit?: number;
  offset?: number;
  tag?: string;
  sort?: string;
}) {
  const { data } = await api.get<{ notes: NoteListItem[]; total: number }>(
    "/notes",
    { params },
  );
  return data;
}

/** Get a single note by path. */
export async function getNote(path: string) {
  const { data } = await api.get<NoteDetail>(`/notes/${path}`);
  return data;
}

/** Create a new note. */
export async function createNote(path: string, content: string) {
  const { data } = await api.post<NoteDetail>("/notes", { path, content });
  return data;
}

/** Update a note with optional optimistic concurrency. */
export async function updateNote(
  path: string,
  content: string,
  checksum?: string,
) {
  const headers: Record<string, string> = {};
  if (checksum) headers["If-Match"] = checksum;
  const { data } = await api.put<NoteDetail>(`/notes/${path}`, { content }, { headers });
  return data;
}

/** Delete a note. */
export async function deleteNote(path: string) {
  await api.delete(`/notes/${path}`);
}

/** Full-text search. */
export async function searchNotes(q: string, limit = 20) {
  const { data } = await api.get<{ results: SearchResult[] }>("/search", {
    params: { q, limit },
  });
  return data.results;
}

/** Get the knowledge graph. */
export async function getGraph() {
  const { data } = await api.get<GraphData>("/graph");
  return data;
}

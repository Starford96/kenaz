/**
 * Typed API functions built on the generated OpenAPI client.
 * All types flow from api/schema/kenaz/openapi.yaml → schema.d.ts → here.
 */
import api from "./client";
import type { components } from "./schema";

// Re-export schema types used by components.
export type NoteListItem = components["schemas"]["NoteListItem"];
export type NoteDetail = components["schemas"]["NoteDetail"];
export type SearchResult = components["schemas"]["SearchResult"];
export type GraphNode = components["schemas"]["GraphNode"];
export type GraphLink = components["schemas"]["GraphLink"];

export interface GraphData {
  nodes: GraphNode[];
  links: GraphLink[];
}

/** List notes with optional pagination and filtering. */
export async function listNotes(params?: {
  limit?: number;
  offset?: number;
  tag?: string;
  sort?: "updated_at" | "title" | "path";
}) {
  const { data } = await api.GET("/notes", { params: { query: params } });
  if (!data) throw new Error("failed to list notes");
  return data;
}

/** Get a single note by path. */
export async function getNote(path: string) {
  const { data, error } = await api.GET("/notes/{path}", {
    params: { path: { path } },
  });
  if (error) throw new Error(error.error);
  return data!;
}

/** Create a new note. */
export async function createNote(path: string, content: string) {
  const { data, error } = await api.POST("/notes", {
    body: { path, content },
  });
  if (error) throw new Error(error.error);
  return data!;
}

/** Update a note with optional optimistic concurrency. */
export async function updateNote(
  path: string,
  content: string,
  checksum?: string,
) {
  const { data, error } = await api.PUT("/notes/{path}", {
    params: {
      path: { path },
      header: checksum ? { "If-Match": checksum } : undefined,
    },
    body: { content },
  });
  if (error) throw new Error(error.error);
  return data!;
}

/** Delete a note. */
export async function deleteNote(path: string) {
  const { error } = await api.DELETE("/notes/{path}", {
    params: { path: { path } },
  });
  if (error) throw new Error(error.error);
}

/** Full-text search. */
export async function searchNotes(q: string, limit = 20) {
  const { data, error } = await api.GET("/search", {
    params: { query: { q, limit } },
  });
  if (error) throw new Error(error.error);
  return data!.results;
}

/** Get the knowledge graph. */
export async function getGraph() {
  const { data } = await api.GET("/graph");
  if (!data) throw new Error("failed to get graph");
  return data as GraphData;
}

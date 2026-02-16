import createClient from "openapi-fetch";
import type { paths } from "./schema";

const baseUrl = import.meta.env.VITE_API_BASE ?? "/api";
const token = import.meta.env.VITE_AUTH_TOKEN as string | undefined;

/** Typed API client generated from OpenAPI spec. */
const api = createClient<paths>({
  baseUrl,
  headers: token ? { Authorization: `Bearer ${token}` } : {},
});

export default api;

// Re-export schema types for convenience.
export type { paths, components, operations } from "./schema";

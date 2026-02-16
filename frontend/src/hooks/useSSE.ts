import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Connects to the SSE endpoint and invalidates react-query caches
 * when the backend reports note changes.
 */
export function useSSE() {
  const qc = useQueryClient();

  useEffect(() => {
    const base = import.meta.env.VITE_API_BASE ?? "/api";
    const url = `${base}/events`;
    const es = new EventSource(url);

    es.addEventListener("note.created", () => {
      qc.invalidateQueries({ queryKey: ["notes"] });
    });

    es.addEventListener("note.updated", (e) => {
      try {
        const { path } = JSON.parse(e.data);
        qc.invalidateQueries({ queryKey: ["note", path] });
        qc.invalidateQueries({ queryKey: ["notes"] });
      } catch {
        qc.invalidateQueries({ queryKey: ["notes"] });
      }
    });

    es.addEventListener("note.deleted", () => {
      qc.invalidateQueries({ queryKey: ["notes"] });
    });

    es.addEventListener("graph.updated", () => {
      qc.invalidateQueries({ queryKey: ["graph"] });
    });

    return () => es.close();
  }, [qc]);
}

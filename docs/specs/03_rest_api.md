# 3. REST API Specification

**Goal**: HTTP interface for frontend and external tools.

## 3.1. Infrastructure
-   **Router**: `go-chi/chi/v5`.
-   **Global Middleware** (applied to all routes):
    -   `RequestID`: Unique request tracking.
    -   `RealIP`: Extract real client IP behind proxies.
    -   `SlogRequestLogger`: Structured JSON logging (method, path, status, duration).
    -   `Recoverer`: Panic recovery.
-   **API Middleware** (applied to `/api` group):
    -   `AuthMiddleware`: Bearer Token validation with configurable modes:
        -   `disabled` (default): all requests pass through.
        -   `token`: requires `Authorization: Bearer <token>` header; fails fast at startup if token is empty.
    -   `CORS`: Allow requests from frontend origin.

## 3.2. Endpoints

### Health (unauthenticated, outside `/api` group)
-   `GET /health/live`: Liveness probe. Returns `{"status":"ok"}`.
-   `GET /health/ready`: Readiness probe. Returns `{"status":"ok"}`.

### Notes
-   `GET /api/notes`: List notes. Supported query params:
    -   `limit`, `offset`: Pagination.
    -   `sort`: `updated_at`, `title`, `path`.
    -   `tag`: Filter by tag.
-   `GET /api/notes/{path}`: Get single note.
    -   Returns: `{ path, title, content, checksum, tags, frontmatter, backlinks, updated_at }`
    -   Supports URL-encoded paths (e.g., `topics%2Fnote.md`).
-   `POST /api/notes`: Create new note.
    -   Body: `{ path: "folder/file.md", content: "..." }`
-   `PUT /api/notes/{path}`: Update note.
    -   Header: `If-Match: "checksum"` (Optimistic Concurrency).
    -   Body: `{ content: "..." }`
    -   Returns 409 Conflict if checksum mismatch.
-   `DELETE /api/notes/{path}`: Delete note.
-   `POST /api/notes/rename`: Rename note or directory.
    -   Body: `{ old_path: "...", new_path: "..." }`

### Search
-   `GET /api/search`:
    -   Query: `?q=search term`
    -   Returns: List of matches with context snippets.

### Graph
-   `GET /api/graph`:
    -   Returns full knowledge graph for visualization.
    -   Format: `{ nodes: [{id, title, tags}], links: [{source, target}] }`

### Attachments
-   `GET /attachments/{filename}`: Serve static files from `vault/attachments` (public, no auth).
-   `POST /api/attachments`: Upload file (multipart/form-data, auth-protected).

### SSE
-   `GET /api/events`: Server-Sent Events endpoint (auth-protected). See [04_realtime_updates.md](04_realtime_updates.md).

### Frontend SPA Fallback
-   `GET /*`: When `frontend.enabled=true`, serves static assets from `frontend.dist_path` or falls back to `index.html` for client-side routing.

## 3.3. Testing Strategy

### Unit Tests
-   **Middlewares**: Test Auth middleware accepts valid token and rejects missing/invalid ones. Test disabled mode passes all requests.
-   **Handlers (Mocked Service)**:
    -   Test status codes (200, 201, 400, 404, 409, 500) for each endpoint.
    -   Verify JSON response structure matches spec.
    -   Test Optimistic Locking: Verify PUT returns 409 if `If-Match` doesn't match.
    -   Test rename endpoint with notes and directories.

### Integration Tests
-   **E2E (HTTP + DB + FS)**:
    -   POST to create note -> GET to verify key fields -> Search to find it.
    -   Test pagination on List Notes endpoint.

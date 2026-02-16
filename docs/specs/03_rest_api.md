# 3. REST API Specification

**Goal**: HTTP interface for frontend and external tools.

## 3.1. Infrastructure
-   **Router**: `go-chi/chi/v5`.
-   **Middleware**:
    -   `Logger`: Structured logging (slog).
    -   `Auth`: Simple Bearer Token check against `config.yaml`.
    -   `Recoverer`: Panic recovery.
    -   `CORS`: Allow requests from frontend origin.

## 3.2. Endpoints

### Notes
-   `GET /api/notes`: List notes. Supported query params:
    -   `limit`, `offset`: Pagination.
    -   `sort`: `updated_at`, `title`.
    -   `tag`: Filter by tag.
-   `GET /api/notes/{path}`: Get single note.
    -   Returns: `{ metadata: {...}, content: "...", backlinks: [...] }`
-   `POST /api/notes`: Create new note.
    -   Body: `{ path: "folder/file.md", content: "..." }`
-   `PUT /api/notes/{path}`: Update note.
    -   Header: `If-Match: "checksum"` (Optimistic Concurrency).
    -   Body: `{ content: "..." }`
-   `DELETE /api/notes/{path}`: Delete note.

### Search
-   `GET /api/search`:
    -   Query: `?q=search term`
    -   Returns: List of matches with context snippets.

### Graph
-   `GET /api/graph`:
    -   Returns full graph or subgraph for visualization.
    -   Format: `{ nodes: [{id: "path"}], links: [{source: "a", target: "b"}] }`

### Attachments
-   `GET /attachments/{filename}`: Serve static files from `vault/attachments`.
-   `POST /api/attachments`: Upload file (multipart/form-data).

## 3.3. Testing Strategy

### Unit Tests
-   **Middlewares**: Test Auth middleware accepts valid token and rejects missing/invalid ones.
-   **Handlers (Mocked Service)**:
    -   Test status codes (200, 201, 400, 404, 500) for each endpoint.
    -   Verify JSON response structure matches spec.
    -   Test Optimistic Locking: Verify PUT returns 409 if `If-Match` doesn't match mock.

### Integration Tests
-   **E2E (HTTP + DB + FS)**:
    -   POST to create note -> GET to verify key fields -> Search to find it.
    -   Test pagination on List Notes endpoint.

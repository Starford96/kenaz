# 4. Real-time Updates (SSE)

**Goal**: Push changes to UI instantly.

## 4.1. Architecture
-   **Protocol**: Server-Sent Events (SSE).
-   **Endpoint**: `/api/events`.

## 4.2. Implementation
-   **Broker Struct**:
    -   Maintains list of active `chan []byte` (clients).
    -   Mutex for thread-safe add/remove.
-   **Broadcaster**:
    -   Listens to internal Go channel from **Indexer**.
    -   Loops through clients and sends payload.

## 4.3. Event Types
Format: `id: ... \n event: ... \n data: ... \n\n`

### Events
1.  **`note.created`**
    ```json
    { "path": "new-note.md", "title": "New Note" }
    ```
2.  **`note.updated`**
    ```json
    { "path": "existing.md", "checksum": "new-hash" }
    ```
3.  **`note.deleted`**
    ```json
    { "path": "removed.md" }
    ```
4.  **`graph.updated`** (Throttled)
    -   Signal to frontend to refresh the graph structure.

## 4.4. Client Handling
-   Frontend (`EventSource`) auto-reconnects on drop.
-   Server handles `Context.Done()` to clean up disconnected clients.

## 4.5. Testing Strategy

### Unit Tests
-   **Broker**:
    -   Test adding and removing clients.
    -   Verify broadcasting sends message to all active clients.
    -   Ensure thread safety (race detector) when adding/removing clients concurrently.

### Integration Tests
-   **Connection**: Connect a dummy client -> Trigger internal event -> Verify client receives SSE message with correct format.
-   **Lifecycle**: Verify client disconnect cleans up resources in Broker.

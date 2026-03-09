# 4. Real-time Updates (SSE)

**Goal**: Push changes to UI instantly.

## 4.1. Architecture
-   **Protocol**: Server-Sent Events (SSE).
-   **Endpoint**: `GET /api/events` (auth-protected).

## 4.2. Implementation
-   **Broker Struct** (`internal/sse`):
    -   Maintains a set of active `chan []byte` (clients).
    -   **Concurrency Model**: Single internal goroutine event loop owns all mutable state (client set + throttle timestamp). Public methods communicate via channels — no mutexes required.
    -   Buffered channels: publish buffer (256), per-client buffer (64).
    -   Clients with full buffers are skipped (non-blocking broadcast).
-   **Lifecycle**:
    -   `NewBroker(graphThrottle)`: Starts event loop goroutine.
    -   `Subscribe()`: Returns a buffered client channel.
    -   `Unsubscribe(ch)`: Removes and closes client channel.
    -   `Close()`: Stops loop, closes all client channels, drains gracefully.
-   **Shutdown**: SSE broker is closed *before* HTTP server shutdown, so active SSE connections drain cleanly.

## 4.3. Event Types
Format: `event: <type>\ndata: <json>\n\n`

### Events
1.  **`note.created`**
    ```json
    { "path": "new-note.md" }
    ```
2.  **`note.updated`**
    ```json
    { "path": "existing.md" }
    ```
3.  **`note.deleted`**
    ```json
    { "path": "removed.md" }
    ```
4.  **`graph.updated`** (Throttled, 2s minimum interval)
    -   Emitted alongside note events but deduplicated by time.
    -   Signal to frontend to refresh the graph structure.

## 4.4. Client Handling
-   Frontend (`EventSource`) auto-reconnects on drop.
-   Server handles `Context.Done()` to clean up disconnected clients.
-   Frontend invalidates React Query caches on received events.

## 4.5. Testing Strategy

### Unit Tests
-   **Broker**:
    -   Test adding and removing clients.
    -   Verify broadcasting sends message to all active clients.
    -   Ensure thread safety (race detector) when adding/removing clients concurrently.
    -   Test graph throttle behavior (events within 2s are coalesced).
    -   Test buffer overflow (client with full buffer is skipped, not blocked).

### Integration Tests
-   **Connection**: Connect a dummy client -> Trigger internal event -> Verify client receives SSE message with correct format.
-   **Lifecycle**: Verify client disconnect cleans up resources in Broker.

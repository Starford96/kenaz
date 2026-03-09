# 2. Indexing & Search Strategy

**Goal**: Fast metadata access and full-text search using SQLite.

## 2.1. Schema (`internal/index`)
**Driver**: `mattn/go-sqlite3`
**Mode**: WAL (Write-Ahead Logging) with 5-second busy timeout, foreign keys enabled.
**Connection**: Single connection (`MaxOpenConns=1`) for thread safety.

### Tables
1.  **`notes`** (Metadata + Content)
    -   `path` (TEXT PRIMARY KEY)
    -   `title` (TEXT NOT NULL DEFAULT '')
    -   `checksum` (TEXT NOT NULL DEFAULT '')
    -   `tags` (TEXT NOT NULL DEFAULT '[]', JSON array)
    -   `body` (TEXT NOT NULL DEFAULT '')
    -   `updated_at` (DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)

2.  **`links`** (Graph Edges)
    -   `source` (TEXT NOT NULL)
    -   `target` (TEXT NOT NULL)
    -   `type` (TEXT NOT NULL DEFAULT 'inline')
    -   UNIQUE(source, target)
    -   Indexes: `idx_links_source`, `idx_links_target`

3.  **`files_fts`** (Full Text Search - FTS5, build-tagged)
    -   `path` (UNINDEXED)
    -   `title`
    -   `body`
    -   `tags`
    -   Tokenizers: `unicode61 remove_diacritics 2`
    -   Fallback: When built without `-tags sqlite_fts5`, search uses `LIKE` queries instead.

## 2.2. Indexer Service
-   **Startup Sync**:
    -   Walk the `vault` directory.
    -   For each file:
        -   Calculate SHA-256 hash.
        -   Check DB: if missing or hash differs -> Parse & Upsert (single transaction: notes + links + FTS).
        -   If file in DB but not on disk -> Delete from all tables.
-   **Watcher (Real-time)** (`internal/index/watcher.go`):
    -   Library: `fsnotify/fsnotify`.
    -   Events:
        -   `Create/Write`: Parse file, upsert to index, trigger SSE callback.
        -   `Rename`: Delete old path immediately, debounced reconciliation (200ms) indexes new path.
        -   `Remove`: Delete from all tables.
    -   Dynamic directory watching: new directories added to fsnotify at runtime.
    -   `indexNewDir`: indexes `.md` files in directories that arrive with content.

## 2.3. Search Logic
-   **Full Text** (FTS5):
    ```sql
    SELECT path, snippet(files_fts, 2, '<b>', '</b>', '...', 64)
    FROM files_fts
    WHERE files_fts MATCH ?
    ORDER BY rank;
    ```
-   **Full Text** (LIKE fallback, no FTS5):
    ```sql
    SELECT path FROM notes WHERE title LIKE ? OR body LIKE ?;
    ```
-   **Backlinks**:
    ```sql
    SELECT source FROM links WHERE target = ?;
    ```
-   **Graph**:
    Returns all nodes (path, title, tags) and links (source, target) for visualization.

## 2.4. Testing Strategy

### Unit Tests
-   **Schema**: Verify all tables (including FTS5 virtual table) are created correctly on empty DB.
-   **CRUD**: Test Insert/Update/Delete of note metadata and links.
-   **Search**:
    -   Test exact match, prefix match, and partial match searches.
    -   Verify FTS5 tokenization handles Cyrillic characters correctly.
    -   Test Backlinks query returns correct sources.

### Integration Tests
-   **Sync Logic**:
    -   Create a known state on disk -> Run Sync -> Verify DB matches disk.
    -   Delete file on disk -> Run Sync -> Verify DB entry removed.
-   **Watcher**:
    -   Start watcher -> Create file -> Wait -> Verify DB update.
    -   Modify file -> Wait -> Verify metadata/content update in DB.
    -   Create new subdirectory -> Wait -> Verify dynamically watched.
    -   Rename file -> Wait 200ms -> Verify reconciliation.

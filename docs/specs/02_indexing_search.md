# 2. Indexing & Search Strategy

**Goal**: Fast metadata access and full-text search using SQLite.

## 2.1. Schema (`internal/index`)
**Driver**: `mattn/go-sqlite3`

### Tables
1.  **`notes`** (Metadata)
    -   `path` (TEXT PRIMARY KEY)
    -   `checksum` (TEXT)
    -   `updated_at` (DATETIME)
    -   `tags` (JSON or TEXT array)

2.  **`links`** (Graph Edges)
    -   `source` (TEXT, FK references notes.path)
    -   `target` (TEXT)
    -   `type` (TEXT, e.g., "inline", "frontmatter")
    -   UNIQUE(source, target)

3.  **`files_fts`** (Full Text Search - FTS5)
    -   `path` (UNINDEXED)
    -   `title`
    -   `body`
    -   `tags`
    -   Tokenizers: `unicode61 remove_diacritics 2`

## 2.2. Indexer Service
-   **Startup Sync**:
    -   Walk the `vault` directory.
    -   For each file:
        -   Calculate hash.
        -   Check DB: if missing or hash differs -> Parse & Upsert.
        -   If file in DB but not on disk -> Delete.
-   **Watcher (Real-time)**:
    -   Library: `fsnotify/fsnotify`.
    -   Events:
        -   `Write`: Reparse file, update `notes`, `links`, and `files_fts`.
        -   `Rename`: Update `path` in `notes` table. ideally update backlinks in other files (Advanced).
        -   `Remove`: Delete from all tables.

## 2.3. Search Logic
-   **Full Text**:
    ```sql
    SELECT path, snippet(files_fts, 2, '<b>', '</b>', '...', 64)
    FROM files_fts
    WHERE files_fts MATCH ?
    ORDER BY rank;
    ```
-   **Backlinks**:
    ```sql
    SELECT source FROM links WHERE target = ?;
    ```

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

# Kenaz Implementation Progress

## Completed Batches

### Batch 1 — Scaffold & Config ✅
- Go module, CLI entry point, config structs (vault, SQLite, auth)
- Chi router with health endpoints, graceful shutdown
- Makefile, Dockerfile, docker-compose, .env.example

### Batch 2 — Storage & Parsing ✅
- Domain models (Note, NoteMetadata, Link)
- Storage provider interface + filesystem implementation (atomic writes, path validation)
- Markdown parser: YAML frontmatter, wikilinks, tags, title derivation
- Unit tests for parser and storage

### Batch 3 — SQLite Index ✅
- SQLite schema: notes, links, files_fts (FTS5)
- Repository: upsert/delete, full-text search, backlinks
- Startup sync from vault to DB
- File watcher (fsnotify) for real-time index updates
- Build-tagged FTS5 with LIKE fallback

### Batch 4 — REST API ✅
- Notes CRUD: GET/POST/PUT/DELETE /api/notes
- Search: GET /api/search
- Graph: GET /api/graph
- Bearer token auth middleware (disabled when empty)
- Optimistic concurrency (If-Match checksum, 409 on mismatch)
- 16 handler tests

### Batch 5 — SSE Events & MCP Server ✅
- SSE broker with client lifecycle management
- GET /api/events endpoint (note.created/updated/deleted, graph.updated)
- Graph event throttling (2s)
- Watcher → SSE callback wiring
- MCP server (stdio): search_notes, read_note, create_note, list_notes, get_backlinks
- CLI subcommands: `kenaz serve` (HTTP, default) and `kenaz mcp` (stdio)
- SSE broker tests (subscribe, publish, throttle, handler, buffer overflow)
- MCP tool tests (create, read, list, backlinks, missing note)

## Remaining

### Batch 6 — Frontend (Spec 06)
- React + Ant Design UI
- Note editor, search, graph visualization
- Real-time sync via SSE

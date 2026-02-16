# Kenaz Progress

_Last updated: 2026-02-16 (UTC)_

## Current Status

Project is in active backend implementation.

### Completed

1. **Batch 1 — Scaffold**
   - Go service scaffold created from template approach.
   - HTTP server baseline, config loading, Makefile, Docker artifacts.
   - Commit: `1ee93c6`

2. **Agent Rules / Conventions**
   - Repository execution rules and commit conventions defined.
   - Commit: `37692f9`

3. **Batch 2 — Storage & Parsing (Spec 01)**
   - Domain models for notes.
   - Filesystem storage provider with safe path handling and atomic writes.
   - Markdown parser for frontmatter and wikilinks.
   - Unit tests for parser/storage.
   - Commit: `71dd083`

4. **Batch 3 — Indexing & Search (Spec 02)**
   - SQLite index layer and schema.
   - FTS-based search and backlinks support.
   - Startup sync + filesystem watcher.
   - Commit: `704b0ed`

5. **Compatibility Fix**
   - Tests now pass both with and without `sqlite_fts5` build tag.
   - Commit: `24ab572`

## Verification Snapshot

- `go test ./...` — passing
- `go test -tags sqlite_fts5 ./...` — passing

## Next Planned Batch

### Batch 5 — Realtime + MCP (Specs 04 & 05)

Planned scope:
- SSE broker and `/api/events`
- Event flow from watcher/index updates
- MCP server tools (`search_notes`, `read_note`, `create_note`, `list_notes`, `get_backlinks`)

## Backlog

- Frontend implementation according to Spec 06

## Latest Update

6. **Batch 4 — REST API (Spec 03)**
   - Notes CRUD endpoints implemented.
   - Search and graph endpoints implemented.
   - Bearer auth middleware added (disabled when token is empty).
   - Optimistic concurrency support via `If-Match` checksum.
   - API integration tests added.
   - Commit: `8ca3cef`

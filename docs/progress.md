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

### Auth Hardening ✅
- Auth mode config: `disabled` (default, backward-safe) and `token` (fail-fast if token empty)
- Config validation catches invalid mode and empty token at startup
- All protected endpoints under `/api` (including `/api/events` SSE) share auth middleware
- Health endpoints (`/health/live`, `/health/ready`) remain public
- 6 config validation tests + 3 SSE auth tests

### Watcher Robustness ✅
- New directories dynamically added to fsnotify watch list at runtime
- Rename events: old path deleted immediately, debounced reconciliation (200ms) indexes new path and cleans stragglers
- `indexNewDir` indexes .md files in directories that arrive with content
- 4 watcher integration tests: new file, new subdir, delete, rename reconciliation

### FE-1 — Frontend App Shell ✅
- Vite + React 18 + TypeScript scaffold in `frontend/`
- Obsidian-inspired dark theme (Ant Design v5, compact algorithm)
- 3-pane layout: left sidebar (file tree), center tabbed viewer, right context panel (backlinks + outline)
- API client (axios) with auth token injection for all backend endpoints
- State: @tanstack/react-query for server data, zustand for UI (tabs, sidebar, panels)
- SSE hook: auto-invalidates react-query caches on note/graph events
- Quick search modal (Cmd/Ctrl+K) with debounced FTS, keyboard nav
- Wiki-link rendering with clickable [[links]] and CSS styling
- Vite dev proxy to backend on :8080
- `npm run build` passes (tsc + vite)

### API Contract Generation ✅
- OpenAPI 3.1 spec at `api/schema/kenaz/openapi.yaml` **generated from backend code**
- `swaggo/swag` v1.16.4 annotations on all handlers and named DTOs in `internal/api/`
- Swagger 2.0 → OpenAPI 3.1 conversion via `scripts/swagger2openapi.mjs`
- `make openapi` runs swag + conversion pipeline
- `make openapi-check` drift guard: fails if generated spec differs from committed
- `openapi-typescript` generates `frontend/src/api/schema.d.ts` from spec
- `openapi-fetch` typed, spec-driven fetch client
- `make client-gen` chains `openapi` → frontend type generation
- All frontend types flow from backend annotations (single source of truth)

### FE-2A — Markdown Editing Workflow ✅
- CodeMirror 6 Markdown editor with oneDark theme and Obsidian-like styling
- Read/edit toggle per note (toolbar button + Cmd/Ctrl+E keyboard shortcut)
- Save via Cmd/Ctrl+S (both global and CodeMirror keymap)
- react-query mutations with optimistic updates and checksum-based concurrency
- Create note modal (+ button in sidebar) with auto `.md` suffix and title derivation
- Ant Design `App.useApp()` integration for toast messages (save/error feedback)

### FE-2B — Graph, Autocomplete & Attachments ✅
- Wikilink `[[` autocomplete in CodeMirror editor using live notes list from API
- Interactive 2D force-directed graph view (react-force-graph-2d) with node click navigation
- Graph lazy-loaded as separate chunk (~190KB) via React.lazy
- Graph accessible via sidebar button, opens in a dedicated tab
- Attachment upload via paste/drag in editor: inserts placeholder during upload, replaces with `![name](url)` on success
- Custom autocomplete tooltip styling matching dark theme

### FE-3A — Tab Persistence, Command Palette & Typography ✅
- Tab persistence via zustand `persist` middleware (localStorage key `kenaz-ui`)
- Sidebar collapsed and context panel state also persisted across sessions
- Stale/deleted note tabs show error state with close button
- Command palette (Cmd/Ctrl+K) with mixed items: files (FTS search) + commands (New Note, Open Graph, Toggle Sidebar, Toggle Context Panel)
- Icons per item kind, keyboard navigation, spotlight-style dark modal with footer hints
- "New Note" command dispatches custom event to open create modal from sidebar
- Empty-query palette shows commands + recent notes for quick access
- Markdown preview typography: headers with border-bottom, blockquotes with purple left border, styled code blocks (inline + fenced), lists with muted markers, tables, HR, images, links
- `.md-preview` CSS class applied to read-mode note content
- Editor and preview share consistent line-height rhythm (1.75)

### Documentation Hardening — MCP + Note Contract ✅
- Added `docs/note_format.md` as canonical note content contract
- Added frontmatter schema guidance and markdown examples for agent-created notes
- Expanded MCP spec with concrete tool payload examples
- Linked note format contract from README and AGENTS rules

## Remaining

### Future Enhancements
- Graph: zoom controls, search/highlight, cluster coloring
- Editor: split-pane live preview
- Bulk operations (tag, move, delete)
- Settings panel (vault path, auth token, theme)

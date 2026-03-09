# Kenaz Architecture

High-level architecture overview for the Kenaz local-first Markdown knowledge base.

## System Overview

Kenaz is a single-binary application with an embedded React frontend. It manages a vault of plain `.md` files, indexes them in SQLite with FTS5, and exposes both HTTP and MCP interfaces.

```
┌─────────────────────────────────────────────────────────────┐
│                       Kenaz Binary                          │
│                                                             │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌─────────┐ │
│  │  CLI     │   │  HTTP    │   │  MCP     │   │  SSE    │ │
│  │ (urfave) │   │ (Chi)    │   │ (stdio)  │   │ Broker  │ │
│  └────┬─────┘   └────┬─────┘   └────┬─────┘   └────┬────┘ │
│       │              │              │              │        │
│       ▼              ▼              ▼              │        │
│  ┌────────────────────────────────────────────┐   │        │
│  │             Note Service                   │◄──┘        │
│  │         (business logic layer)             │            │
│  └──────┬───────────────┬─────────────────────┘            │
│         │               │                                   │
│   ┌─────▼─────┐   ┌────▼──────┐                           │
│   │  Storage  │   │  Index    │                            │
│   │   (FS)    │   │ (SQLite)  │                            │
│   └─────┬─────┘   └────┬──────┘                            │
│         │               │                                   │
│   ┌─────▼─────┐   ┌────▼──────┐   ┌─────────────┐        │
│   │   Vault   │   │ kenaz.db  │   │  Watcher    │        │
│   │  (*.md)   │   │  (FTS5)   │   │ (fsnotify)  │        │
│   └───────────┘   └───────────┘   └─────────────┘        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    React Frontend                           │
│  (embedded in binary or Vite dev server)                    │
│                                                             │
│  React 19 · Ant Design v6 · CodeMirror 6                   │
│  Zustand (UI state) · React Query (server state)           │
│  openapi-fetch (typed API client) · SSE (real-time)        │
└─────────────────────────────────────────────────────────────┘
```

## Operational Modes

The binary has two CLI subcommands:

| Command | Transport | Purpose |
|---------|-----------|---------|
| `kenaz serve` (default) | HTTP :8080 | REST API + embedded SPA + SSE events |
| `kenaz mcp` | stdio | MCP server for LLM integration (Claude, Cursor, etc.) |

## Layered Architecture

### 1. Transport Layer

**HTTP (Chi v5)** — REST API under `/api`, SPA fallback, static attachments.

Middleware stack (in order):
1. `RequestID` — unique request tracking
2. `RealIP` — extract client IP
3. `SlogRequestLogger` — structured JSON logging
4. `Recoverer` — panic recovery
5. `AuthMiddleware` — Bearer token (configurable: `disabled`/`token`)

**MCP (mark3labs/mcp-go)** — stdio JSON-RPC for LLM tools.

### 2. Service Layer (`internal/noteservice`)

Central business logic. All transport handlers delegate to this service.

Operations: create, read, update, delete notes; rename/move; search; graph; backlinks; list directories.

### 3. Storage Layer (`internal/storage`)

Filesystem abstraction for vault operations.

Key safety features:
- **Atomic writes**: temp file → fsync → rename (prevents corruption)
- **Path validation**: blocks directory traversal (`..`)
- **Configurable exclusions**: `.git`, `attachments`, etc.

### 4. Index Layer (`internal/index`)

SQLite database with WAL mode (single connection for thread safety).

Schema:
```
notes (path PK, title, body, checksum, tags, updated_at)
  │
  ├── links (source FK → notes, target, type, UNIQUE(source,target))
  │
  └── files_fts (FTS5: path, title, body, tags)
                  tokenize = unicode61 remove_diacritics 2
```

## Data Flow

### Write Path (note creation/update)

```
Client → HTTP PUT /api/notes/{path}
  → Auth middleware (Bearer token check)
  → Handler validates If-Match checksum (optimistic concurrency)
  → NoteService.Update()
    → Storage.Write() (atomic: temp → fsync → rename)
    → Watcher detects fsnotify event
    → Parser extracts frontmatter, wikilinks, tags
    → Index.Upsert() (SQLite TX: notes + links + FTS5)
    → SSE Broker broadcasts note.updated + graph.updated
  → Frontend receives SSE event → invalidates React Query cache
```

### Read Path (search)

```
Client → HTTP GET /api/search?q=term
  → Auth middleware
  → Handler delegates to NoteService.Search()
    → Index.Search() → FTS5 MATCH query with snippets
  → JSON response with ranked results
```

### Startup Sync

```
Application start
  → Walk vault directory
  → For each .md file: compute SHA-256 checksum
    → If not in DB or checksum differs → parse + upsert
    → If in DB but not on disk → delete from index
  → Start fsnotify watcher for real-time sync
```

## Real-Time Updates

**SSE Broker** (`internal/sse`) — single goroutine event loop, no mutexes.

```
Watcher event → parse → index upsert → SSE callback
  → Broker broadcasts to all connected clients:
    - note.created  {path, title}
    - note.updated  {path, checksum}
    - note.deleted  {path}
    - graph.updated (throttled, 2s minimum interval)
```

Frontend `EventSource` auto-reconnects on drop. Server cleans up on `Context.Done()`.

## Frontend Architecture

Three-pane Obsidian-inspired layout:

```
┌──────────┬────────────────────────┬───────────────┐
│          │       Tab Bar          │               │
│ Sidebar  ├────────────────────────┤ Context Panel │
│          │                        │  (backlinks)  │
│ File tree│  NoteView / GraphView  │  (outline)    │
│ Search   │                        │               │
│ Actions  │  Read mode (Markdown)  │               │
│          │  Edit mode (CodeMirror)│               │
└──────────┴────────────────────────┴───────────────┘
```

**State management:**
- **Zustand** — UI state (tabs, sidebar, panels) persisted to `localStorage`
- **React Query** — server data with cache invalidation on SSE events
- **URL sync** — active note path synced to browser URL

**API communication:**
- `openapi-fetch` typed client generated from OpenAPI spec
- All types flow from backend swaggo annotations → OpenAPI 3.1 → TypeScript types
- Single source of truth for the API contract

**Key features:**
- `Cmd/Ctrl+K` command palette with FTS search + commands
- `[[wikilink]]` autocomplete in CodeMirror
- Attachment upload via paste/drag-and-drop
- Mobile-responsive design (drawer-based sidebar/context panel)
- Note export (single .md or folder as .zip)

## MCP Server

Stdio transport with 9 tools for LLM integration:

| Tool | Purpose |
|------|---------|
| `search_notes` | Full-text search with snippets |
| `read_note` | Read note content |
| `create_note` | Create with canonical format |
| `update_note` | Update with optional optimistic concurrency |
| `delete_note` | Delete a note |
| `list_notes` | List all or folder-specific notes |
| `get_backlinks` | Incoming links to a note |
| `get_note_contract` | Returns canonical note format contract |
| `upload_asset` | Download URL and save as vault attachment |

Resource: `kenaz://note-format` — exposes note format contract as text/markdown.

## Configuration

Single YAML file with environment variable expansion (`${VAR:-default}`):

```yaml
app:
  log_level: INFO
  http:
    port: 8080

vault:
  path: ./vault
  ignore_dirs: [.git, attachments]

sqlite:
  path: ./kenaz.db

auth:
  mode: disabled | token
  token: <bearer-token>

frontend:
  enabled: true
  dist_path: ./frontend/dist
```

## Build & Deployment

**Docker** — multi-stage build:
1. Go backend (CGO_ENABLED=1, `-tags sqlite_fts5`)
2. Node frontend (npm ci + vite build)
3. Alpine runtime (~minimal image)

**Local registry:** `192.168.48.58:5005`

Key constraints:
- CGO_ENABLED=1 required for SQLite FTS5
- Build tag `-tags sqlite_fts5` must be present (fallback: LIKE-based search)
- SQLite single-connection WAL mode (no external DB required)

## Technology Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Language | Go | 1.25 |
| Router | Chi | v5 |
| Database | SQLite + FTS5 | via mattn/go-sqlite3 |
| File watcher | fsnotify | latest |
| CLI | urfave/cli | v3 |
| MCP | mark3labs/mcp-go | latest |
| Frontend | React | 19 |
| UI framework | Ant Design | v6 |
| Editor | CodeMirror | 6 |
| Build | Vite | 7 |
| Server state | React Query | v5 |
| UI state | Zustand | v5 |
| API client | openapi-fetch | v0.17 |
| Graph | react-force-graph-2d | v1.29 |

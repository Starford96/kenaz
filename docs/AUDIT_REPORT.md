# Codebase Audit Report

**Project**: Kenaz — Local-first knowledge base with Markdown storage, full-text search, and graph visualization
**Date**: 2026-02-26
**Scope**: Full codebase (Go backend + React frontend)
**Aspects run**: architecture, code-quality, security, errors, tests, dependencies, performance, simplification, debt, go-specific
**Health Score**: 52/100 — Grade D (Needs work)

---

## Executive Summary

- **Total findings**: 47
- **P0 Critical**: 5 | **P1 Should fix**: 16 | **P2 Improve**: 18 | **P3 Nice to have**: 8
- **Top 3 areas needing attention**:
  1. **Duplicated business logic** — API service and MCP server independently implement CRUD flows with behavioral divergence (empty checksums, inconsistent error handling)
  2. **Security hardening** — HTTP server lacks timeouts (Slowloris DoS), CORS `*` on SSE, auth disabled by default in Docker, unbounded request bodies
  3. **Error handling fragility** — String-based error matching, swallowed errors in critical paths (MCP, index ops, JSON encoding), no sentinel errors

---

## Strengths

1. **Clean package structure** — Acyclic import graph, clear separation of concerns (`api/`, `index/`, `storage/`, `parser/`, `sse/`, `mcpserver/`), proper use of `internal/`
2. **Atomic file writes** — `storage.FS.Write()` uses tmp→fsync→rename, preventing data corruption
3. **SSE broker design** — Channel-based single-goroutine event loop, no mutexes, clean concurrency model with graph event throttling
4. **Path traversal protection** — Both `storage.FS.safePath()` and `AttachmentHandler.safeName()` properly validate and test for directory traversal
5. **SQL parameterization** — All dynamic values use `?` placeholders; no user input in SQL queries
6. **Type-safe API client** — Frontend uses `openapi-fetch` with generated types from OpenAPI spec
7. **No global state** — All dependencies injected via constructors; no `init()` functions; compiled regexps are the only package-level state
8. **Well-structured tests** — Proper use of `t.Helper()`, `t.TempDir()`, `t.Cleanup()`; security tests for path traversal; FTS5 build tag separation

---

## P0 — Critical

### 1. HTTP Server Missing Read/Write/Idle Timeouts
- **Aspect**: Security
- **Location**: `internal/entry.go:153-156`
- **Tag**: [EXISTING]
- **Issue**: `http.Server` has no `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, or `IdleTimeout`. Slowloris-style attacks can exhaust goroutines and file descriptors with ~10k slow connections.
- **Impact**: Denial of service for any network-exposed deployment
- **Fix**: Set `ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`, `IdleTimeout: 120s`. Leave `WriteTimeout` at 0 for SSE compatibility, or use `http.TimeoutHandler` wrapper on non-SSE routes.

### 2. go-sqlite3 CVE-2025-6965 (CVSS 9.8 Critical)
- **Aspect**: Dependencies
- **Location**: `go.mod:11` — `github.com/mattn/go-sqlite3@v1.14.24`
- **Tag**: [EXISTING]
- **Issue**: Bundled SQLite is vulnerable to memory corruption via aggregate term overflow (CVE-2025-6965). Latest go-sqlite3 v1.14.32+ bundles SQLite >= 3.50.2 with the fix.
- **Impact**: Potential memory corruption via crafted SQL queries
- **Fix**: `go get github.com/mattn/go-sqlite3@latest`

### 3. MCP Server Creates Notes with Empty Checksum (Behavioral Bug)
- **Aspect**: Code Quality, Architecture
- **Location**: `internal/mcpserver/server.go:166` vs `internal/api/service.go:90-91`
- **Tag**: [EXISTING]
- **Issue**: MCP `createNote` stores `Checksum: ""` in the index while API `CreateNote` computes SHA-256. This breaks optimistic concurrency (checksum-based conflict detection) for MCP-created notes.
- **Impact**: Data integrity — concurrent edits via API after MCP creation skip conflict detection
- **Fix**: Extract shared domain service ensuring consistent checksum computation (also fixes #4)

### 4. `storage.FS.List()` Reads Every File to Compute Checksums
- **Aspect**: Performance
- **Location**: `internal/storage/fs.go:60-93`
- **Tag**: [EXISTING]
- **Issue**: `List()` calls `os.ReadFile()` on every `.md` file just to compute SHA-256 checksums. For 1,000 notes at 5KB average, this reads ~5MB into memory on every startup sync, reconcile, and MCP `listNotes` call.
- **Impact**: Startup latency and memory spikes scale linearly with vault size
- **Fix**: Use `os.Stat()` mtime+size as cheap change proxy; only read file content when mtime/size changed. The index already stores checksums for comparison.

### 5. N+1 Query Pattern in `index.Sync()`
- **Aspect**: Performance
- **Location**: `internal/index/sync.go:26-48`
- **Tag**: [EXISTING]
- **Issue**: For every file on disk, `Sync()` issues a separate `SELECT checksum FROM notes WHERE path = ?`. With N files, that's N individual SQLite queries.
- **Impact**: Startup time grows linearly with vault size; unnecessary round-trips
- **Fix**: Single `SELECT path, checksum FROM notes` → `map[string]string`, then in-memory comparison

---

## P1 — Should Fix

### 6. Duplicated Business Logic: API Service vs MCP Server
- **Aspect**: Architecture, Code Quality, Simplification
- **Location**: `internal/api/service.go:47-165` vs `internal/mcpserver/server.go:136-218`
- **Tag**: [EXISTING]
- **Issue**: Create/update/delete/index flows are independently implemented in both layers (~80 lines duplicated). Already diverged: MCP stores empty checksums (#3), MCP swallows parse/index errors (#10).
- **Impact**: Bug fixes must be applied in two places; behavioral drift already causing issues
- **Fix**: Extract `internal/noteservice/` with shared `Create/Update/Delete` methods. Both API and MCP become thin transport adapters.

### 7. No Sentinel Errors — String-Based Error Matching
- **Aspect**: Errors, Code Quality, Go
- **Location**: `internal/api/service.go:81,96,100` + `internal/api/handlers.go:119,174-178`
- **Tag**: [EXISTING]
- **Issue**: Zero `var Err... = errors.New(...)` in the entire codebase. Error dispatch uses `err.Error() == "already exists"` / `"not found"` / `"conflict"`. Wrapped errors would silently break this matching.
- **Impact**: Fragile — any error message change silently returns 500 instead of 409/404/409
- **Fix**: Define `var ErrNotFound`, `ErrConflict`, `ErrAlreadyExists`; use `errors.Is()` in handlers

### 8. Internal Error Messages Leaked in API Responses
- **Aspect**: Security, Errors
- **Location**: `internal/api/handlers.go:59,123,180,229,249`
- **Tag**: [EXISTING]
- **Issue**: Raw `err.Error()` passed to clients exposes file paths (`/app/vault/notes/secret.md`), SQLite errors, and package names.
- **Impact**: Information disclosure to attackers
- **Fix**: Return generic messages (`"internal error"`, `"not found"`) to clients; log details with `slog.Error()`

### 9. SSE Endpoint Sets `Access-Control-Allow-Origin: *`
- **Aspect**: Security
- **Location**: `internal/sse/broker.go:219`
- **Tag**: [EXISTING]
- **Issue**: Any website can connect to the SSE endpoint and receive real-time note activity. `EventSource` API doesn't support custom headers, so Bearer token auth doesn't protect SSE when CORS is `*`.
- **Impact**: Cross-origin data leakage of note activity stream
- **Fix**: Remove `*`; restrict to configured frontend origin or use per-connection query token

### 10. MCP Server Swallows Parse/Index/Delete Errors
- **Aspect**: Errors
- **Location**: `internal/mcpserver/server.go:157,163,202,209,228`
- **Tag**: [EXISTING]
- **Issue**: `_ = parser.Parse(data)`, `_ = s.db.UpsertNote(...)`, `_ = s.db.DeleteNote(path)` — parse and index errors silently discarded in all MCP write operations.
- **Impact**: Notes written to disk but not indexed; search/graph missing data; LLM caller unaware of failure
- **Fix**: Return errors as MCP tool errors

### 11. Auth Disabled by Default in Docker
- **Aspect**: Security
- **Location**: `config/config.yaml:12`, `build/Dockerfile:33`, `docker-compose.yaml`
- **Tag**: [EXISTING]
- **Issue**: Default `AUTH_MODE=disabled` with Docker binding to `0.0.0.0:8080`. Network-accessible full CRUD without authentication.
- **Impact**: Anyone on the network gets full vault access
- **Fix**: Bind Docker to `127.0.0.1:8080` by default, or add startup warning when auth disabled on non-loopback

### 12. Unbounded `io.ReadAll` on UpdateNote Request Body
- **Aspect**: Security
- **Location**: `internal/api/handlers.go:150`
- **Tag**: [EXISTING]
- **Issue**: No size limit on `io.ReadAll(r.Body)` — multi-GB request exhausts server memory. `CreateNote` also unbounded via `json.NewDecoder`.
- **Impact**: DoS via large request body
- **Fix**: `r.Body = http.MaxBytesReader(w, r.Body, 10<<20)` before reading

### 13. Watcher Goroutine Return Value Silently Discarded
- **Aspect**: Errors, Go
- **Location**: `internal/entry.go:163-168`
- **Tag**: [EXISTING]
- **Issue**: `g.Go(func() error { index.Watch(...); return nil })` — `Watch` return value ignored; watcher startup failures (fsnotify, inotify limits) go undetected.
- **Impact**: File watcher silently stops; new changes never indexed, SSE events stop
- **Fix**: `return index.Watch(gCtx, db, store, ...)`

### 14. `GetChecksum`/`GetNote` Silently Convert DB Errors to Success
- **Aspect**: Errors, Go
- **Location**: `internal/index/repo.go:89-96,117-131`
- **Tag**: [EXISTING]
- **Issue**: Returns `("", nil)` / `(nil, nil)` for ALL errors (including DB failures), not just `sql.ErrNoRows`. DB corruption treated as "not found".
- **Impact**: Database errors masked; unnecessary re-indexing or data duplication
- **Fix**: `if errors.Is(err, sql.ErrNoRows) { return "", nil }; return "", fmt.Errorf("...: %w", err)`

### 15. API Handlers Don't Propagate `context.Context`
- **Aspect**: Go
- **Location**: `internal/api/handlers.go` (all handlers), `internal/api/service.go` (all methods)
- **Tag**: [EXISTING]
- **Issue**: `r.Context()` never passed to Service. Cancelled client connections don't abort in-flight storage/DB operations. No mechanism for request-scoped timeouts.
- **Impact**: Slow queries block goroutines even after client disconnect
- **Fix**: Add `context.Context` as first parameter to all `Service` and `storage.Provider` methods

### 16. Inconsistent `UpdatedAt` — Returns `time.Now()` Instead of File Mtime
- **Aspect**: Code Quality, Debt
- **Location**: `internal/api/service.go:72,162`, `internal/index/sync.go:73-78`
- **Tag**: [EXISTING]
- **Issue**: `GetNote` returns `UpdatedAt: time.Now()` instead of stored timestamp. Sync sets zero-value. Sort by "recently updated" is meaningless.
- **Impact**: Misleading timestamps; sorting by update time doesn't work correctly
- **Fix**: Propagate actual file `os.Stat().ModTime()` through the call chain

### 17. Duplicated `indexFile` Logic
- **Aspect**: Code Quality, Simplification
- **Location**: `internal/api/service.go:148-165` vs `internal/index/sync.go:65-80`
- **Tag**: [EXISTING]
- **Issue**: Nearly identical private functions (parse→checksum→NoteRow→upsert) maintained separately. `sync.indexFile` doesn't set `UpdatedAt`.
- **Impact**: Changes must be made in two places; already diverged
- **Fix**: Export single `index.IndexFile(db, path, data)` used by both

### 18. urfave/cli/v3 on Pre-Release Beta
- **Aspect**: Dependencies
- **Location**: `go.mod:12` — `v3.0.0-beta1`
- **Tag**: [EXISTING]
- **Issue**: 20+ releases behind stable v3.6.1. Missing bug fixes and security patches from 2 years of development.
- **Fix**: `go get github.com/urfave/cli/v3@latest`

### 19. Zero Frontend Tests
- **Aspect**: Tests
- **Location**: `frontend/` — no test files
- **Tag**: [EXISTING]
- **Issue**: Entire React 19 SPA (27 source files) has zero tests. All UI behavior is untested.
- **Impact**: Any refactor is a regression risk; no safety net
- **Fix**: Add Vitest + React Testing Library; prioritize Zustand stores, API client functions, critical user flows

### 20. Watcher Tests Use `time.Sleep` — Systematic Flakiness
- **Aspect**: Tests
- **Location**: `internal/index/watcher_test.go:52-152` — 8 hardcoded sleeps
- **Tag**: [EXISTING]
- **Issue**: Tests rely on 100-600ms `time.Sleep` for fsnotify event propagation. Flaky on slow CI; wastes time on fast machines.
- **Impact**: CI unreliability; developers ignore failures
- **Fix**: Replace with `require.Eventually` or polling helper with configurable timeout

### 21. `CreateNote`/`UpdateNote` Double-Read and Double-Parse
- **Aspect**: Performance
- **Location**: `internal/api/service.go:78-109`
- **Tag**: [EXISTING]
- **Issue**: Both operations call `indexFile(path, content)` (parse + hash) then `GetNote(path)` which re-reads from disk, re-parses, and re-hashes.
- **Impact**: 2x latency and CPU per write operation
- **Fix**: Build `NoteDetail` from already-parsed data in `indexFile()` return value

---

## P2 — Improve

### 22. Service Layer Mixed into Transport Package (`api/`)
- **Aspect**: Architecture
- **Location**: `internal/api/service.go`
- **Issue**: `api` package contains both HTTP handlers and business logic; cannot swap transport without duplicating logic
- **Fix**: Extract `internal/service/` or `internal/noteservice/` package

### 23. JSON Encoding Error Silently Dropped
- **Aspect**: Errors
- **Location**: `internal/api/json.go:11`
- **Issue**: `_ = json.NewEncoder(w).Encode(v)` — failed encoding produces empty/partial response with 200 status
- **Fix**: Log encoding errors; consider writing error to response if headers not yet flushed

### 24. Swallowed SQL DELETE Errors in Index Operations
- **Aspect**: Errors
- **Location**: `internal/index/repo.go:56,82-83`, `internal/index/fts_fts5.go:25,34-36`
- **Issue**: 5 `_, _ = tx.Exec("DELETE ...")` calls discard errors. Stale data persists in index.
- **Fix**: Check and return errors within transactions

### 25. Backlinks Error Silently Swallowed in GetNote
- **Aspect**: Errors
- **Location**: `internal/api/service.go:57`
- **Issue**: `bl, _ := s.db.Backlinks(path)` — DB failure returns empty backlinks with no indication
- **Fix**: At minimum log; consider returning error

### 26. SHA-256 Checksum Duplicated Across 4 Packages
- **Aspect**: Code Quality, Debt
- **Location**: `api/service.go:167`, `storage/fs.go:182`, `index/sync.go:70`, `mcpserver/server.go:190`
- **Issue**: Same 3-line function independently implemented in 4 places
- **Fix**: Create `internal/checksum/checksum.go` with single `Checksum([]byte) string`

### 27. Nil-to-Empty-Slice Normalization Repeated ~8 Times
- **Aspect**: Code Quality
- **Location**: `api/service.go`, `index/repo.go`, `mcpserver/server.go`
- **Issue**: `if tags == nil { tags = []string{} }` pattern scattered everywhere for JSON serialization
- **Fix**: Initialize target before `json.Unmarshal` or add `nonNilSlice()` helper

### 28. DTO Types Defined But Unused / Drifting
- **Aspect**: Code Quality
- **Location**: `internal/api/dto.go:23-69`
- **Issue**: `SearchResult`, `GraphNode`, `GraphLink`, `NoteDetailDTO`, `NoteListItemDTO` defined for Swagger but never serialized. Handlers use `map[string]interface{}`.
- **Fix**: Use typed response structs (`SearchResponse{Results: items}`) for compile-time safety

### 29. Dead Code: `models.Note`, `models.Link`, `WithLogger`, `LoadWithDefaults`, `MustLoad`
- **Aspect**: Code Quality, Debt
- **Location**: `models/note.go:7-32`, `option.go:20-25`, `pkg/config/config.go:40-55`
- **Issue**: 5+ exported types/functions never referenced anywhere
- **Fix**: Remove unused code

### 30. Tag Filtering Uses LIKE on JSON-Encoded Column
- **Aspect**: Performance, Code Quality
- **Location**: `internal/index/repo.go:150-153`
- **Issue**: `WHERE tags LIKE '%"tagname"%'` — full table scan, potential false positives
- **Fix**: Create `note_tags` junction table or use SQLite `json_each()`

### 31. Graph Query Loads Entire Dataset Unbounded
- **Aspect**: Performance
- **Location**: `internal/index/repo.go:199-242`
- **Issue**: `SELECT path, title FROM notes` + `SELECT source, target FROM links` with no LIMIT. Frontend force-graph has O(N^2) simulation cost.
- **Fix**: Add pagination, node limit, or neighborhood subgraph mode

### 32. `reconcileAfterRename` Re-reads Entire Vault
- **Aspect**: Performance
- **Location**: `internal/index/watcher.go:157-204`
- **Issue**: Single file rename triggers `store.List("")` which reads ALL files (compounds with #4)
- **Fix**: Track specific old/new paths from fsnotify events; reconcile only those files

### 33. GraphView Re-maps Data on Every Render (Unmemoized)
- **Aspect**: Performance
- **Location**: `frontend/src/components/GraphView.tsx:95-101`
- **Issue**: `graphData` computed inline without `useMemo` — new objects restart force simulation
- **Fix**: Wrap in `useMemo` with `[data]` dependency

### 34. `NoteView.tsx` Too Complex (371 Lines, 7+ Concerns)
- **Aspect**: Code Quality
- **Location**: `frontend/src/components/NoteView.tsx`
- **Issue**: Handles fetching, editing, drafts, mutations, shortcuts, frontmatter, wikilinks, links, scrolling, and rendering
- **Fix**: Extract `useNoteSave()` hook, `MarkdownPreview` component, `resolveNoteLink()` utility

### 35. Frontend `uploadAttachment` Bypasses Typed API Client
- **Aspect**: Code Quality
- **Location**: `frontend/src/api/notes.ts:92-109`
- **Issue**: Raw `fetch()` duplicates `baseUrl` and `token` from `client.ts`
- **Fix**: Export `baseUrl`/`token` from `client.ts` and reuse

### 36. Frontend CustomEvent Bus Instead of Store
- **Aspect**: Architecture
- **Location**: `frontend/src/components/SearchModal.tsx`, `Sidebar.tsx`
- **Issue**: `window.dispatchEvent(new CustomEvent("kenaz:create-note"))` bypasses React/Zustand data flow; not type-safe
- **Fix**: Use Zustand store action `setCreateModalOpen(true)`

### 37. Test Setup Duplicated Across 4 Packages
- **Aspect**: Tests
- **Location**: `api_test.go:22-60`, `server_test.go:15-39`, `index_test.go:9-24`, `watcher_test.go:16-35`
- **Issue**: ~15 lines of identical temp-dir + DB + cleanup setup in each
- **Fix**: Create `internal/testutil` with `TestDB(t)` and `TestVault(t)`

### 38. `index.DB` Has No Interface — Untestable in Isolation
- **Aspect**: Tests, Go
- **Location**: `internal/index/repo.go`
- **Issue**: `*index.DB` used everywhere; impossible to mock for unit tests
- **Fix**: Extract `NoteIndex` interface; accept it in service/MCP constructors

### 39. SQLite Connection Pool Not Configured
- **Aspect**: Go, Performance
- **Location**: `internal/index/schema.go:39`
- **Issue**: No `SetMaxOpenConns`/`SetMaxIdleConns`. Default unlimited connections can cause `SQLITE_BUSY` under concurrent access.
- **Fix**: `conn.SetMaxOpenConns(1); conn.SetMaxIdleConns(1)` for SQLite single-writer model

---

## P3 — Nice to Have

### 40. `Run()` Function Is 180 Lines (God Function)
- **Aspect**: Code Quality
- **Location**: `internal/entry.go:29-209`
- **Fix**: Extract `buildRouter()`, `serveFrontend()`, keep `Run()` as orchestrator

### 41. `pkg/config` Generic Loader in `pkg/` but Only Used Internally
- **Aspect**: Go, Architecture
- **Location**: `pkg/config/config.go`
- **Fix**: Move to `internal/config` or remove dead `LoadWithDefaults`/`MustLoad`

### 42. No Database Migration Strategy
- **Aspect**: Debt
- **Location**: `internal/index/schema.go:11-30`
- **Issue**: `CREATE TABLE IF NOT EXISTS` on every startup; no schema versioning
- **Fix**: Adopt lightweight migration tool (goose) or schema_version table

### 43. Missing Explicit `@codemirror/autocomplete` and `@codemirror/commands` Dependencies
- **Aspect**: Debt
- **Location**: `frontend/package.json`, `wikilinkComplete.ts`, `MarkdownEditor.tsx`
- **Issue**: Imported packages not listed; works via transitive deps only
- **Fix**: Add as explicit dependencies

### 44. Hardcoded Magic Numbers Throughout Codebase
- **Aspect**: Debt
- **Location**: Various — `limit: 1000`, `50<<20`, `200ms`, `2s`, `64`, `256`, `20`, `50`
- **Fix**: Extract named constants; consider making limits configurable

### 45. `map[string]interface{}` Used Instead of `any` (30+ Sites)
- **Aspect**: Debt
- **Location**: `parser.go`, `handlers.go`, `service.go`, `models/note.go`, etc.
- **Fix**: Replace `interface{}` with `any` for Go 1.18+ readability

### 46. Inline Styles Throughout Frontend Components
- **Aspect**: Code Quality
- **Location**: `App.tsx`, `NoteView.tsx`, `Sidebar.tsx`, `SearchModal.tsx`, etc.
- **Fix**: Extract common styles to module-level constants or CSS modules

### 47. Unused `useMediaQuery` Hook
- **Aspect**: Code Quality
- **Location**: `frontend/src/hooks/useMediaQuery.ts`
- **Fix**: Delete the file

---

## Aspect Summaries

| Aspect | Findings | Health |
|--------|----------|--------|
| Architecture | 4 | 🔴 P1 findings |
| Code Quality | 9 | 🔴 P0/P1 findings |
| Security | 5 | 🔴 P0/P1 findings |
| Error Handling | 7 | 🔴 P1 findings |
| Tests | 4 | 🔴 P1 findings |
| Dependencies | 3 | 🔴 P0/P1 findings |
| Performance | 8 | 🔴 P0 findings |
| Simplification | 2 | 🟡 P2 findings |
| Tech Debt | 4 | 🟡 P2/P3 findings |
| Go-Specific | 3 | 🔴 P1 findings |

Health: 🟢 = 0 findings or P3 only | 🟡 = P2 findings | 🔴 = P0/P1 findings

---

## Suggested Refactoring Order

1. **Upgrade go-sqlite3** (#2) — 5 minutes, fixes critical CVE
2. **Add HTTP server timeouts** (#1) — 10 minutes, fixes Slowloris DoS
3. **Extract shared domain service** (#6) — fixes P0 checksum bug (#3), P1 duplication, P1 error swallowing (#10), P1 indexFile duplication (#17), and P2 context propagation (#15) in one refactor
4. **Introduce sentinel errors** (#7) — small change with cascading quality improvement
5. **Fix error leaking in API responses** (#8) — quick security win
6. **Add `http.MaxBytesReader`** (#12) — 3-line fix per handler
7. **Fix SSE CORS** (#9) — single line change
8. **Fix `GetChecksum`/`GetNote` error handling** (#14) — small fix, big correctness gain
9. **Optimize `storage.List()` and `Sync()`** (#4, #5) — major startup perf improvement
10. **Return Watch error in errgroup** (#13) — one-line fix for silent watcher death
11. **Fix `UpdatedAt` to use file mtime** (#16) — correctness fix for timestamps
12. **Upgrade urfave/cli to stable** (#18) — routine dependency update
13. **Add frontend test infrastructure** (#19) — long-term investment in frontend quality

---

## Tools Recommended

| Tool | Purpose | Install |
|------|---------|---------|
| `govulncheck` | Go vulnerability scanner | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| `npm audit` | npm vulnerability scanner | Built-in to npm |
| `goose` | Database migration tool | `go install github.com/pressly/goose/v3/cmd/goose@latest` |
| `vitest` | Frontend test runner | `cd frontend && npm install -D vitest @testing-library/react` |
| `httprate` | Chi rate limiting | `go get github.com/go-chi/httprate` |

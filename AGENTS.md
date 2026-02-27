# Kenaz — Local-first Markdown knowledge base

Go backend (Chi router, SQLite FTS5, CGO_ENABLED=1) + React frontend (Vite, Ant Design).

## Build & Test

- `make build` — build binary (requires `-tags sqlite_fts5`)
- `make test` — run all tests
- `make lint` — golangci-lint (incremental from origin/main)
- `make fmt` — gofumpt
- `make openapi` — regenerate OpenAPI spec
- `make client-gen` — regenerate typed TS client
- Frontend: `cd frontend && npm run dev`

## Commit Convention

`<type>[scope]: <description>` — types: feat, fix, refactor, docs, build, ci, chore, test, perf, style, improvement, breaking, revert

## Note Authoring

Follow @docs/note_format.md. File/dir names: English only (kebab-case). Frontmatter keys: English. Values and body: any language including Cyrillic.

## Docker Registry

Local HTTP registry at `192.168.48.58:5005`. Push: `make docker-push`.


## Documentation

All docs in English only. See `docs/` for specs.

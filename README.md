# Kenaz ᚲ

**Local-first knowledge base with Markdown storage, full-text search, and graph visualization.**

Kenaz combines the simplicity of Markdown files with the power of modern search and visualization, serving as a robust backend for personal knowledge management.

## Quick Start

```bash
cp .env.example .env
make build
make run
```

## Health Checks

- `GET /health/live` — liveness probe
- `GET /health/ready` — readiness probe

## Configuration

See `config/config.yaml` and `.env.example` for available settings.

## Frontend

The web UI lives in `frontend/` (Vite + React + Ant Design).

```bash
cd frontend
npm install
npm run dev      # Dev server on http://localhost:5173 (proxies to backend)
npm run build    # Production build → frontend/dist/
```

Set `VITE_AUTH_TOKEN` in `frontend/.env` if the backend uses token auth mode.

## Development

```bash
make deps          # Download dependencies
make build         # Build binary
make test          # Run tests
make lint          # Run linter
make fmt           # Format code
make docker-build  # Build Docker image
make docker-up     # Start local environment
make help          # Show all targets
```

## Documentation

- [Project Vision](docs/project_vision.md)
- [Implementation Plan](docs/implementation_plan.md)
- [Storage & Parsing](docs/specs/01_storage_parsing.md)
- [Indexing & Search](docs/specs/02_indexing_search.md)
- [REST API](docs/specs/03_rest_api.md)
- [Real-time Updates](docs/specs/04_realtime_updates.md)
- [MCP Server](docs/specs/05_mcp_server.md)
- [Note Format Contract](docs/note_format.md)
- [Frontend](docs/specs/06_frontend.md)

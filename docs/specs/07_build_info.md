# 7. Build Info & Versioning

**Goal**: Expose build metadata (version, commit, build time) through all interfaces so that AI agents, operators, and automation tools can determine exactly what is deployed and detect when updates are available.

## 7.1. Problem

Currently there is no way to determine which version of Kenaz is running:

- `Makefile` injects `-X main.Version=$(VERSION)` via ldflags, but `var Version` is **not declared** in `cmd/app/main.go` — the injection has no effect.
- No `kenaz version` CLI subcommand exists (inconsistent with the `serve`/`mcp` pattern).
- No `/api/version` REST endpoint exists.
- MCP server exposes no version tool or resource beyond `kenaz://note-format`.
- Docker images carry no OCI labels (`org.opencontainers.image.revision`, etc.).
- `Makefile` defaults to `VERSION=latest`, so every build produces an identically-named image with no traceability.

This makes it impossible to:
- Know what is deployed without `docker inspect` + manual source cross-referencing.
- Automate update checks (compare running version vs. latest Git tag).
- Reproduce issues with a specific commit.

---

## 7.2. Build-Time Embedding

### 7.2.1. Variables in `cmd/app/main.go`

Declare the three package-level variables that ldflags will populate:

```go
// Build info — injected at compile time via ldflags.
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)
```

### 7.2.2. `Makefile` updates

Extend the existing `LDFLAGS` to capture commit and build timestamp alongside the existing version:

```makefile
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILT   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-s -w \
    -X main.Version=$(VERSION) \
    -X main.Commit=$(COMMIT) \
    -X main.BuildTime=$(BUILT)"
```

---

## 7.3. CLI `version` Subcommand

Add a `version` command to `cmd/app/main.go`, consistent with the existing `serve`/`mcp` pattern:

```go
{
    Name:  "version",
    Usage: "Print the version and build info",
    Action: func(ctx context.Context, cmd *cli.Command) error {
        fmt.Printf("kenaz %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
        return nil
    },
},
```

Example output:

```
$ kenaz version
kenaz v1.2.0 (commit: a3f1c9b, built: 2026-03-09T12:00:00Z)
```

---

## 7.4. REST Endpoint `GET /api/version`

Add to the router outside the `/api` auth group — public, same access level as `/health/live`.

### Request

```
GET /api/version
```

### Response `200 OK`

```json
{
  "version":   "v1.2.0",
  "commit":    "a3f1c9b",
  "built_at":  "2026-03-09T12:00:00Z"
}
```

### Handler

The handler reads the package-level variables injected at build time and returns them as JSON. No service or storage dependency required.

### Router registration (in `internal/app.go` or equivalent)

```go
r.Get("/api/version", handleVersion)
```

---

## 7.5. MCP Resource `kenaz://server-info`

Add a static MCP resource alongside the existing `kenaz://note-format`, so AI agents can read version info without HTTP access.

- **URI**: `kenaz://server-info`
- **MIME**: `application/json`
- **Content**: same shape as the REST response

```json
{
  "version":  "v1.2.0",
  "commit":   "a3f1c9b",
  "built_at": "2026-03-09T12:00:00Z"
}
```

Registration in `internal/mcpserver` follows the same pattern as the `kenaz://note-format` resource.

---

## 7.6. Docker OCI Labels

### `build/Dockerfile` additions

```dockerfile
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT=unknown

LABEL org.opencontainers.image.title="kenaz" \
      org.opencontainers.image.source="https://github.com/Starford96/kenaz" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILT}"
```

### `Makefile` `docker-push` target update

```makefile
docker-push:
	docker buildx build \
	  --platform $(PLATFORM) \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILT=$(BUILT) \
	  -t $(REGISTRY)/kenaz:$(VERSION) \
	  -f $(BUILD_DIR)/Dockerfile \
	  --push .
```

After this change, `docker inspect` will return:

```json
{
  "org.opencontainers.image.version":  "v1.2.0",
  "org.opencontainers.image.revision": "a3f1c9b",
  "org.opencontainers.image.created":  "2026-03-09T12:00:00Z"
}
```

---

## 7.7. Release Workflow

1. Tag the commit: `git tag v1.2.0 && git push origin v1.2.0`
2. Build and push: `make docker-push VERSION=v1.2.0`
3. Update `homelab/kenaz/docker-compose.yml` to pin the concrete tag:

```yaml
image: 192.168.48.58:5005/kenaz:v1.2.0
```

Using `latest` is acceptable only for local development; production deployments must pin a version.

---

## 7.8. Automated Update Check

Once all changes are in place, a homelab automation agent can check for updates as follows:

1. **Read running version** (MCP, no HTTP needed):
   ```
   Resource URI: kenaz://server-info
   ```
   — or via HTTP:
   ```
   GET http://<host>:8080/api/version
   ```

2. **Fetch latest release** from GitHub API:
   ```
   GET https://api.github.com/repos/Starford96/kenaz/releases/latest
   → .tag_name
   ```

3. **Compare** → notify if behind:
   ```
   🔔 kenaz update available
     Running: v1.1.0 (commit a3f1c9b, built 2026-02-01T10:00:00Z)
     Latest:  v1.2.0
     Release: https://github.com/Starford96/kenaz/releases/tag/v1.2.0
   ```

---

## 7.9. Testing Strategy

### Unit Tests

- **`kenaz version`**: verify output format matches `kenaz <Version> (commit: <Commit>, built: <BuildTime>)`.
- **`GET /api/version`**: verify `200 OK`, correct JSON fields (`version`, `commit`, `built_at`), no auth required.
- **MCP resource `kenaz://server-info`**: verify URI, MIME type (`application/json`), and payload shape.

### Integration Tests

- Build binary with explicit ldflags (`-X main.Version=test-v`), run `kenaz version`, assert output contains `test-v`.
- Start server, call `GET /api/version`, assert response matches injected values.

---

## 7.10. Summary

| Gap | Fix |
|-----|-----|
| `var Version` not declared — ldflags injection is a no-op | Declare `Version`, `Commit`, `BuildTime` in `cmd/app/main.go` |
| No CLI version flag | Add `kenaz version` subcommand |
| No HTTP version endpoint | `GET /api/version` (unauthenticated) |
| No MCP version surface | `kenaz://server-info` resource (`application/json`) |
| No Docker image metadata | OCI labels in `build/Dockerfile` + `docker-push` build args |
| No release tagging | Semantic version tags + pinned image in `docker-compose.yml` |

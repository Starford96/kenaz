# 5. MCP Server Integration

**Goal**: Enable AI Agents (like Claude/Gemini in IDEs) to interact with the knowledge base.

## 5.1. Library & Transport
-   **Library**: `mark3labs/mcp-go`.
-   **Transport**: Stdio (Standard Input/Output) for local integration.

## 5.2. Tools
Expose internal Service methods as MCP Tools.

For canonical note content expectations, see:
- [`docs/note_format.md`](../note_format.md)

1.  **`search_notes`**
    -   Arg: `query` (string, required)
    -   Desc: "Full-text search through notes content and titles."
    -   Returns: List of matching paths + snippets (JSON, limit 20).

2.  **`read_note`**
    -   Arg: `path` (string, required)
    -   Desc: "Read the full content of a Markdown note."
    -   Returns: Raw file content.

3.  **`create_note`**
    -   Args: `path` (string, required), `content` (string, required)
    -   Desc: "Create a new Markdown note at the specified path."
    -   Content must follow the canonical note format (see `get_note_contract`).
    -   Language policy: file/directory names must be in English; values and body may use any language.

4.  **`update_note`**
    -   Args: `path` (string, required), `content` (string, required), `checksum` (string, optional)
    -   Desc: "Update an existing note. Optionally provide SHA-256 checksum for optimistic concurrency."
    -   Content must follow the canonical note format.

5.  **`delete_note`**
    -   Arg: `path` (string, required)
    -   Desc: "Delete an existing note at the specified path."

6.  **`list_notes`**
    -   Args: `folder` (optional string)
    -   Desc: "List all notes or notes in a specific folder."
    -   Returns: Newline-separated paths.

7.  **`get_backlinks`**
    -   Arg: `path` (string, required)
    -   Desc: "Find all notes that link to this one."
    -   Returns: Newline-separated source paths.

8.  **`get_note_contract`**
    -   Args: none
    -   Desc: "Returns the canonical Kenaz note format contract. Call before creating/updating notes."
    -   Returns: Contract text (Markdown).

9.  **`upload_asset`**
    -   Args: `url` (string, required), `filename` (string, optional)
    -   Desc: "Download a file from URL or base64 data URI and save as attachment."
    -   Stored in `attachments/` directory.
    -   Returns: `savedPath` and `markdownImage` ready to paste into a note.
    -   Supported formats: png, jpg, jpeg, gif, webp, svg, pdf. Max size: 10 MB.

## 5.3. Resources
-   **URI**: `kenaz://note-format`
-   **MIME**: `text/markdown`
-   Exposes the canonical note format contract as a readable resource for LLM consumers.

## 5.4. Testing Strategy

### Unit Tests
-   **Tools**:
    -   Verify inputs map correctly to service calls.
    -   Verify error handling (file not found returns appropriate MCP error).
    -   Contract tool returns expected content.
    -   Resource handler returns correct URI/MIME/text.

### Integration Tests
-   **Stdio**: Run the server process, send JSON-RPC request to stdin, verify response on stdout.
-   **MCP Inspector**: Use the MCP Inspector tool to interactively test all tools and resources during development.

## 5.5. Tool Payload Examples

### `create_note` (recommended content format)

```json
{
  "path": "projects/kenaz/fe3-polish.md",
  "content": "---\ntitle: \"Kenaz FE-3 Polish\"\ntags: [\"frontend\", \"ux\"]\nupdated_at: \"2026-02-16T10:15:00Z\"\n---\n\n# Kenaz FE-3 Polish\n\n## Summary\n\nPolish tasks for Obsidian-like UX.\n\n## Related\n\n- [[frontend-spec]]\n"
}
```

### `update_note` (with optimistic concurrency)

```json
{
  "path": "projects/kenaz/fe3-polish.md",
  "content": "---\ntitle: \"Kenaz FE-3 Polish (Updated)\"\ntags: [\"frontend\", \"ux\", \"done\"]\nupdated_at: \"2026-03-01T12:00:00Z\"\n---\n\n# Kenaz FE-3 Polish\n\nCompleted all tasks.\n",
  "checksum": "a1b2c3d4e5f6..."
}
```

### `read_note`

```json
{
  "path": "projects/kenaz/fe3-polish.md"
}
```

### `search_notes`

```json
{
  "query": "wikilink autocomplete"
}
```

### `list_notes`

```json
{
  "folder": "projects/kenaz"
}
```

### `get_backlinks`

```json
{
  "path": "frontend-spec.md"
}
```

### `upload_asset`

```json
{
  "url": "https://example.com/diagram.png",
  "filename": "architecture-diagram.png"
}
```

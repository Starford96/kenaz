# 5. MCP Server Integration

**Goal**: Enable AI Agents (like Claude/Gemini in IDEs) to interact with the knowledge base.

## 5.1. Library & Transport
-   **Library**: `mark3labs/mcp-go`.
-   **Transport**: Stdio (Standard Input/Output) for local integration.

## 5.2. Tools
Expose internal Service methods as MCP Tools.

1.  **`search_notes`**
    -   Arg: `query` (string)
    -   Desc: "Fuzzy search through notes content and titles."
    -   Returns: List of matching paths + snippets.

2.  **`read_note`**
    -   Arg: `path` (string)
    -   Desc: "Read the full content of a Markdown note."
    -   Returns: File content.

3.  **`create_note`**
    -   Args: `path` (string), `content` (string)
    -   Desc: "Create a new note at the specified path."

4.  **`list_notes`**
    -   Args: `folder` (optional string)
    -   Desc: "List all notes or notes in a specific folder."

5.  **`get_backlinks`**
    -   Arg: `path` (string)
    -   Desc: "Find all notes that link to this one."

## 5.3. Resources
-   **URI Scheme**: `note://internal/{path}`
-   Allows LLM to request a note as a resource directly if supported.

## 5.4. Testing Strategy

### Unit Tests
-   **Tools**:
    -   Verify inputs map correctly to service calls (e.g., `read_note` calls `storage.Read`).
    -   Verify error handling (e.g., file not found returns appropriate MCP error).

### Integration Tests
-   **Stdio**: Run the server process, send JSON-RPC request to stdin, verify response on stdout.
-   **MCP Inspector**: Use the MCP Inspector tool to interactively test all tools and resources during development.

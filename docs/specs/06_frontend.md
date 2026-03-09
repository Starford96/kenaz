# 6. Frontend Specification

**Goal**: Modern, responsive UI using React and Ant Design.

## 6.1. Tech Stack
-   **Build Tool**: Vite 7.
-   **Framework**: React 19.
-   **UI Library**: Ant Design v6.
    -   Theme: Obsidian-inspired dark theme (`darkAlgorithm` + `compactAlgorithm`).
    -   Custom color palette: teal accent (`#2dd4bf`), dark backgrounds.
    -   Fonts: DM Sans (UI), JetBrains Mono (code).
-   **State Management**:
    -   Server State: `@tanstack/react-query` v5 (caching, cache invalidation via SSE).
    -   Local State: `zustand` v5 (sidebar toggle, active tabs, panels) with `persist` middleware to `localStorage`.
-   **Routing**: `react-router-dom` v7 with bidirectional URL sync.
-   **Editor**: CodeMirror 6 (`@codemirror/lang-markdown`, `@codemirror/theme-one-dark`).
-   **Graph**: `react-force-graph-2d` (lazy-loaded chunk, ~190KB).
-   **Markdown Preview**: `react-markdown` + `remark-gfm` for GFM rendering.

## 6.2. Layout Architecture

Three-pane Obsidian-inspired layout with mobile-responsive design:

-   **Sidebar (Left)**:
    -   **File Tree**: Recursive component rendering folder structure (sorted: folders first, alphabetical).
    -   **Search Trigger**: Opens command palette.
    -   **Graph Toggle**: Switch to graph view (opens in dedicated tab).
    -   **Actions**: Create note, rename, delete, download folder as ZIP.
-   **Main Content (Center)**:
    -   **Tabbed Interface**: Editable tabs for multiple notes (state persisted to `localStorage`).
    -   **Editor/Viewer**:
        -   Mode: Read (rendered Markdown) vs Edit (CodeMirror 6).
        -   Toggle: toolbar button or `Cmd/Ctrl+E`.
        -   Save: `Cmd/Ctrl+S`.
        -   Rendered view handles `[[wikilinks]]` with click navigation (styled differently if target missing).
    -   **Graph Tab**: Virtual `__graph__` tab with force-directed graph view.
-   **Context Panel (Right - Optional)**:
    -   **Backlinks**: List of incoming links to current note.
    -   **Outline**: Table of Contents extracted from H1–H3 headings.

**Mobile Layout**:
-   Sidebar and context panel rendered as drawer overlays.
-   `MobileHeader` and `BottomActionBar` for navigation.
-   Breakpoints: `768px` (mobile), `1024px` (context panel).

## 6.3. Key Components
1.  **`MarkdownEditor`**:
    -   CodeMirror 6 with Markdown language support and oneDark theme.
    -   `[[` autocomplete: triggers search for existing notes via API.
    -   Attachment upload: paste/drag-and-drop inserts placeholder, replaces with `![name](url)` on success.
    -   Keyboard: `Mod+S` (save), `Mod+E` (toggle edit mode).
2.  **`GraphView`**:
    -   Library: `react-force-graph-2d`.
    -   Interactive: click node → open note in tab.
    -   Lazy-loaded via `React.lazy` (~190KB chunk).
3.  **`SearchModal`** (Command Palette, `Cmd/Ctrl+K`):
    -   Mixed items: FTS search results + static commands (New Note, Open Graph, Toggle Sidebar, Toggle Context Panel).
    -   Keyboard navigation (arrows + Enter).
    -   Empty query shows commands + recent notes.
4.  **`NoteView`**:
    -   Read/edit mode toggle.
    -   Wikilink rendering with click navigation.
    -   YAML frontmatter stripping in preview.
    -   Optimistic updates with checksum-based conflict detection.
    -   Download note as `.md`.
5.  **`Sidebar`**:
    -   Hierarchical file tree with expand/collapse.
    -   Inline rename/delete actions.
    -   Download folder as ZIP (`jszip`).
6.  **`CreateNoteModal`**:
    -   New note creation with auto `.md` suffix.
    -   Title derivation from path.

## 6.4. Data Integration
-   **API Client**: `openapi-fetch` — type-safe, spec-driven client.
    -   Types generated from OpenAPI spec via `openapi-typescript`.
    -   Base URL from `VITE_API_BASE` (default: `/api`).
    -   Auth header injection from `VITE_AUTH_TOKEN`.
-   **Real-time Listener** (`useSSE` hook):
    -   Connects to `/api/events` via `EventSource`.
    -   Auto-reconnects on drop.
    -   On `note.created/deleted`: invalidates `["notes"]` cache.
    -   On `note.updated`: invalidates `["note", path]` + `["notes"]` caches.
    -   On `graph.updated`: invalidates `["graph"]` cache.
-   **URL Sync** (`useUrlSync` hook):
    -   Active note path synced to browser URL pathname.
    -   Hash anchors for heading scroll.

## 6.5. Code Generation Pipeline
-   Backend: swaggo annotations on handlers → Swagger 2.0 → OpenAPI 3.1 (`make openapi`).
-   Frontend: `openapi-typescript` generates `src/api/schema.d.ts` from spec (`npm run generate`).
-   `make client-gen` chains the full pipeline.
-   Single source of truth: backend annotations → types flow to frontend.

## 6.6. Testing Strategy

### Unit Tests
-   **Store**: Zustand store tests (tab management, persistence, migration).
-   **Framework**: Vitest with jsdom environment.

### Dev Environment
-   Vite dev server on `:5173` with proxy to backend `:8080` (`/api`, `/attachments`).
-   `npm run dev` for hot reload, `npm run build` for production.

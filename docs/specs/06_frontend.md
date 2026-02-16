# 6. Frontend Specification

**Goal**: Modern, responsive UI using React and Ant Design.

## 6.1. Tech Stack
-   **Build Tool**: Vite.
-   **Framework**: React 18+.
-   **UI Library**: Ant Design (v5).
    -   Theme: Configurable (Dark/Light), Compact algorithm for info density.
-   **State Management**:
    -   Server State: `tanstack/react-query`.
    -   Local State: `zustand` (sidebar toggle, active tabs).
-   **Routing**: `react-router-dom`.

## 6.2. Layout Architecture
-   **Sidebar (Left)**:
    -   **File Tree**: Recursive component rendering folder structure.
    -   **Search Bar**: Quick access to trigger global search.
    -   **Graph Toggle**: Switch to graph view.
-   **Main Content (Center)**:
    -   **Tabbed Interface**: Allow multiple notes open (inspired by Obsidian/IDE).
    -   **Editor/Viewer**:
        -   Mode: Read (Rendered Markdown) vs Edit (CodeMirror/Monaco).
        -   Rendered view handles `[[wikilinks]]` by checking if target exists (styled differently if missing).
-   **Context Panel (Right - Optional)**:
    -   **Backlinks**: List of incoming links to current note.
    -   **Outline**: Table of Contents.

## 6.3. Key Components
1.  **`MarkdownEditor`**:
    -   Syntax highlighting for Markdown.
    -   Autocomplete for `[[` (triggers search for existing notes).
2.  **`GraphView`**:
    -   Library: `react-force-graph-2d`.
    -   Interactive: Click node -> Open note.
3.  **`GlobalSearchModal`** (`Cmd+K`):
    -   Modal with instant search results (from FTS5 API).
    -   Keyboard navigation.

## 6.4. Data Integration
-   **API Client**: `axios` instance.
    -   Base URL from config.
    -   Auth header injection.
-   **Real-time Listener**:
    -   Global `useEffect` connects to `/api/events`.
    -   On `note.updated`: Invalidate React Query cache for that note.
    -   On `note.created/deleted`: Invalidate File Tree query.

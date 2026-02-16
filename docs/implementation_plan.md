# Kenaz Implementation Plan

This document serves as the master index for the technical implementation of Kenaz. Each section links to a detailed specification file.

## 1. Core Backend
-   **[Storage & Parsing Strategy](specs/01_storage_parsing.md)**
    -   File System Adapter (Atomic writes)
    -   Frontmatter Parser
    -   Wikilink Extraction Rules
-   **[Indexing & Search Strategy](specs/02_indexing_search.md)**
    -   SQLite Schema (Notes, Links, FTS5)
    -   Indexer Service & File Watcher (fsnotify)

## 2. API & Communication
-   **[REST API Specification](specs/03_rest_api.md)**
    -   Router (`go-chi`)
    -   Endpoints (CRUD, Search, Backlinks)
    -   Authentication
-   **[Real-time Updates (SSE)](specs/04_realtime_updates.md)**
    -   Event Broadcasting Architecture
    -   Event Types (`note.updated`, etc.)

## 3. Integrations
-   **[MCP Server Integration](specs/05_mcp_server.md)**
    -   Tools for LLM usage
    -   Integration with stdio/SSE
-   **[Frontend Specification](specs/06_frontend.md)**
    -   React + Ant Design Architecture
    -   Component Hierarchy
    -   State Management & Real-time Sync

## Execution Order
1.  **Skeleton**: Go module, basic `main.go`, config.
2.  **Storage**: Working FS layer (Spec 01).
3.  **Index**: SQLite + FTS5 working (Spec 02).
4.  **API**: Read/Write endpoints + Search (Spec 03).
5.  **Frontend**: Basic read-only UI -> Editor -> Search (Spec 06).
6.  **Real-time + MCP**: Polish (Specs 04 & 05).

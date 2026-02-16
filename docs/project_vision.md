# Kenaz Project Vision

## Mission
To build a high-performance, local-first knowledge base that empowers users with full ownership of their data.

## Name & Symbol
**Kenaz** (ᚲ) is the rune of **knowledge, enlightenment, and creativity**. It represents the torch that illuminates the darkness, making it a fitting name for a tool designed to bring clarity to your thoughts and notes. The symbol `ᚲ` serves as the project's logo.

Kenaz combines the simplicity of Markdown files with the power of modern search and visualization, serving as a robust backend for personal knowledge management.

## Core Values
1.  **Data Ownership**: Plain `.md` files are the source of truth. No proprietary formats.
2.  **Performance**: Immediate response times for search and navigation.
3.  **Interoperability**: API-first design allowing integration with any frontend or automation tool (MCP).
4.  **Minimalism**: Do one thing well—manage knowledge connections.

## Key Goals
-   **Robust Storage**: Atomic file operations and reliable metadata parsing (Frontmatter).
-   **Intelligent Indexing**: Real-time SQLite indexing with FTS5 for instant search results, including multi-language support (Unicode61).
-   **Deep Connectivity**: First-class support for bidirectional linking (`[[wikilinks]]`) and graph visualization.
-   **Seamless Integration**:
    -   **REST API**: Comprehensive endpoints for external apps.
    -   **MCP Server**: Native integration with LLMs for AI-assisted note-taking.
    -   **Real-time Protocol**: Server-Sent Events (SSE) for instant UI updates across devices/tabs.
-   **Modern Frontend**: A responsive, clean React application using Ant Design for a premium user experience.

## Non-Goals (MVP)
-   Multi-user collaboration.
-   Cloud sync (relying on user's own file sync like Dropbox/iCloud/Syncthing).
-   Plugin system (core features first).

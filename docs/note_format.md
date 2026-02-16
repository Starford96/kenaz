# Note Format Contract (for Humans and Agents)

This document defines the expected Markdown/frontmatter format for notes created via UI, API, or MCP tools.

## Goals

- Keep notes interoperable and easy to parse.
- Ensure generated notes are consistent across agents and clients.
- Preserve compatibility with wikilinks and indexing.

## Canonical File Rules

1. Notes are UTF-8 Markdown files (`.md`).
2. Paths are vault-relative, for example: `projects/kenaz/roadmap.md`.
3. Use Unix-style path separators (`/`).
4. Do not use traversal segments (`..`) in note paths.

## Recommended Frontmatter Schema

Frontmatter is optional, but strongly recommended for agent-created notes.

```yaml
---
title: "Roadmap"
tags:
  - planning
  - kenaz
created_at: "2026-02-16T10:00:00Z"
updated_at: "2026-02-16T10:00:00Z"
aliases:
  - "Project Plan"
status: "draft"
---
```

### Field Guidance

- `title` (string): Human-readable note title.
- `tags` (array of strings): Lowercase preferred.
- `created_at` / `updated_at` (RFC3339 UTC): Optional but helpful for automation.
- `aliases` (array of strings): Optional alternate names.
- `status` (string): Optional workflow state (`draft`, `active`, `archived`).

Unknown fields are allowed and preserved.

## Body Conventions

- Use Markdown headings (`#`, `##`) for structure.
- Use wikilinks for internal references: `[[target-note]]`.
- Alias syntax is supported: `[[target-note|Readable Label]]`.
- Prefer short paragraphs and explicit section headings for agent-generated content.

## Minimal Agent Template

```markdown
---
title: "{{TITLE}}"
tags: ["{{TAG1}}", "{{TAG2}}"]
updated_at: "{{RFC3339_UTC}}"
---

# {{TITLE}}

## Summary

{{ONE_PARAGRAPH_SUMMARY}}

## Details

{{MAIN_CONTENT}}

## Related

- [[another-note]]
```

## MCP Tool Expectations

### `create_note`

- Inputs:
  - `path` (string, required)
  - `content` (string, required)
- Agent should pass valid Markdown content.
- If frontmatter is included, it should follow the schema guidance above.

### `read_note`

- Returns raw file content.
- Callers should be prepared for notes with or without frontmatter.

### `search_notes`

- Search is full-text and path-aware; high-quality titles/tags improve discoverability.

### `list_notes`

- Returns note metadata/listing; path naming consistency improves navigation.

### `get_backlinks`

- Backlinks are based on wikilink targets; consistent wikilink syntax is required.

## Example: Good Agent-Created Note

```markdown
---
title: "Kenaz Frontend FE-3 Polish"
tags: ["frontend", "ux", "kenaz"]
status: "active"
updated_at: "2026-02-16T10:15:00Z"
---

# Kenaz Frontend FE-3 Polish

## Summary

This note tracks UX polish tasks for the Obsidian-like interface.

## Tasks

- Persist tabs across reloads
- Improve command palette
- Refine markdown typography

## Related

- [[kenaz-progress]]
- [[frontend-spec]]
```

## Validation Checklist (for Agents)

Before creating a note, verify:

- Path ends with `.md`
- Path is vault-relative and safe
- Markdown is non-empty
- Frontmatter (if present) is valid YAML
- Wikilinks use `[[target]]` or `[[target|label]]`

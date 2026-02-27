package mcpserver

// NoteFormatContract describes the canonical Markdown note format that
// LLM consumers should follow when creating or updating notes.
const NoteFormatContract = `# Kenaz Note Format Contract

Every Markdown note stored in Kenaz MUST follow this structure.

## Structure

` + "```" + `markdown
---
title: Human-readable title        # REQUIRED – used in search, sidebar, graph
tags:                               # OPTIONAL – YAML list; used for filtering
  - tag-one
  - tag-two
created: 2025-01-15                 # OPTIONAL – ISO-8601 date or datetime
---

Body text in standard Markdown.

Use [[wikilinks]] to reference other notes (without .md extension).
Use [[target|alias]] for display text that differs from the target.
` + "```" + `

## Rules

1. **YAML frontmatter is mandatory.** The ` + "```" + `---` + "```" + ` fences must be the first
   thing in the file (no leading blank lines).
2. **` + "`" + `title` + "`" + ` field is required.** It is the primary display name everywhere.
3. **Tags** are lowercase, kebab-case (e.g. ` + "`" + `project-x` + "`" + `, ` + "`" + `meeting-notes` + "`" + `).
4. **Wikilinks** use double brackets: ` + "`" + `[[other-note]]` + "`" + `. The target is the
   filename stem (no ` + "`" + `.md` + "`" + ` extension, path separators OK: ` + "`" + `[[folder/note]]` + "`" + `).
5. **File paths** end with ` + "`" + `.md` + "`" + ` and use forward slashes.
6. **Encoding** is UTF-8 with a trailing newline.
7. **No HTML** unless absolutely necessary; prefer Markdown equivalents.
8. **Language policy:** file names and directory names MUST be in English (Latin characters).
   Frontmatter keys MUST be in English (they are schema fields). Frontmatter values
   (title, tags, aliases, etc.) and body content may use any language including Cyrillic.

## Assets & Images

- Upload assets via the ` + "`" + `upload_asset` + "`" + ` tool. It returns a ` + "`" + `markdownImage` + "`" + ` field ready to paste into the note body.
- Assets are stored in the shared ` + "`" + `attachments/` + "`" + ` directory (flat, no sub-folders).
- Reference in notes using the absolute path: ` + "`" + `![description](/attachments/filename.png)` + "`" + `
- Supported formats: png, jpg, jpeg, gif, webp, svg, pdf.
- Do **not** use relative paths like ` + "`" + `./attachments/...` + "`" + ` — always use ` + "`" + `/attachments/filename` + "`" + `.

## Example

` + "```" + `markdown
---
title: Weekly standup 2025-01-20
tags:
  - meeting-notes
  - project-x
created: 2025-01-20
---

# Weekly standup 2025-01-20

Attendees: Alice, Bob.

![Whiteboard photo](/attachments/standup-2025-01-20.jpg)

## Action items

- [[alice]] to review the [[design-doc]]
- Bob to update [[project-x/roadmap|the roadmap]]
` + "```" + `
`

# AGENTS.md — Kenaz Development Rules

This file defines execution rules for AI agents working on this repository.

## Core Workflow

1. **Plan first, then implement**
   - Produce a short implementation plan before coding.
   - Apply changes in small, reviewable batches.

2. **No commit/push without explicit approval**
   - Agents may edit files and run local checks.
   - `git commit` and especially `git push` require explicit user confirmation.

3. **Keep changes minimal and scoped**
   - Do not perform unrelated refactors.
   - Do not change files outside the requested scope unless necessary for build/test correctness.

## Code Quality Gates (before reporting completion)

Run and report:

- `go test ./...`
- project lint/format commands if available (`make lint`, `make fmt`)
- placeholder check when template-derived files are involved:
  - `grep -R "{{" --include="*.go" --include="*.yaml" --include="*.yml" .`

If any command cannot run, explain why and provide the exact blocker.

## Git Conventions

### Commit Messages

Format:

`<type>[scope]: <description>`

- `type` (required): one of
  - `breaking`, `build`, `ci`, `chore`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, `style`, `test`, `improvement`
- `scope` (optional): component or area affected
- `description` (required): concise summary in imperative mood

Examples:

- `feat(authorization): implement RBAC architecture`
- `fix(repository): correct pagination cursor handling`
- `refactor: simplify config loading`
- `docs: update API usage examples`

## Documentation Language Policy

All documentation in this repository must be written in **English only**.

This includes (non-exhaustive):

- `README.md`
- `docs/**/*.md`
- architecture notes
- API documentation
- agent-generated documentation artifacts

## OpenClaw Invocation Rule

When invoking `openclaw agent` from automation/scripts, always set channel explicitly to avoid incorrect telemetry attribution:

- Use: `--channel telegram`
- Never rely on the default channel value.

## Note Authoring Contract

When creating notes via API/MCP or generating sample note content, follow:

- `docs/note_format.md`

Prefer valid frontmatter and canonical wikilink syntax.

### Note Language Policy

- **File names and directory names** must be in English (Latin characters only, kebab-case preferred).
- **Frontmatter keys** must be in English (they are schema fields: `title`, `tags`, `status`, etc.).
- **Frontmatter values** (title, tags, aliases, etc.) may use any language, including Cyrillic.
- **Body content** (Markdown text) may be written in any language, including Cyrillic.

Agents must enforce these rules when creating or updating notes.

## Docker Registry

The project uses a local HTTP registry for Docker images:

- **Address**: `192.168.48.58:5005`
- **Push command**: `make docker-push` (builds and pushes `kenaz:latest`)
- **Custom version**: `make docker-push VERSION=1.2.3`
- The registry is configured as an insecure (HTTP) registry in Docker daemon config (`~/.docker/daemon.json`).

## Reporting Format

When finishing a batch, provide:

1. Files changed
2. What was implemented
3. Validation results (tests/lint/checks)
4. Open risks or follow-up items

Keep reports concise and actionable.

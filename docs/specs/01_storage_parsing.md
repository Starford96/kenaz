# 1. Storage & Parsing Strategy

**Goal**: Reliable file operations and metadata extraction.

## 1.1. Domain Models (`internal/models`)
-   **NoteMetadata**: Lightweight struct for listings.
    -   `Path`: Relative path (ID).
    -   `Checksum`: `string` (SHA256).
    -   `UpdatedAt`: `time.Time`.
-   **NoteDetail**: Full note representation (returned by API).
    -   `Path`, `Title`, `Content`, `Checksum`, `Tags`, `Frontmatter`, `Backlinks`, `UpdatedAt`.
-   **NoteListItem**: Compact listing item.
    -   `Path`, `Title`, `Checksum`, `Tags`, `UpdatedAt`.

## 1.2. File System Adapter (`internal/storage`)
-   **Interface**: `Provider`
    -   `List(dir string) ([]NoteMetadata, error)` — metadata for every `.md` file under dir.
    -   `Read(path string) ([]byte, error)` — raw bytes of a file.
    -   `Write(path string, content []byte) error` — atomic write.
    -   `Delete(path string) error` — remove a file.
    -   `DirExists(path string) (bool, error)` — check if directory exists.
    -   `DeleteDir(path string) error` — remove directory and all contents.
    -   `ListDirs() ([]string, error)` — all directory paths relative to vault root.
    -   `Move(oldPath, newPath string) error` — atomic rename.
-   **Implementation Details**:
    -   Use `os` and `io` packages.
    -   **Atomic Writes**:
        1.  Write content to a temporary file in the same directory.
        2.  `fsync` to ensure data is on disk.
        3.  `os.Rename` to overwrite the target file atomically.
    -   **Security**: Validate all paths to ensure they stay within the `vault` root (prevent directory traversal).
    -   **Configurable Exclusions**: Directories like `.git`, `attachments` can be excluded via config (`vault.ignore_dirs`).

## 1.3. Parser (`internal/parser`)
-   **Frontmatter**:
    -   Library: `gopkg.in/yaml.v3`.
    -   Logic: Extract content between the first pair of `---` delimiters.
    -   Fallback: If no delimiters, treat entire file as body.
-   **Wikilinks**:
    -   Regex: `\[\[(.*?)\]\]`
    -   Normalization: Handle pipes for aliases (e.g., `[[Link|Alias]]` -> Target: `Link`).
-   **Tags**:
    -   Regex: `(?:^|\s)#([A-Za-z][A-Za-z0-9_/-]*)` — extracted from body.
    -   Also extracted from frontmatter `tags` field.
    -   Deduplicated.
-   **Title Derivation**:
    -   From frontmatter `title` field, or first H1 heading, or filename.

## 1.4. Testing Strategy

### Unit Tests
-   **Parser**:
    -   Test frontmatter extraction with valid YAML, invalid YAML, and missing frontmatter.
    -   Test wikilink regex against various patterns (aliases, special chars).
    -   Test tag extraction from body and frontmatter.
    -   Test title derivation fallback chain.
-   **Storage**:
    -   Use temp dirs to test Read/Write/Delete/Move/DirExists/DeleteDir/ListDirs.
    -   Verify `Move` operations update paths correctly.
    -   Test directory traversal blocking (e.g., `../../etc/passwd`).

### Integration Tests
-   **Atomic Writes**: Simulate concurrent writes to the same file to ensure no data corruption (basic check).
-   **Cycle**: Write file -> Read back -> Verify content and checksum match.

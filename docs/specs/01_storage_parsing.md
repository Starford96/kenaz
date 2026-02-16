# 1. Storage & Parsing Strategy

**Goal**: Reliable file operations and metadata extraction.

## 1.1. Domain Models (`internal/models`)
-   **Note**: Struct representing a file.
    -   `Path`: Relative path (ID).
    -   `Content`: Raw bytes.
    -   `Frontmatter`: `Map[string]interface{}`.
    -   `Links`: `[]string` (targets).
    -   `Tags`: `[]string`.
    -   `CreatedAt`: `time.Time`.
    -   `UpdatedAt`: `time.Time`.
    -   `Checksum`: `string` (SHA256).

## 1.2. File System Adapter (`internal/storage`)
-   **Interface**: `Provider`
    -   `List(dir string) ([]NoteMetadata, error)`
    -   `Read(path string) (*Note, error)`
    -   `Write(path string, content []byte) error`
    -   `Delete(path string) error`
    -   `Move(oldPath, newPath string) error`
-   **Implementation Details**:
    -   Use `os` and `io` packages.
    -   **Atomic Writes**:
        1.  Write content to a temporary file in the same directory.
        2.  `fsync` to ensure data is on disk.
        3.  `os.Rename` to overwrite the target file atomically.
    -   **Security**: Validate all paths to ensure they stay within the `vault` root (prevent directory traversal).

## 1.3. Parser (`internal/parser`)
-   **Frontmatter**:
    -   Library: `gopkg.in/yaml.v3`.
    -   Logic: Extract content between the first pair of `---` delimiters.
    -   Fallback: If no delimiters, treat entire file as body.
-   **Wikilinks**:
    -   Regex: `\[\[(.*?)\]\]`
    -   Normalization: Handle pipes for aliases (e.g., `[[Link|Alias]]` -> Target: `Link`).

## 1.4. Testing Strategy

### Unit Tests
-   **Parser**:
    -   Test frontmatter extraction with valid YAML, invalid YAML, and missing frontmatter.
    -   Test wikilink regex against various patterns (aliases, special chars).
-   **Storage**:
    -   Mock the filesystem (using `afero` or temp dirs) to test Read/Write/Delete.
    -   Verify `Move` operations updates paths correctly.
    -   Test directory traversal blocking (e.g., `../../etc/passwd`).

### Integration Tests
-   **Atomic Writes**: Simulate concurrent writes to the same file to ensure no data corruption (basic check).
-   **Cycle**: Write file -> Read back -> Verify content and checksum match.

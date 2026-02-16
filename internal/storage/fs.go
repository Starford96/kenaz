package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/starford/kenaz/internal/models"
)

// FS implements Provider backed by the local file system.
type FS struct {
	root string // absolute path to vault directory
}

// NewFS creates a new FS provider rooted at the given directory.
// The directory must already exist.
func NewFS(root string) (*FS, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("storage: resolve root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("storage: stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("storage: root is not a directory: %s", abs)
	}
	return &FS{root: abs}, nil
}

// safePath resolves a relative path against the vault root and rejects
// any result that escapes it (directory traversal).
func (f *FS) safePath(rel string) (string, error) {
	if rel == "" {
		return f.root, nil
	}
	cleaned := filepath.Clean(rel)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("storage: absolute paths not allowed: %s", rel)
	}
	joined := filepath.Join(f.root, cleaned)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("storage: resolve path: %w", err)
	}
	// Ensure the resolved path is still under root.
	if !strings.HasPrefix(abs, f.root+string(os.PathSeparator)) && abs != f.root {
		return "", fmt.Errorf("storage: path escapes vault root: %s", rel)
	}
	return abs, nil
}

// List walks dir (relative to root) and returns metadata for every .md file.
func (f *FS) List(dir string) ([]models.NoteMetadata, error) {
	base, err := f.safePath(dir)
	if err != nil {
		return nil, err
	}
	var out []models.NoteMetadata
	err = filepath.WalkDir(base, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(f.root, p)
		out = append(out, models.NoteMetadata{
			Path:      rel,
			Checksum:  checksum(data),
			UpdatedAt: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: list: %w", err)
	}
	return out, nil
}

// Read returns the raw bytes of a vault file.
func (f *FS) Read(path string) ([]byte, error) {
	abs, err := f.safePath(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("storage: read %s: %w", path, err)
	}
	return data, nil
}

// Write atomically writes content: tmp file → fsync → rename.
func (f *FS) Write(path string, content []byte) error {
	abs, err := f.safePath(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".kenaz-tmp-*")
	if err != nil {
		return fmt.Errorf("storage: create temp: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up on any failure path.
	success := false
	defer func() {
		if !success {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return fmt.Errorf("storage: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("storage: fsync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("storage: close temp: %w", err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return fmt.Errorf("storage: rename: %w", err)
	}
	success = true
	return nil
}

// Delete removes a file from the vault.
func (f *FS) Delete(path string) error {
	abs, err := f.safePath(path)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil {
		return fmt.Errorf("storage: delete %s: %w", path, err)
	}
	return nil
}

// Move renames a file within the vault.
func (f *FS) Move(oldPath, newPath string) error {
	absOld, err := f.safePath(oldPath)
	if err != nil {
		return err
	}
	absNew, err := f.safePath(newPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(absNew)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("storage: mkdir for move: %w", err)
	}
	if err := os.Rename(absOld, absNew); err != nil {
		return fmt.Errorf("storage: move: %w", err)
	}
	return nil
}

func checksum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

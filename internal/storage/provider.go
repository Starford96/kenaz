// Package storage defines the vault file-system abstraction.
package storage

import "github.com/starford/kenaz/internal/models"

// Provider is the interface for vault file operations.
type Provider interface {
	// List returns metadata for every .md file under dir (relative to vault root).
	List(dir string) ([]models.NoteMetadata, error)
	// Read returns the raw bytes of the file at path (relative to vault root).
	Read(path string) ([]byte, error)
	// Write atomically writes content to path (relative to vault root).
	Write(path string, content []byte) error
	// Delete removes the file at path (relative to vault root).
	Delete(path string) error
	// Move renames oldPath to newPath (both relative to vault root).
	Move(oldPath, newPath string) error
}

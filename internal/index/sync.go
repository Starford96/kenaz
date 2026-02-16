package index

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"

	"github.com/starford/kenaz/internal/parser"
	"github.com/starford/kenaz/internal/storage"
)

// Sync walks the vault and brings the index up to date:
//   - new/changed files are parsed and upserted
//   - files removed from disk are deleted from the index
func Sync(db *DB, store storage.Provider, logger *slog.Logger) error {
	metas, err := store.List("")
	if err != nil {
		return err
	}

	indexed, err := db.AllPaths()
	if err != nil {
		return err
	}

	disk := make(map[string]struct{}, len(metas))
	for _, m := range metas {
		disk[m.Path] = struct{}{}

		existing, err := db.GetChecksum(m.Path)
		if err != nil {
			return err
		}
		if existing == m.Checksum {
			continue // unchanged
		}

		data, err := store.Read(m.Path)
		if err != nil {
			logger.Warn("sync: read failed", slog.String("path", m.Path), slog.String("error", err.Error()))
			continue
		}
		if err := indexFile(db, m.Path, data); err != nil {
			logger.Warn("sync: index failed", slog.String("path", m.Path), slog.String("error", err.Error()))
		} else {
			logger.Debug("sync: indexed", slog.String("path", m.Path))
		}
	}

	// Remove stale entries.
	for p := range indexed {
		if _, ok := disk[p]; !ok {
			if err := db.DeleteNote(p); err != nil {
				logger.Warn("sync: delete failed", slog.String("path", p), slog.String("error", err.Error()))
			} else {
				logger.Debug("sync: removed stale", slog.String("path", p))
			}
		}
	}

	return nil
}

// indexFile parses data and upserts it into the DB.
func indexFile(db *DB, path string, data []byte) error {
	res, err := parser.Parse(data)
	if err != nil {
		return err
	}
	h := sha256.Sum256(data)
	cs := hex.EncodeToString(h[:])

	row := NoteRow{
		Path:     path,
		Title:    res.Title,
		Checksum: cs,
		Tags:     res.Tags,
	}
	return db.UpsertNote(row, res.Body, res.Links)
}

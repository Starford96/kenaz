package index

import (
	"log/slog"

	"github.com/starford/kenaz/internal/checksum"
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

	checksums, err := db.AllChecksums()
	if err != nil {
		return err
	}

	disk := make(map[string]struct{}, len(metas))
	for _, m := range metas {
		disk[m.Path] = struct{}{}

		if checksums[m.Path] == m.Checksum {
			continue
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
	for p := range checksums {
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
	cs := checksum.Sum(data)

	row := NoteRow{
		Path:     path,
		Title:    res.Title,
		Checksum: cs,
		Tags:     res.Tags,
	}
	return db.UpsertNote(row, res.Body, res.Links)
}

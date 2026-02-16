package index

import (
	"context"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/starford/kenaz/internal/storage"
)

// EventCallback is called after a watcher-driven index change.
// kind is one of "created", "updated", "deleted".
type EventCallback func(kind string, path string)

// Watch starts an fsnotify watcher on the vault root and processes file
// change events until ctx is cancelled. It calls cb (if non-nil) after
// each successful index mutation.
func Watch(ctx context.Context, db *DB, store storage.Provider, vaultRoot string, logger *slog.Logger, cb EventCallback) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	// Watch the vault root recursively is not supported by fsnotify out of
	// the box; we add the root and rely on non-recursive watching for MVP.
	// A future improvement can walk subdirs and add each.
	if err := addDirsRecursive(w, vaultRoot); err != nil {
		return err
	}

	logger.Info("watcher: started", slog.String("root", vaultRoot))

	for {
		select {
		case <-ctx.Done():
			logger.Info("watcher: stopped")
			return nil

		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			rel, err := filepath.Rel(vaultRoot, ev.Name)
			if err != nil || !strings.HasSuffix(ev.Name, ".md") {
				continue
			}

			switch {
			case ev.Op&(fsnotify.Create|fsnotify.Write) != 0:
				data, readErr := store.Read(rel)
				if readErr != nil {
					logger.Warn("watcher: read failed", slog.String("path", rel), slog.String("error", readErr.Error()))
					continue
				}
				if idxErr := indexFile(db, rel, data); idxErr != nil {
					logger.Warn("watcher: index failed", slog.String("path", rel), slog.String("error", idxErr.Error()))
					continue
				}
				kind := "updated"
				if ev.Op&fsnotify.Create != 0 {
					kind = "created"
				}
				logger.Debug("watcher: indexed", slog.String("path", rel), slog.String("op", kind))
				if cb != nil {
					cb(kind, rel)
				}

			case ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0:
				if delErr := db.DeleteNote(rel); delErr != nil {
					logger.Warn("watcher: delete failed", slog.String("path", rel), slog.String("error", delErr.Error()))
					continue
				}
				logger.Debug("watcher: deleted", slog.String("path", rel))
				if cb != nil {
					cb("deleted", rel)
				}
			}

		case watchErr, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logger.Error("watcher: error", slog.String("error", watchErr.Error()))
		}
	}
}

// addDirsRecursive adds vaultRoot and all its subdirectories to the watcher.
func addDirsRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(path)
		}
		return nil
	})
}

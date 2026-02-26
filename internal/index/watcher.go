package index

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/starford/kenaz/internal/storage"
)

// EventCallback is called after a watcher-driven index change.
// kind is one of "created", "updated", "deleted".
type EventCallback func(kind string, path string)

// Watch starts an fsnotify watcher on the vault root and processes file
// change events until ctx is cancelled. It calls cb (if non-nil) after
// each successful index mutation.
//
// New directories created at runtime are automatically added to the watch
// list. Rename events trigger a reconciliation pass that removes stale
// index entries whose files no longer exist on disk.
func Watch(ctx context.Context, db *DB, store storage.Provider, vaultRoot string, logger *slog.Logger, cb EventCallback) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := addDirsRecursive(w, vaultRoot); err != nil {
		return err
	}

	logger.Info("watcher: started", slog.String("root", vaultRoot))

	// reconcileTimer is used to debounce rename reconciliation.
	var reconcileTimer *time.Timer
	var reconcileCh <-chan time.Time

	scheduleReconcile := func() {
		if reconcileTimer == nil {
			reconcileTimer = time.NewTimer(200 * time.Millisecond)
			reconcileCh = reconcileTimer.C
		} else {
			reconcileTimer.Reset(200 * time.Millisecond)
		}
	}

	for {
		select {
		case <-ctx.Done():
			if reconcileTimer != nil {
				reconcileTimer.Stop()
			}
			logger.Info("watcher: stopped")
			return nil

		case <-reconcileCh:
			reconcileAfterRename(db, store, logger, cb)

		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}

			absPath := ev.Name

			// --- Handle new directories: add to watcher ---
			if ev.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(absPath); statErr == nil && info.IsDir() {
					if addErr := addDirsRecursive(w, absPath); addErr != nil {
						logger.Warn("watcher: add new dir failed",
							slog.String("path", absPath),
							slog.String("error", addErr.Error()))
					} else {
						logger.Debug("watcher: watching new dir", slog.String("path", absPath))
					}
					// Index any .md files already in the new directory.
					indexNewDir(db, store, vaultRoot, absPath, logger, cb)
					continue
				}
			}

			// Only process .md files from here on.
			if !strings.HasSuffix(absPath, ".md") {
				continue
			}

			rel, relErr := filepath.Rel(vaultRoot, absPath)
			if relErr != nil {
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

			case ev.Op&fsnotify.Remove != 0:
				if delErr := db.DeleteNote(rel); delErr != nil {
					logger.Warn("watcher: delete failed", slog.String("path", rel), slog.String("error", delErr.Error()))
					continue
				}
				logger.Debug("watcher: deleted", slog.String("path", rel))
				if cb != nil {
					cb("deleted", rel)
				}

			case ev.Op&fsnotify.Rename != 0:
				// fsnotify fires Rename on the OLD path only. The new
				// path will arrive as a separate Create event (if it
				// stays within a watched dir). We delete the old entry
				// immediately and schedule a short reconciliation pass
				// to catch any stragglers.
				if delErr := db.DeleteNote(rel); delErr != nil {
					logger.Warn("watcher: rename delete failed", slog.String("path", rel), slog.String("error", delErr.Error()))
				} else {
					logger.Debug("watcher: rename old deleted", slog.String("path", rel))
					if cb != nil {
						cb("deleted", rel)
					}
				}
				scheduleReconcile()
			}

		case watchErr, ok := <-w.Errors:
			if !ok {
				return nil
			}
			logger.Error("watcher: error", slog.String("error", watchErr.Error()))
		}
	}
}

// reconcileAfterRename does a lightweight sync using batch lookups:
// finds index entries without a corresponding file on disk and removes them,
// and finds on-disk files that are not indexed and indexes them.
func reconcileAfterRename(db *DB, store storage.Provider, logger *slog.Logger, cb EventCallback) {
	checksums, err := db.AllChecksums()
	if err != nil {
		logger.Warn("reconcile: all checksums failed", slog.String("error", err.Error()))
		return
	}

	metas, err := store.List("")
	if err != nil {
		logger.Warn("reconcile: list failed", slog.String("error", err.Error()))
		return
	}

	disk := make(map[string]string, len(metas))
	for _, m := range metas {
		disk[m.Path] = m.Checksum
	}

	for p := range checksums {
		if _, ok := disk[p]; !ok {
			if delErr := db.DeleteNote(p); delErr == nil {
				logger.Debug("reconcile: removed stale", slog.String("path", p))
				if cb != nil {
					cb("deleted", p)
				}
			}
		}
	}

	for p, cs := range disk {
		if checksums[p] == cs {
			continue
		}
		data, readErr := store.Read(p)
		if readErr != nil {
			continue
		}
		if idxErr := indexFile(db, p, data); idxErr == nil {
			logger.Debug("reconcile: indexed new", slog.String("path", p))
			if cb != nil {
				cb("created", p)
			}
		}
	}
}

// indexNewDir indexes any .md files found in a newly created directory.
func indexNewDir(db *DB, store storage.Provider, vaultRoot, dirPath string, logger *slog.Logger, cb EventCallback) {
	_ = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(vaultRoot, path)
		if relErr != nil {
			return nil
		}
		data, readErr := store.Read(rel)
		if readErr != nil {
			return nil
		}
		if idxErr := indexFile(db, rel, data); idxErr == nil {
			logger.Debug("watcher: indexed from new dir", slog.String("path", rel))
			if cb != nil {
				cb("created", rel)
			}
		}
		return nil
	})
}

// addDirsRecursive adds root and all its subdirectories to the watcher.
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

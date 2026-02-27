package index

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/starford/kenaz/internal/storage"
)

// watcherTestEnv sets up a vault dir, storage, and DB for watcher tests.
func watcherTestEnv(t *testing.T) (string, storage.Provider, *DB) {
	t.Helper()
	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	dbFile, err := os.CreateTemp("", "kenaz-watcher-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbFile.Name()) })
	db, err := Open(dbFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return vaultDir, store, db
}

// eventually polls fn every tick until it returns true or timeout elapses.
func eventually(t *testing.T, timeout, tick time.Duration, fn func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(tick)
	}
	t.Error(msg)
}

func TestWatcher_NewFileIndexed(t *testing.T) {
	vaultDir, store, db := watcherTestEnv(t)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var events []string

	go Watch(ctx, db, store, vaultDir, logger, func(kind, path string) {
		mu.Lock()
		events = append(events, kind+":"+path)
		mu.Unlock()
	})

	time.Sleep(100 * time.Millisecond)

	_ = os.WriteFile(filepath.Join(vaultDir, "new.md"), []byte("# New"), 0o644)

	eventually(t, 5*time.Second, 50*time.Millisecond, func() bool {
		cs, _ := db.GetChecksum("new.md")
		return cs != ""
	}, "new file not indexed by watcher")

	eventually(t, 2*time.Second, 50*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, e := range events {
			if e == "created:new.md" {
				return true
			}
		}
		return false
	}, "expected created:new.md callback")
}

func TestWatcher_NewDirWatched(t *testing.T) {
	vaultDir, store, db := watcherTestEnv(t)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Watch(ctx, db, store, vaultDir, logger, nil)

	time.Sleep(100 * time.Millisecond)

	subDir := filepath.Join(vaultDir, "subdir")
	_ = os.MkdirAll(subDir, 0o755)

	eventually(t, 2*time.Second, 50*time.Millisecond, func() bool {
		return true
	}, "")

	_ = os.WriteFile(filepath.Join(subDir, "deep.md"), []byte("# Deep"), 0o644)

	eventually(t, 5*time.Second, 50*time.Millisecond, func() bool {
		cs, _ := db.GetChecksum(filepath.Join("subdir", "deep.md"))
		return cs != ""
	}, "file in new subdir not indexed by watcher")
}

func TestWatcher_DeleteRemovesFromIndex(t *testing.T) {
	vaultDir, store, db := watcherTestEnv(t)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	_ = os.WriteFile(filepath.Join(vaultDir, "del.md"), []byte("# Delete Me"), 0o644)
	Sync(db, store, logger)

	cs, _ := db.GetChecksum("del.md")
	if cs == "" {
		t.Fatal("precondition: file should be indexed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Watch(ctx, db, store, vaultDir, logger, nil)
	time.Sleep(100 * time.Millisecond)

	_ = os.Remove(filepath.Join(vaultDir, "del.md"))

	eventually(t, 5*time.Second, 50*time.Millisecond, func() bool {
		cs, _ := db.GetChecksum("del.md")
		return cs == ""
	}, "deleted file still in index")
}

func TestWatcher_RenameReconciles(t *testing.T) {
	vaultDir, store, db := watcherTestEnv(t)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	_ = os.WriteFile(filepath.Join(vaultDir, "old.md"), []byte("# Rename"), 0o644)
	Sync(db, store, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go Watch(ctx, db, store, vaultDir, logger, nil)
	time.Sleep(100 * time.Millisecond)

	_ = os.Rename(filepath.Join(vaultDir, "old.md"), filepath.Join(vaultDir, "renamed.md"))

	eventually(t, 5*time.Second, 50*time.Millisecond, func() bool {
		oldCS, _ := db.GetChecksum("old.md")
		newCS, _ := db.GetChecksum("renamed.md")
		return oldCS == "" && newCS != ""
	}, "rename reconciliation failed: old path should be removed and new path indexed")
}

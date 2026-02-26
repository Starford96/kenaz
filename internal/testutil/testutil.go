// Package testutil provides shared test helpers for setting up vaults and databases.
package testutil

import (
	"os"
	"testing"

	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/storage"
)

// TestDB creates a temporary SQLite database that is automatically cleaned up.
func TestDB(t *testing.T) *index.DB {
	t.Helper()
	dbFile, err := os.CreateTemp("", "kenaz-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbFile.Name()) })

	db, err := index.Open(dbFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestVault creates a temporary vault directory with a storage.Provider.
func TestVault(t *testing.T) (string, storage.Provider) {
	t.Helper()
	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	return vaultDir, store
}

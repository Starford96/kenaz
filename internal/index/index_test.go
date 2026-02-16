package index

import (
	"os"
	"testing"
	"time"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	f, err := os.CreateTemp("", "kenaz-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	db, err := Open(f.Name())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSchemaCreation(t *testing.T) {
	db := testDB(t)
	// Verify tables exist by querying them.
	var count int
	if err := db.conn.QueryRow(`SELECT count(*) FROM notes`).Scan(&count); err != nil {
		t.Fatalf("notes table missing: %v", err)
	}
	if err := db.conn.QueryRow(`SELECT count(*) FROM links`).Scan(&count); err != nil {
		t.Fatalf("links table missing: %v", err)
	}
	// FTS5 virtual table.
	if err := db.conn.QueryRow(`SELECT count(*) FROM files_fts`).Scan(&count); err != nil {
		t.Fatalf("files_fts table missing: %v", err)
	}
}

func TestUpsertAndSearch(t *testing.T) {
	db := testDB(t)

	row := NoteRow{
		Path:      "hello.md",
		Title:     "Hello World",
		Checksum:  "abc123",
		Tags:      []string{"go", "test"},
		UpdatedAt: time.Now(),
	}
	if err := db.UpsertNote(row, "This is a hello world note about Kenaz.", []string{"other.md"}); err != nil {
		t.Fatalf("UpsertNote: %v", err)
	}

	// Search by content.
	results, err := db.Search("hello", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "hello.md" {
		t.Errorf("path = %q", results[0].Path)
	}

	// Search by title.
	results, err = db.Search("World", 10)
	if err != nil {
		t.Fatalf("Search title: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for title search, got %d", len(results))
	}
}

func TestBacklinks(t *testing.T) {
	db := testDB(t)

	// Note A links to Note B.
	_ = db.UpsertNote(NoteRow{Path: "a.md", Checksum: "1", Tags: []string{}, UpdatedAt: time.Now()}, "body", []string{"b.md"})
	// Note C also links to Note B.
	_ = db.UpsertNote(NoteRow{Path: "c.md", Checksum: "2", Tags: []string{}, UpdatedAt: time.Now()}, "body", []string{"b.md"})

	bl, err := db.Backlinks("b.md")
	if err != nil {
		t.Fatalf("Backlinks: %v", err)
	}
	if len(bl) != 2 {
		t.Fatalf("expected 2 backlinks, got %d", len(bl))
	}
}

func TestDeleteNote(t *testing.T) {
	db := testDB(t)
	_ = db.UpsertNote(NoteRow{Path: "del.md", Checksum: "x", Tags: []string{}, UpdatedAt: time.Now()}, "body", []string{"target.md"})

	if err := db.DeleteNote("del.md"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	// Should not appear in search.
	results, _ := db.Search("body", 10)
	for _, r := range results {
		if r.Path == "del.md" {
			t.Error("deleted note still in FTS")
		}
	}

	// Links should be gone.
	bl, _ := db.Backlinks("target.md")
	if len(bl) != 0 {
		t.Errorf("expected 0 backlinks after delete, got %d", len(bl))
	}
}

func TestUpsertUpdatesExisting(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	_ = db.UpsertNote(NoteRow{Path: "up.md", Title: "Old", Checksum: "1", Tags: []string{}, UpdatedAt: now}, "old body", []string{"x.md"})
	_ = db.UpsertNote(NoteRow{Path: "up.md", Title: "New", Checksum: "2", Tags: []string{"new"}, UpdatedAt: now}, "new body", []string{"y.md"})

	cs, _ := db.GetChecksum("up.md")
	if cs != "2" {
		t.Errorf("checksum = %q, want %q", cs, "2")
	}

	// Old link should be replaced.
	bl, _ := db.Backlinks("x.md")
	if len(bl) != 0 {
		t.Error("old link should be removed on upsert")
	}
	bl, _ = db.Backlinks("y.md")
	if len(bl) != 1 {
		t.Error("new link should exist")
	}

	// FTS should have new content.
	results, _ := db.Search("new", 10)
	if len(results) != 1 || results[0].Title != "New" {
		t.Errorf("FTS not updated: %+v", results)
	}
}

func TestGetChecksum_NotFound(t *testing.T) {
	db := testDB(t)
	cs, err := db.GetChecksum("nonexistent.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs != "" {
		t.Errorf("expected empty checksum, got %q", cs)
	}
}

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
	var count int
	if err := db.conn.QueryRow(`SELECT count(*) FROM notes`).Scan(&count); err != nil {
		t.Fatalf("notes table missing: %v", err)
	}
	if err := db.conn.QueryRow(`SELECT count(*) FROM links`).Scan(&count); err != nil {
		t.Fatalf("links table missing: %v", err)
	}
}

func TestUpsertAndGetChecksum(t *testing.T) {
	db := testDB(t)
	row := NoteRow{
		Path:      "hello.md",
		Title:     "Hello World",
		Checksum:  "abc123",
		Tags:      []string{"go", "test"},
		UpdatedAt: time.Now(),
	}
	if err := db.UpsertNote(row, "This is a hello world note.", []string{"other.md"}); err != nil {
		t.Fatalf("UpsertNote: %v", err)
	}
	cs, err := db.GetChecksum("hello.md")
	if err != nil {
		t.Fatalf("GetChecksum: %v", err)
	}
	if cs != "abc123" {
		t.Errorf("checksum = %q, want %q", cs, "abc123")
	}
}

func TestBacklinks(t *testing.T) {
	db := testDB(t)
	_ = db.UpsertNote(NoteRow{Path: "a.md", Checksum: "1", Tags: []string{}, UpdatedAt: time.Now()}, "body", []string{"b.md"})
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
	cs, _ := db.GetChecksum("del.md")
	if cs != "" {
		t.Errorf("deleted note still has checksum %q", cs)
	}
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
	bl, _ := db.Backlinks("x.md")
	if len(bl) != 0 {
		t.Error("old link should be removed on upsert")
	}
	bl, _ = db.Backlinks("y.md")
	if len(bl) != 1 {
		t.Error("new link should exist")
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

func TestSearch_Basic(t *testing.T) {
	db := testDB(t)
	_ = db.UpsertNote(NoteRow{Path: "s.md", Title: "Search Me", Checksum: "1", Tags: []string{}, UpdatedAt: time.Now()}, "uniqueword appears here", nil)

	results, err := db.Search("uniqueword", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].Path != "s.md" {
		t.Errorf("search results = %+v, want 1 hit for s.md", results)
	}
}

func TestMoveNote(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	// Create "old.md" that links to "target.md".
	_ = db.UpsertNote(NoteRow{Path: "old.md", Title: "Old", Checksum: "cs1", Tags: []string{"t1"}, UpdatedAt: now}, "old body mentioning target", []string{"target.md"})
	// Create "ref.md" that links to "old.md".
	_ = db.UpsertNote(NoteRow{Path: "ref.md", Title: "Ref", Checksum: "cs2", Tags: []string{}, UpdatedAt: now}, "see old note", []string{"old.md"})

	if err := db.MoveNote("old.md", "new.md"); err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	// Old path should be gone.
	cs, _ := db.GetChecksum("old.md")
	if cs != "" {
		t.Errorf("old path still has checksum %q", cs)
	}
	// New path should have original checksum.
	cs, _ = db.GetChecksum("new.md")
	if cs != "cs1" {
		t.Errorf("new path checksum = %q, want %q", cs, "cs1")
	}
	// Source link should be updated: new.md → target.md.
	bl, _ := db.Backlinks("target.md")
	if len(bl) != 1 || bl[0] != "new.md" {
		t.Errorf("backlinks for target.md = %v, want [new.md]", bl)
	}
	// Target link should be updated: ref.md → new.md.
	bl, _ = db.Backlinks("new.md")
	if len(bl) != 1 || bl[0] != "ref.md" {
		t.Errorf("backlinks for new.md = %v, want [ref.md]", bl)
	}
	// FTS should find the note at new path.
	results, _ := db.Search("old body", 10)
	if len(results) != 1 || results[0].Path != "new.md" {
		t.Errorf("FTS search after move = %+v, want new.md", results)
	}
}

func TestMoveNote_NotFound(t *testing.T) {
	db := testDB(t)
	err := db.MoveNote("nonexistent.md", "new.md")
	if err == nil {
		t.Fatal("expected error for moving nonexistent note")
	}
}

func TestMoveNote_LinksTargetNoExt(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	// "ref.md" links to "old" (without .md).
	_ = db.UpsertNote(NoteRow{Path: "ref.md", Checksum: "1", Tags: []string{}, UpdatedAt: now}, "body", []string{"old"})
	_ = db.UpsertNote(NoteRow{Path: "old.md", Checksum: "2", Tags: []string{}, UpdatedAt: now}, "body", nil)

	if err := db.MoveNote("old.md", "new.md"); err != nil {
		t.Fatalf("MoveNote: %v", err)
	}

	// Link target "old" should be updated to "new".
	bl, _ := db.Backlinks("new")
	if len(bl) != 1 || bl[0] != "ref.md" {
		t.Errorf("backlinks for no-ext target = %v, want [ref.md]", bl)
	}
}

func TestMoveNotesBatch(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	_ = db.UpsertNote(NoteRow{Path: "dir/a.md", Title: "A", Checksum: "a1", Tags: []string{}, UpdatedAt: now}, "body a", []string{"dir/b.md"})
	_ = db.UpsertNote(NoteRow{Path: "dir/b.md", Title: "B", Checksum: "b1", Tags: []string{}, UpdatedAt: now}, "body b", []string{"dir/a.md"})
	_ = db.UpsertNote(NoteRow{Path: "outside.md", Title: "Out", Checksum: "o1", Tags: []string{}, UpdatedAt: now}, "body out", []string{"dir/a.md"})

	moves := []PathMove{
		{OldPath: "dir/a.md", NewPath: "newdir/a.md"},
		{OldPath: "dir/b.md", NewPath: "newdir/b.md"},
	}
	if err := db.MoveNotesBatch(moves); err != nil {
		t.Fatalf("MoveNotesBatch: %v", err)
	}

	// Old paths gone.
	cs, _ := db.GetChecksum("dir/a.md")
	if cs != "" {
		t.Errorf("old dir/a.md still exists")
	}
	// New paths present.
	cs, _ = db.GetChecksum("newdir/a.md")
	if cs != "a1" {
		t.Errorf("newdir/a.md checksum = %q, want %q", cs, "a1")
	}
	cs, _ = db.GetChecksum("newdir/b.md")
	if cs != "b1" {
		t.Errorf("newdir/b.md checksum = %q, want %q", cs, "b1")
	}
	// Inter-note links updated.
	bl, _ := db.Backlinks("newdir/b.md")
	found := false
	for _, b := range bl {
		if b == "newdir/a.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("backlinks for newdir/b.md = %v, want newdir/a.md", bl)
	}
	// External backlink updated.
	bl, _ = db.Backlinks("newdir/a.md")
	found = false
	for _, b := range bl {
		if b == "outside.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("backlinks for newdir/a.md = %v, want outside.md", bl)
	}
}

func TestMoveNotesBatch_Empty(t *testing.T) {
	db := testDB(t)
	if err := db.MoveNotesBatch(nil); err != nil {
		t.Fatalf("MoveNotesBatch(nil): %v", err)
	}
}

func TestNotesWithPrefix(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	_ = db.UpsertNote(NoteRow{Path: "dir/a.md", Checksum: "1", Tags: []string{}, UpdatedAt: now}, "body", nil)
	_ = db.UpsertNote(NoteRow{Path: "dir/b.md", Checksum: "2", Tags: []string{}, UpdatedAt: now}, "body", nil)
	_ = db.UpsertNote(NoteRow{Path: "other/c.md", Checksum: "3", Tags: []string{}, UpdatedAt: now}, "body", nil)

	notes, err := db.NotesWithPrefix("dir/")
	if err != nil {
		t.Fatalf("NotesWithPrefix: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	paths := map[string]bool{}
	for _, n := range notes {
		paths[n.Path] = true
	}
	if !paths["dir/a.md"] || !paths["dir/b.md"] {
		t.Errorf("unexpected paths: %v", paths)
	}
}

func TestNotesWithPrefix_NoMatch(t *testing.T) {
	db := testDB(t)
	_ = db.UpsertNote(NoteRow{Path: "a.md", Checksum: "1", Tags: []string{}, UpdatedAt: time.Now()}, "body", nil)

	notes, err := db.NotesWithPrefix("nonexistent/")
	if err != nil {
		t.Fatalf("NotesWithPrefix: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

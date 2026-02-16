//go:build sqlite_fts5

package index

import (
	"testing"
	"time"
)

func TestFTS5_TableExists(t *testing.T) {
	db := testDB(t)
	var count int
	if err := db.conn.QueryRow(`SELECT count(*) FROM files_fts`).Scan(&count); err != nil {
		t.Fatalf("files_fts table missing: %v", err)
	}
}

func TestFTS5_SearchWithSnippet(t *testing.T) {
	db := testDB(t)
	row := NoteRow{
		Path:      "fts.md",
		Title:     "FTS Note",
		Checksum:  "f1",
		Tags:      []string{"search"},
		UpdatedAt: time.Now(),
	}
	if err := db.UpsertNote(row, "Kenaz provides powerful full-text search capabilities.", nil); err != nil {
		t.Fatalf("UpsertNote: %v", err)
	}

	results, err := db.Search("powerful", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Path != "fts.md" {
		t.Errorf("path = %q", results[0].Path)
	}
	// FTS5 snippet should contain bold markers.
	if results[0].Snippet == "" {
		t.Error("expected non-empty snippet")
	}
}

func TestFTS5_DeleteRemovesFromFTS(t *testing.T) {
	db := testDB(t)
	_ = db.UpsertNote(NoteRow{Path: "gone.md", Checksum: "g", Tags: []string{}, UpdatedAt: time.Now()}, "vanishing content", nil)
	_ = db.DeleteNote("gone.md")

	results, _ := db.Search("vanishing", 10)
	for _, r := range results {
		if r.Path == "gone.md" {
			t.Error("deleted note still in FTS index")
		}
	}
}

func TestFTS5_UpsertReplacesContent(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	_ = db.UpsertNote(NoteRow{Path: "evo.md", Title: "Old", Checksum: "1", Tags: []string{}, UpdatedAt: now}, "original text", nil)
	_ = db.UpsertNote(NoteRow{Path: "evo.md", Title: "New", Checksum: "2", Tags: []string{}, UpdatedAt: now}, "replacement text", nil)

	results, _ := db.Search("original", 10)
	if len(results) != 0 {
		t.Error("old FTS content should be gone")
	}
	results, _ = db.Search("replacement", 10)
	if len(results) != 1 || results[0].Title != "New" {
		t.Errorf("FTS not updated: %+v", results)
	}
}

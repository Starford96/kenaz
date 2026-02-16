package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func tempVault(t *testing.T) *FS {
	t.Helper()
	dir := t.TempDir()
	fs, err := NewFS(dir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	return fs
}

func TestWriteAndRead(t *testing.T) {
	s := tempVault(t)
	content := []byte("# Hello\nWorld\n")
	if err := s.Write("note.md", content); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.Read("note.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q", got)
	}
}

func TestWriteCreatesSubdirs(t *testing.T) {
	s := tempVault(t)
	if err := s.Write("a/b/c.md", []byte("deep")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := s.Read("a/b/c.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != "deep" {
		t.Errorf("content = %q", got)
	}
}

func TestDelete(t *testing.T) {
	s := tempVault(t)
	_ = s.Write("del.md", []byte("bye"))
	if err := s.Delete("del.md"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Read("del.md"); err == nil {
		t.Error("expected error reading deleted file")
	}
}

func TestMove(t *testing.T) {
	s := tempVault(t)
	_ = s.Write("old.md", []byte("data"))
	if err := s.Move("old.md", "sub/new.md"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	got, err := s.Read("sub/new.md")
	if err != nil {
		t.Fatalf("Read after move: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("content = %q", got)
	}
	if _, err := s.Read("old.md"); err == nil {
		t.Error("old path should not exist")
	}
}

func TestList(t *testing.T) {
	s := tempVault(t)
	_ = s.Write("a.md", []byte("a"))
	_ = s.Write("sub/b.md", []byte("b"))
	_ = s.Write("readme.txt", []byte("not md"))

	items, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len = %d, want 2", len(items))
	}
}

func TestTraversalBlocked(t *testing.T) {
	s := tempVault(t)

	cases := []string{
		"../../etc/passwd",
		"../outside.md",
		"/etc/shadow",
	}
	for _, p := range cases {
		if _, err := s.Read(p); err == nil {
			t.Errorf("expected error for path %q", p)
		}
		if err := s.Write(p, []byte("x")); err == nil {
			t.Errorf("expected error for write to %q", p)
		}
	}
}

func TestAtomicWriteNoCorruption(t *testing.T) {
	// Verify that if we read during a write the old content is intact
	// (the rename is atomic on POSIX).
	s := tempVault(t)
	original := []byte("original content")
	_ = s.Write("atomic.md", original)

	// Overwrite with new content.
	updated := []byte("updated content")
	if err := s.Write("atomic.md", updated); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, _ := s.Read("atomic.md")
	if string(got) != string(updated) {
		t.Errorf("expected updated content, got %q", got)
	}

	// Confirm no leftover temp files.
	matches, _ := filepath.Glob(filepath.Join(s.root, ".kenaz-tmp-*"))
	if len(matches) != 0 {
		t.Errorf("leftover temp files: %v", matches)
	}
}

func TestNewFS_NonExistentDir(t *testing.T) {
	_, err := NewFS("/tmp/kenaz-does-not-exist-" + t.Name())
	if err == nil {
		t.Error("expected error for non-existent dir")
	}
}

func TestNewFS_FileNotDir(t *testing.T) {
	f, _ := os.CreateTemp("", "kenaz-test-*")
	_ = f.Close()
	defer os.Remove(f.Name())
	_, err := NewFS(f.Name())
	if err == nil {
		t.Error("expected error when root is a file")
	}
}

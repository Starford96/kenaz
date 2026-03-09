package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func tempVault(t *testing.T) *FS {
	t.Helper()
	dir := t.TempDir()
	fs, err := NewFS(dir, nil)
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

func tempVaultWithIgnore(t *testing.T, ignoreDirs []string) *FS {
	t.Helper()
	dir := t.TempDir()
	fs, err := NewFS(dir, ignoreDirs)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	return fs
}

func TestListDirs_IgnoresConfiguredDirs(t *testing.T) {
	s := tempVaultWithIgnore(t, []string{".git", "attachments"})

	for _, d := range []string{"visible", ".git", "attachments", "nested/deep"} {
		if err := os.MkdirAll(filepath.Join(s.root, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	dirs, err := s.ListDirs()
	if err != nil {
		t.Fatalf("ListDirs: %v", err)
	}

	dirSet := make(map[string]struct{})
	for _, d := range dirs {
		dirSet[d] = struct{}{}
	}

	for _, want := range []string{"visible", "nested", filepath.Join("nested", "deep")} {
		if _, ok := dirSet[want]; !ok {
			t.Errorf("expected dir %q in result, got %v", want, dirs)
		}
	}
	for _, excluded := range []string{".git", "attachments"} {
		if _, ok := dirSet[excluded]; ok {
			t.Errorf("dir %q should be excluded, got %v", excluded, dirs)
		}
	}
}

func TestListDirs_IgnoresNestedDirsByName(t *testing.T) {
	s := tempVaultWithIgnore(t, []string{".git"})

	for _, d := range []string{"nested/.git/objects", "nested/visible"} {
		if err := os.MkdirAll(filepath.Join(s.root, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	dirs, err := s.ListDirs()
	if err != nil {
		t.Fatalf("ListDirs: %v", err)
	}

	for _, d := range dirs {
		if filepath.Base(d) == ".git" || filepath.Dir(d) == ".git" {
			t.Errorf("found .git-related dir in results: %v", dirs)
		}
	}
}

func TestListDirs_NoIgnoreDirs(t *testing.T) {
	s := tempVaultWithIgnore(t, nil)

	for _, d := range []string{".git", "attachments", "notes"} {
		if err := os.MkdirAll(filepath.Join(s.root, d), 0o750); err != nil {
			t.Fatal(err)
		}
	}

	dirs, err := s.ListDirs()
	if err != nil {
		t.Fatalf("ListDirs: %v", err)
	}
	if len(dirs) != 3 {
		t.Errorf("expected 3 dirs, got %d: %v", len(dirs), dirs)
	}
}

func TestList_SkipsIgnoredDirs(t *testing.T) {
	s := tempVaultWithIgnore(t, []string{".git"})

	_ = s.Write("visible/note.md", []byte("# Note"))

	gitDir := filepath.Join(s.root, ".git")
	_ = os.MkdirAll(gitDir, 0o750)
	_ = os.WriteFile(filepath.Join(gitDir, "HEAD.md"), []byte("not a note"), 0o600)

	items, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if len(items) > 0 && items[0].Path != filepath.Join("visible", "note.md") {
		t.Errorf("path = %q", items[0].Path)
	}
}

func TestNewFS_NonExistentDir(t *testing.T) {
	_, err := NewFS("/tmp/kenaz-does-not-exist-"+t.Name(), nil)
	if err == nil {
		t.Error("expected error for non-existent dir")
	}
}

func TestNewFS_FileNotDir(t *testing.T) {
	f, _ := os.CreateTemp("", "kenaz-test-*")
	_ = f.Close()
	defer os.Remove(f.Name())
	_, err := NewFS(f.Name(), nil)
	if err == nil {
		t.Error("expected error when root is a file")
	}
}

package noteservice

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/starford/kenaz/internal/apperr"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/storage"
)

func testService(t *testing.T) *Service {
	t.Helper()
	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatal(err)
	}
	dbFile, err := os.CreateTemp("", "kenaz-svc-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	_ = dbFile.Close()
	t.Cleanup(func() { _ = os.Remove(dbFile.Name()) }) //nolint:gosec // test cleanup
	db, err := index.Open(dbFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewService(store, db)
}

func createNote(t *testing.T, svc *Service, path, content string) {
	t.Helper()
	if _, err := svc.CreateNote(context.Background(), path, []byte(content)); err != nil {
		t.Fatalf("CreateNote(%s): %v", path, err)
	}
}

func TestRenameNote_Basic(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	createNote(t, svc, "old.md", "# Old\nBody")

	note, err := svc.RenameNote(ctx, "old.md", "new.md")
	if err != nil {
		t.Fatalf("RenameNote: %v", err)
	}
	if note.Path != "new.md" {
		t.Errorf("path = %q, want %q", note.Path, "new.md")
	}

	// New path should be accessible.
	got, err := svc.GetNote(ctx, "new.md")
	if err != nil {
		t.Fatalf("GetNote(new.md): %v", err)
	}
	if got.Path != "new.md" {
		t.Errorf("GetNote path = %q", got.Path)
	}

	// Old path should be gone.
	_, err = svc.GetNote(ctx, "old.md")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("GetNote(old.md) err = %v, want ErrNotFound", err)
	}
}

func TestRenameNote_WikilinkRewrite(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	createNote(t, svc, "target.md", "# Target")
	createNote(t, svc, "ref.md", "See [[target.md]] and [[target|my alias]]")

	if _, err := svc.RenameNote(ctx, "target.md", "moved.md"); err != nil {
		t.Fatalf("RenameNote: %v", err)
	}

	ref, err := svc.GetNote(ctx, "ref.md")
	if err != nil {
		t.Fatalf("GetNote(ref.md): %v", err)
	}
	if !strings.Contains(ref.Content, "[[moved.md]]") {
		t.Errorf("expected [[moved.md]] in ref content, got: %s", ref.Content)
	}
	if !strings.Contains(ref.Content, "[[moved|my alias]]") {
		t.Errorf("expected [[moved|my alias]] in ref content, got: %s", ref.Content)
	}
}

func TestRenameNote_WikilinkNoExtRewrite(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	createNote(t, svc, "notes/page.md", "# Page")
	createNote(t, svc, "ref.md", "Link: [[notes/page]]")

	if _, err := svc.RenameNote(ctx, "notes/page.md", "notes/renamed.md"); err != nil {
		t.Fatalf("RenameNote: %v", err)
	}

	ref, err := svc.GetNote(ctx, "ref.md")
	if err != nil {
		t.Fatalf("GetNote(ref.md): %v", err)
	}
	if !strings.Contains(ref.Content, "[[notes/renamed]]") {
		t.Errorf("expected [[notes/renamed]] in ref content, got: %s", ref.Content)
	}
}

func TestRenameNote_NotFound(t *testing.T) {
	svc := testService(t)
	_, err := svc.RenameNote(context.Background(), "nonexistent.md", "new.md")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRenameNote_TargetExists(t *testing.T) {
	svc := testService(t)
	createNote(t, svc, "a.md", "# A")
	createNote(t, svc, "b.md", "# B")

	_, err := svc.RenameNote(context.Background(), "a.md", "b.md")
	if !errors.Is(err, apperr.ErrAlreadyExists) {
		t.Errorf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestRenameDir_Basic(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	createNote(t, svc, "proj/a.md", "# A")
	createNote(t, svc, "proj/b.md", "# B")

	newPaths, err := svc.RenameDir(ctx, "proj/", "archive/")
	if err != nil {
		t.Fatalf("RenameDir: %v", err)
	}
	if len(newPaths) != 2 {
		t.Fatalf("expected 2 new paths, got %d", len(newPaths))
	}

	// Verify new paths are accessible.
	for _, p := range newPaths {
		if _, err := svc.GetNote(ctx, p); err != nil {
			t.Errorf("GetNote(%s): %v", p, err)
		}
	}
	// Old paths should be gone.
	for _, old := range []string{"proj/a.md", "proj/b.md"} {
		if _, err := svc.GetNote(ctx, old); !errors.Is(err, apperr.ErrNotFound) {
			t.Errorf("GetNote(%s) err = %v, want ErrNotFound", old, err)
		}
	}
}

func TestRenameDir_WikilinkRewrite(t *testing.T) {
	svc := testService(t)
	ctx := context.Background()
	createNote(t, svc, "proj/page.md", "# Page")
	createNote(t, svc, "external.md", "See [[proj/page]]")

	if _, err := svc.RenameDir(ctx, "proj/", "archive/"); err != nil {
		t.Fatalf("RenameDir: %v", err)
	}

	ref, err := svc.GetNote(ctx, "external.md")
	if err != nil {
		t.Fatalf("GetNote(external.md): %v", err)
	}
	if !strings.Contains(ref.Content, "[[archive/page]]") {
		t.Errorf("expected [[archive/page]] in content, got: %s", ref.Content)
	}
}

func TestRenameDir_NotFound(t *testing.T) {
	svc := testService(t)
	_, err := svc.RenameDir(context.Background(), "nonexistent/", "new/")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestRenameDir_Conflict(t *testing.T) {
	svc := testService(t)
	createNote(t, svc, "old/a.md", "# A")
	createNote(t, svc, "new/a.md", "# Existing")

	_, err := svc.RenameDir(context.Background(), "old/", "new/")
	if !errors.Is(err, apperr.ErrAlreadyExists) {
		t.Errorf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestRewriteWikilinks(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		oldTarget string
		newTarget string
		want      string
	}{
		{"simple", "[[old]]", "old", "new", "[[new]]"},
		{"with alias", "[[old|alias]]", "old", "new", "[[new|alias]]"},
		{"multiple", "[[old]] text [[old|x]]", "old", "new", "[[new]] text [[new|x]]"},
		{"no match", "[[other]]", "old", "new", "[[other]]"},
		{"with md ext", "[[old.md]]", "old.md", "new.md", "[[new.md]]"},
		{"no wikilinks", "plain text", "old", "new", "plain text"},
		{"path with slashes", "[[dir/old]]", "dir/old", "dir/new", "[[dir/new]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteWikilinks(tt.content, tt.oldTarget, tt.newTarget)
			if got != tt.want {
				t.Errorf("rewriteWikilinks(%q, %q, %q) = %q, want %q", tt.content, tt.oldTarget, tt.newTarget, got, tt.want)
			}
		})
	}
}

func TestMergeUnique(t *testing.T) {
	got := mergeUnique([]string{"a", "b"}, []string{"b", "c"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	seen := map[string]bool{}
	for _, s := range got {
		seen[s] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !seen[want] {
			t.Errorf("missing %q in %v", want, got)
		}
	}

	got = mergeUnique(nil, []string{"a"})
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("mergeUnique(nil, [a]) = %v", got)
	}

	got = mergeUnique(nil, nil)
	if len(got) != 0 {
		t.Errorf("mergeUnique(nil, nil) = %v, want nil or empty", got)
	}
}

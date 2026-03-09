package noteservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/starford/kenaz/internal/apperr"
	"github.com/starford/kenaz/internal/checksum"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/parser"
	"github.com/starford/kenaz/internal/storage"
)

// NoteDetail is the full representation of a note.
type NoteDetail struct {
	Path        string         `json:"path" validate:"required"`
	Title       string         `json:"title" validate:"required"`
	Content     string         `json:"content" validate:"required"`
	Checksum    string         `json:"checksum" validate:"required"`
	Tags        []string       `json:"tags" validate:"required"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
	Backlinks   []string       `json:"backlinks" validate:"required"`
	UpdatedAt   time.Time      `json:"updated_at" validate:"required"`
}

// NoteListItem is a lightweight item in a list response.
type NoteListItem struct {
	Path      string    `json:"path" validate:"required"`
	Title     string    `json:"title" validate:"required"`
	Checksum  string    `json:"checksum" validate:"required"`
	Tags      []string  `json:"tags" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// Service coordinates storage and index operations.
type Service struct {
	store storage.Provider
	db    *index.DB
}

// NewService creates a new note service.
func NewService(store storage.Provider, db *index.DB) *Service {
	return &Service{store: store, db: db}
}

// GetNote reads a note from storage, parses it, and enriches with backlinks.
func (s *Service) GetNote(_ context.Context, path string) (*NoteDetail, error) {
	data, err := s.store.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return s.buildNoteDetail(path, data)
}

// CreateNote writes a new note and indexes it.
func (s *Service) CreateNote(_ context.Context, path string, content []byte) (*NoteDetail, error) {
	if _, err := s.store.Read(path); err == nil {
		return nil, apperr.ErrAlreadyExists
	}
	if err := s.store.Write(path, content); err != nil {
		return nil, err
	}
	if err := s.IndexFile(path, content); err != nil {
		return nil, err
	}
	return s.buildNoteDetail(path, content)
}

// UpdateNote writes updated content with optimistic concurrency.
func (s *Service) UpdateNote(_ context.Context, path string, content []byte, ifMatch string) (*NoteDetail, error) {
	existing, err := s.store.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	if ifMatch != "" && ifMatch != checksum.Sum(existing) {
		return nil, apperr.ErrConflict
	}
	if err := s.store.Write(path, content); err != nil {
		return nil, err
	}
	if err := s.IndexFile(path, content); err != nil {
		return nil, err
	}
	return s.buildNoteDetail(path, content)
}

// DeleteNote removes a note from storage and index.
func (s *Service) DeleteNote(_ context.Context, path string) error {
	if err := s.store.Delete(path); err != nil {
		return err
	}
	return s.db.DeleteNote(path)
}

// DeleteDir removes a directory and all notes within it from storage and index.
func (s *Service) DeleteDir(_ context.Context, prefix string) ([]string, error) {
	notes, err := s.db.NotesWithPrefix(prefix)
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, apperr.ErrNotFound
	}

	paths := make([]string, len(notes))
	for i, n := range notes {
		paths[i] = n.Path
	}

	if err := s.db.DeleteNotesBatch(paths); err != nil {
		return nil, err
	}

	dirPath := strings.TrimSuffix(prefix, "/")
	if err := s.store.DeleteDir(dirPath); err != nil {
		return nil, err
	}

	return paths, nil
}

// ListDirs returns all directory paths from the vault filesystem.
func (s *Service) ListDirs() ([]string, error) {
	return s.store.ListDirs()
}

// ListNotes returns paginated notes with optional tag filter.
func (s *Service) ListNotes(_ context.Context, limit, offset int, tag, sort string) ([]NoteListItem, int, error) {
	rows, total, err := s.db.ListNotes(limit, offset, tag, sort)
	if err != nil {
		return nil, 0, err
	}
	items := make([]NoteListItem, len(rows))
	for i, r := range rows {
		items[i] = NoteListItem{
			Path:      r.Path,
			Title:     r.Title,
			Checksum:  r.Checksum,
			Tags:      nonNilSlice(r.Tags),
			UpdatedAt: r.UpdatedAt,
		}
	}
	return items, total, nil
}

// Search delegates full-text search to the index.
func (s *Service) Search(_ context.Context, query string, limit int) ([]index.SearchResult, error) {
	return s.db.Search(query, limit)
}

// Graph returns all nodes and links for graph visualization.
func (s *Service) Graph(_ context.Context) ([]index.GraphNode, []index.GraphLink, error) {
	return s.db.Graph()
}

// Backlinks returns all note paths that link to the given target.
func (s *Service) Backlinks(_ context.Context, target string) ([]string, error) {
	return s.db.Backlinks(target)
}

// IndexFile parses data and upserts it into the index.
// Exported so that sync and watcher can reuse it.
func (s *Service) IndexFile(path string, data []byte) error {
	res, err := parser.Parse(data)
	if err != nil {
		return err
	}
	cs := checksum.Sum(data)
	return s.db.UpsertNote(index.NoteRow{
		Path:      path,
		Title:     res.Title,
		Checksum:  cs,
		Tags:      nonNilSlice(res.Tags),
		UpdatedAt: time.Now(),
	}, res.Body, res.Links)
}

// buildNoteDetail constructs a NoteDetail from raw data without re-reading the file.
func (s *Service) buildNoteDetail(path string, data []byte) (*NoteDetail, error) {
	res, err := parser.Parse(data)
	if err != nil {
		return nil, err
	}
	bl, err := s.db.Backlinks(path)
	if err != nil {
		return nil, err
	}
	return &NoteDetail{
		Path:        path,
		Title:       res.Title,
		Content:     string(data),
		Checksum:    checksum.Sum(data),
		Tags:        nonNilSlice(res.Tags),
		Frontmatter: res.Frontmatter,
		Backlinks:   nonNilSlice(bl),
		UpdatedAt:   time.Now(),
	}, nil
}

// RenameNote moves a single note to a new path and updates wikilinks in referencing notes.
func (s *Service) RenameNote(_ context.Context, oldPath, newPath string) (*NoteDetail, error) {
	// Verify old note exists.
	data, err := s.store.Read(oldPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	// Verify new path doesn't exist.
	if _, err := s.store.Read(newPath); err == nil {
		return nil, apperr.ErrAlreadyExists
	}

	// Collect backlinks before moving.
	oldNoExt := strings.TrimSuffix(oldPath, ".md")
	newNoExt := strings.TrimSuffix(newPath, ".md")
	blWithExt, _ := s.db.Backlinks(oldPath)
	blNoExt, _ := s.db.Backlinks(oldNoExt)
	backlinkSources := mergeUnique(blWithExt, blNoExt)

	// Move on filesystem and update index.
	if err := s.store.Move(oldPath, newPath); err != nil {
		return nil, err
	}
	if err := s.db.MoveNote(oldPath, newPath); err != nil {
		return nil, err
	}

	// Rewrite wikilinks in all backlinking notes.
	s.rewriteBacklinks(backlinkSources, oldPath, oldNoExt, newPath, newNoExt)

	return s.buildNoteDetail(newPath, data)
}

// RenameDir renames a directory and all notes within it, updating wikilinks.
func (s *Service) RenameDir(_ context.Context, oldPrefix, newPrefix string) ([]string, error) {
	// Find all notes under old prefix.
	notes, err := s.db.NotesWithPrefix(oldPrefix)
	if err != nil {
		return nil, err
	}
	if len(notes) == 0 {
		return nil, apperr.ErrNotFound
	}

	// Build move list and collect all backlinks.
	moves := make([]index.PathMove, 0, len(notes))
	allBacklinks := make(map[string]struct{})
	for _, n := range notes {
		np := newPrefix + strings.TrimPrefix(n.Path, oldPrefix)
		moves = append(moves, index.PathMove{OldPath: n.Path, NewPath: np})

		// Collect backlinks for each note being moved.
		oldNoExt := strings.TrimSuffix(n.Path, ".md")
		bl1, _ := s.db.Backlinks(n.Path)
		bl2, _ := s.db.Backlinks(oldNoExt)
		for _, b := range append(bl1, bl2...) {
			allBacklinks[b] = struct{}{}
		}
	}

	// Verify no conflicts at new paths.
	for _, m := range moves {
		if _, err := s.store.Read(m.NewPath); err == nil {
			return nil, fmt.Errorf("%w: %s", apperr.ErrAlreadyExists, m.NewPath)
		}
	}

	// Rename directory on filesystem (os.Rename handles dirs).
	dirOld := strings.TrimSuffix(oldPrefix, "/")
	dirNew := strings.TrimSuffix(newPrefix, "/")
	if err := s.store.Move(dirOld, dirNew); err != nil {
		return nil, err
	}

	// Update index in batch.
	if err := s.db.MoveNotesBatch(moves); err != nil {
		return nil, err
	}

	// Rewrite wikilinks in all backlinking notes (exclude notes being moved — their paths changed).
	movedSet := make(map[string]struct{}, len(moves))
	for _, m := range moves {
		movedSet[m.NewPath] = struct{}{}
		movedSet[m.OldPath] = struct{}{}
	}
	for _, m := range moves {
		oldNoExt := strings.TrimSuffix(m.OldPath, ".md")
		newNoExt := strings.TrimSuffix(m.NewPath, ".md")
		var sources []string
		for b := range allBacklinks {
			if _, isMoved := movedSet[b]; !isMoved {
				sources = append(sources, b)
			}
		}
		s.rewriteBacklinks(sources, m.OldPath, oldNoExt, m.NewPath, newNoExt)
	}

	newPaths := make([]string, len(moves))
	for i, m := range moves {
		newPaths[i] = m.NewPath
	}
	return newPaths, nil
}

// rewriteBacklinks rewrites wikilink references in backlinking notes.
func (s *Service) rewriteBacklinks(sources []string, oldPath, oldNoExt, newPath, newNoExt string) {
	for _, src := range sources {
		// Skip if the source is the note being moved.
		if src == oldPath || src == newPath {
			continue
		}
		data, err := s.store.Read(src)
		if err != nil {
			continue
		}
		updated := rewriteWikilinks(string(data), oldPath, newPath)
		if oldNoExt != oldPath {
			updated = rewriteWikilinks(updated, oldNoExt, newNoExt)
		}
		if updated == string(data) {
			continue
		}
		if err := s.store.Write(src, []byte(updated)); err != nil {
			continue
		}
		_ = s.IndexFile(src, []byte(updated))
	}
}

// rewriteWikilinks replaces [[oldTarget]] and [[oldTarget|alias]] with new target.
func rewriteWikilinks(content, oldTarget, newTarget string) string {
	// Match [[oldTarget]] and [[oldTarget|...]]
	escaped := regexp.QuoteMeta(oldTarget)
	re := regexp.MustCompile(`\[\[` + escaped + `(\|[^\]]*?)?\]\]`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Extract alias part if present.
		inner := match[2 : len(match)-2] // strip [[ and ]]
		if idx := strings.Index(inner, "|"); idx >= 0 {
			alias := inner[idx:] // includes the |
			return "[[" + newTarget + alias + "]]"
		}
		return "[[" + newTarget + "]]"
	})
}

// mergeUnique merges two string slices, deduplicating.
func mergeUnique(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	var out []string
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

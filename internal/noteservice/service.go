package noteservice

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/starford/kenaz/internal/apperr"
	"github.com/starford/kenaz/internal/checksum"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/parser"
	"github.com/starford/kenaz/internal/storage"
)

// NoteDetail is the full representation of a note.
type NoteDetail struct {
	Path        string         `json:"path"`
	Title       string         `json:"title"`
	Content     string         `json:"content"`
	Checksum    string         `json:"checksum"`
	Tags        []string       `json:"tags"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
	Backlinks   []string       `json:"backlinks"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// NoteListItem is a lightweight item in a list response.
type NoteListItem struct {
	Path      string    `json:"path"`
	Title     string    `json:"title"`
	Checksum  string    `json:"checksum"`
	Tags      []string  `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
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

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

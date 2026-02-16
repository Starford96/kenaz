package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/parser"
	"github.com/starford/kenaz/internal/storage"
)

// Service coordinates storage and index operations for the API layer.
type Service struct {
	store storage.Provider
	db    *index.DB
}

// NewService creates a new API service.
func NewService(store storage.Provider, db *index.DB) *Service {
	return &Service{store: store, db: db}
}

// NoteDetail is the response payload for a single note.
type NoteDetail struct {
	Path        string                 `json:"path"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Checksum    string                 `json:"checksum"`
	Tags        []string               `json:"tags"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
	Backlinks   []string               `json:"backlinks"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NoteListItem is a lightweight item in a list response.
type NoteListItem struct {
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Checksum  string   `json:"checksum"`
	Tags      []string `json:"tags"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetNote reads a note from storage, parses it, and enriches with backlinks.
func (s *Service) GetNote(path string) (*NoteDetail, error) {
	data, err := s.store.Read(path)
	if err != nil {
		return nil, err
	}
	res, err := parser.Parse(data)
	if err != nil {
		return nil, err
	}
	cs := sha256sum(data)
	bl, _ := s.db.Backlinks(path)
	if bl == nil {
		bl = []string{}
	}
	tags := res.Tags
	if tags == nil {
		tags = []string{}
	}
	return &NoteDetail{
		Path:        path,
		Title:       res.Title,
		Content:     string(data),
		Checksum:    cs,
		Tags:        tags,
		Frontmatter: res.Frontmatter,
		Backlinks:   bl,
		UpdatedAt:   time.Now(),
	}, nil
}

// CreateNote writes a new note and indexes it. Returns error if it already exists.
func (s *Service) CreateNote(path string, content []byte) (*NoteDetail, error) {
	// Check existence.
	if _, err := s.store.Read(path); err == nil {
		return nil, fmt.Errorf("already exists")
	}
	if err := s.store.Write(path, content); err != nil {
		return nil, err
	}
	if err := s.indexFile(path, content); err != nil {
		return nil, err
	}
	return s.GetNote(path)
}

// UpdateNote writes updated content with optimistic concurrency (checksum match).
func (s *Service) UpdateNote(path string, content []byte, ifMatch string) (*NoteDetail, error) {
	existing, err := s.store.Read(path)
	if err != nil {
		return nil, fmt.Errorf("not found")
	}
	currentCS := sha256sum(existing)
	if ifMatch != "" && ifMatch != currentCS {
		return nil, fmt.Errorf("conflict")
	}
	if err := s.store.Write(path, content); err != nil {
		return nil, err
	}
	if err := s.indexFile(path, content); err != nil {
		return nil, err
	}
	return s.GetNote(path)
}

// DeleteNote removes a note from storage and index.
func (s *Service) DeleteNote(path string) error {
	if err := s.store.Delete(path); err != nil {
		return err
	}
	return s.db.DeleteNote(path)
}

// ListNotes returns paginated notes with optional tag filter.
func (s *Service) ListNotes(limit, offset int, tag, sort string) ([]NoteListItem, int, error) {
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
			Tags:      r.Tags,
			UpdatedAt: r.UpdatedAt,
		}
	}
	return items, total, nil
}

// Search delegates to the index.
func (s *Service) Search(query string, limit int) ([]index.SearchResult, error) {
	return s.db.Search(query, limit)
}

// Graph delegates to the index.
func (s *Service) Graph() ([]index.GraphNode, []index.GraphLink, error) {
	return s.db.Graph()
}

func (s *Service) indexFile(path string, data []byte) error {
	res, err := parser.Parse(data)
	if err != nil {
		return err
	}
	cs := sha256sum(data)
	tags := res.Tags
	if tags == nil {
		tags = []string{}
	}
	return s.db.UpsertNote(index.NoteRow{
		Path:      path,
		Title:     res.Title,
		Checksum:  cs,
		Tags:      tags,
		UpdatedAt: time.Now(),
	}, res.Body, res.Links)
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

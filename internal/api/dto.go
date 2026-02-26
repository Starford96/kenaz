package api

import (
	"time"

	"github.com/starford/kenaz/internal/noteservice"
)

// CreateNoteRequest is the request body for creating a note.
type CreateNoteRequest struct {
	Path    string `json:"path" example:"notes/hello.md" validate:"required"`
	Content string `json:"content" example:"# Hello\nWorld" validate:"required"`
}

// UpdateNoteRequest is the request body for updating a note.
type UpdateNoteRequest struct {
	Content string `json:"content" example:"# Updated\nContent" validate:"required"`
}

// NoteDetail is the full note response type (aliased from the domain layer).
type NoteDetail = noteservice.NoteDetail

// NoteListItem is a lightweight item in a list response (aliased from the domain layer).
type NoteListItem = noteservice.NoteListItem

// NoteListResponse wraps paginated note listings.
type NoteListResponse struct {
	Notes []NoteListItem `json:"notes" validate:"required"`
	Total int            `json:"total" example:"42" validate:"required"`
}

// SearchResult is a single search hit in the API response.
type SearchResult struct {
	Path    string `json:"path" example:"notes/hello.md" validate:"required"`
	Title   string `json:"title" example:"Hello" validate:"required"`
	Snippet string `json:"snippet" example:"...matched text..." validate:"required"`
}

// SearchResponse wraps search results.
type SearchResponse struct {
	Results []SearchResult `json:"results" validate:"required"`
}

// GraphNode is a node in the knowledge graph.
type GraphNode struct {
	ID    string `json:"id" example:"notes/hello.md" validate:"required"`
	Title string `json:"title,omitempty" example:"Hello"`
}

// GraphLink is an edge in the knowledge graph.
type GraphLink struct {
	Source string `json:"source" example:"notes/hello.md" validate:"required"`
	Target string `json:"target" example:"notes/world.md" validate:"required"`
}

// GraphResponse wraps the knowledge graph.
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes" validate:"required"`
	Links []GraphLink `json:"links" validate:"required"`
}

// AttachmentUploadResponse is returned after a successful attachment upload.
type AttachmentUploadResponse struct {
	Filename string `json:"filename" example:"image.png" validate:"required"`
	Size     int64  `json:"size" example:"12345" validate:"required"`
	URL      string `json:"url" example:"/attachments/image.png" validate:"required"`
}

// NoteDetailDTO mirrors NoteDetail with explicit types for swag.
type NoteDetailDTO = NoteDetail

// NoteListItemDTO mirrors NoteListItem for swag.
type NoteListItemDTO struct {
	Path      string    `json:"path" example:"notes/hello.md"`
	Title     string    `json:"title" example:"Hello"`
	Checksum  string    `json:"checksum" example:"abc123..."`
	Tags      []string  `json:"tags" example:"tag1,tag2"`
	UpdatedAt time.Time `json:"updated_at"`
}

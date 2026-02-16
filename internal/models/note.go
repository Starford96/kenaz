// Package models defines the domain types for Kenaz.
package models

import "time"

// Note represents a parsed Markdown file in the vault.
type Note struct {
	Path        string                 `json:"path"`
	Content     []byte                 `json:"-"`
	Body        string                 `json:"body"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Links       []string               `json:"links,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Checksum    string                 `json:"checksum"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NoteMetadata is a lightweight representation returned by list operations.
type NoteMetadata struct {
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Link represents a directed edge between two notes.
type Link struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "inline" or "frontmatter"
}

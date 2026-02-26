// Package models defines the domain types for Kenaz.
package models

import "time"

// NoteMetadata is a lightweight representation returned by list operations.
type NoteMetadata struct {
	Path      string    `json:"path"`
	Checksum  string    `json:"checksum"`
	UpdatedAt time.Time `json:"updated_at"`
}

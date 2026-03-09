package index

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NoteRow represents a row in the notes table.
type NoteRow struct {
	Path      string
	Title     string
	Checksum  string
	Tags      []string
	UpdatedAt time.Time
}

// SearchResult represents one search hit.
type SearchResult struct {
	Path    string
	Title   string
	Snippet string
}

// UpsertNote inserts or replaces a note, its FTS entry, and links within a transaction.
func (db *DB) UpsertNote(n NoteRow, body string, links []string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("index: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // best-effort on failure path

	tagsJSON, _ := json.Marshal(n.Tags)

	// Upsert notes table (includes body for fallback search).
	_, err = tx.Exec(`
		INSERT INTO notes (path, title, checksum, tags, body, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title      = excluded.title,
			checksum   = excluded.checksum,
			tags       = excluded.tags,
			body       = excluded.body,
			updated_at = excluded.updated_at
	`, n.Path, n.Title, n.Checksum, string(tagsJSON), body, n.UpdatedAt)
	if err != nil {
		return fmt.Errorf("index: upsert note: %w", err)
	}

	// FTS upsert (no-op when FTS5 tag is absent).
	if err := ftsUpsert(tx, n.Path, n.Title, body, n.Tags); err != nil {
		return err
	}

	// Replace links: delete old then bulk insert.
	if _, err := tx.Exec(`DELETE FROM links WHERE source = ?`, n.Path); err != nil {
		return fmt.Errorf("index: delete old links: %w", err)
	}
	if len(links) > 0 {
		stmt, err := tx.Prepare(`INSERT OR IGNORE INTO links (source, target, type) VALUES (?, ?, 'inline')`)
		if err != nil {
			return fmt.Errorf("index: prepare link insert: %w", err)
		}
		defer stmt.Close()
		for _, target := range links {
			if _, err := stmt.Exec(n.Path, target); err != nil {
				return fmt.Errorf("index: insert link: %w", err)
			}
		}
	}

	return tx.Commit()
}

// DeleteNote removes a note, its FTS entry, and outgoing links.
func (db *DB) DeleteNote(path string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("index: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := ftsDelete(tx, path); err != nil {
		return fmt.Errorf("index: fts delete: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM links WHERE source = ?`, path); err != nil {
		return fmt.Errorf("index: delete links: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM notes WHERE path = ?`, path); err != nil {
		return fmt.Errorf("index: delete note: %w", err)
	}

	return tx.Commit()
}

// GetChecksum returns the stored checksum for a note, or empty string if not found.
func (db *DB) GetChecksum(path string) (string, error) {
	var cs string
	err := db.conn.QueryRow(`SELECT checksum FROM notes WHERE path = ?`, path).Scan(&cs)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("index: get checksum %s: %w", path, err)
	}
	return cs, nil
}

// AllChecksums returns a map of path→checksum for every indexed note.
func (db *DB) AllChecksums() (map[string]string, error) {
	rows, err := db.conn.Query(`SELECT path, checksum FROM notes`)
	if err != nil {
		return nil, fmt.Errorf("index: all checksums: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var p, cs string
		if err := rows.Scan(&p, &cs); err != nil {
			return nil, err
		}
		out[p] = cs
	}
	return out, rows.Err()
}

// AllPaths returns every indexed note path.
func (db *DB) AllPaths() (map[string]struct{}, error) {
	rows, err := db.conn.Query(`SELECT path FROM notes`)
	if err != nil {
		return nil, fmt.Errorf("index: all paths: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out[p] = struct{}{}
	}
	return out, rows.Err()
}

// GetNote returns a single note row or nil if not found.
func (db *DB) GetNote(path string) (*NoteRow, error) {
	var n NoteRow
	var tagsJSON string
	err := db.conn.QueryRow(
		`SELECT path, title, checksum, tags, updated_at FROM notes WHERE path = ?`, path,
	).Scan(&n.Path, &n.Title, &n.Checksum, &tagsJSON, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("index: get note %s: %w", path, err)
	}
	_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
	n.Tags = nonNilSlice(n.Tags)
	return &n, nil
}

// ListNotes returns note rows with optional pagination and tag filter.
func (db *DB) ListNotes(limit, offset int, tag, sort string) ([]NoteRow, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if sort == "" {
		sort = "updated_at"
	}
	// Whitelist sort columns.
	switch sort {
	case "updated_at", "title", "path":
	default:
		sort = "updated_at"
	}

	where := ""
	args := []any{}
	if tag != "" {
		where = `WHERE tags LIKE ?`
		args = append(args, `%"`+tag+`"%`)
	}

	// Total count.
	var total int
	countQ := `SELECT count(*) FROM notes ` + where
	if err := db.conn.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("index: count notes: %w", err)
	}

	q := fmt.Sprintf(`SELECT path, title, checksum, tags, updated_at FROM notes %s ORDER BY %s DESC LIMIT ? OFFSET ?`, where, sort)
	queryArgs := append(args, limit, offset)
	rows, err := db.conn.Query(q, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("index: list notes: %w", err)
	}
	defer rows.Close()

	var out []NoteRow
	for rows.Next() {
		var n NoteRow
		var tagsJSON string
		if err := rows.Scan(&n.Path, &n.Title, &n.Checksum, &tagsJSON, &n.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
		n.Tags = nonNilSlice(n.Tags)
		out = append(out, n)
	}
	return out, total, rows.Err()
}

// GraphNode represents a node in the knowledge graph.
type GraphNode struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// GraphLink represents an edge in the knowledge graph.
type GraphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// Graph returns all nodes and links for graph visualization.
func (db *DB) Graph() ([]GraphNode, []GraphLink, error) {
	// Nodes from notes table.
	rows, err := db.conn.Query(`SELECT path, title FROM notes`)
	if err != nil {
		return nil, nil, fmt.Errorf("index: graph nodes: %w", err)
	}
	defer rows.Close()

	nodeSet := make(map[string]string)
	var nodes []GraphNode
	for rows.Next() {
		var path, title string
		if err := rows.Scan(&path, &title); err != nil {
			return nil, nil, err
		}
		nodeSet[path] = title
		nodes = append(nodes, GraphNode{ID: path, Title: title})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Links.
	lrows, err := db.conn.Query(`SELECT source, target FROM links`)
	if err != nil {
		return nil, nil, fmt.Errorf("index: graph links: %w", err)
	}
	defer lrows.Close()

	var links []GraphLink
	for lrows.Next() {
		var l GraphLink
		if err := lrows.Scan(&l.Source, &l.Target); err != nil {
			return nil, nil, err
		}
		// Add target as a node if it is not already indexed.
		if _, exists := nodeSet[l.Target]; !exists {
			nodeSet[l.Target] = ""
			nodes = append(nodes, GraphNode{ID: l.Target})
		}
		links = append(links, l)
	}
	return nodes, links, lrows.Err()
}

// Backlinks returns all note paths that link to the given target.
func (db *DB) Backlinks(target string) ([]string, error) {
	rows, err := db.conn.Query(`SELECT source FROM links WHERE target = ?`, target)
	if err != nil {
		return nil, fmt.Errorf("index: backlinks: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// MoveNote atomically updates a note's path in the index, including FTS and links.
func (db *DB) MoveNote(oldPath, newPath string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("index: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Read existing note data for FTS re-insert.
	var title, body, tagsJSON, cs string
	var updatedAt time.Time
	err = tx.QueryRow(
		`SELECT title, body, checksum, tags, updated_at FROM notes WHERE path = ?`, oldPath,
	).Scan(&title, &body, &cs, &tagsJSON, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("index: move note: old path not found")
		}
		return fmt.Errorf("index: move note read: %w", err)
	}

	// Delete old note row and insert new one (path is PK, can't just UPDATE).
	if _, err := tx.Exec(`DELETE FROM notes WHERE path = ?`, oldPath); err != nil {
		return fmt.Errorf("index: move delete old: %w", err)
	}
	if _, err := tx.Exec(
		`INSERT INTO notes (path, title, checksum, tags, body, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		newPath, title, cs, tagsJSON, body, updatedAt,
	); err != nil {
		return fmt.Errorf("index: move insert new: %w", err)
	}

	// Update FTS.
	if err := ftsDelete(tx, oldPath); err != nil {
		return fmt.Errorf("index: move fts delete: %w", err)
	}
	var tags []string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	if err := ftsUpsert(tx, newPath, title, body, tags); err != nil {
		return fmt.Errorf("index: move fts insert: %w", err)
	}

	// Update links where this note is the source.
	if _, err := tx.Exec(`UPDATE links SET source = ? WHERE source = ?`, newPath, oldPath); err != nil {
		return fmt.Errorf("index: move links source: %w", err)
	}
	// Update links where this note is the target (backlinks).
	// Wikilinks may store targets with or without .md extension.
	if _, err := tx.Exec(`UPDATE links SET target = ? WHERE target = ?`, newPath, oldPath); err != nil {
		return fmt.Errorf("index: move links target: %w", err)
	}
	oldNoExt := strings.TrimSuffix(oldPath, ".md")
	newNoExt := strings.TrimSuffix(newPath, ".md")
	if oldNoExt != oldPath {
		if _, err := tx.Exec(`UPDATE links SET target = ? WHERE target = ?`, newNoExt, oldNoExt); err != nil {
			return fmt.Errorf("index: move links target no-ext: %w", err)
		}
	}

	return tx.Commit()
}

// MoveNotesBatch atomically updates paths for multiple notes (directory rename).
func (db *DB) MoveNotesBatch(moves []PathMove) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("index: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, m := range moves {
		var title, body, tagsJSON, cs string
		var updatedAt time.Time
		err = tx.QueryRow(
			`SELECT title, body, checksum, tags, updated_at FROM notes WHERE path = ?`, m.OldPath,
		).Scan(&title, &body, &cs, &tagsJSON, &updatedAt)
		if err != nil {
			return fmt.Errorf("index: batch move read %s: %w", m.OldPath, err)
		}
		if _, err := tx.Exec(`DELETE FROM notes WHERE path = ?`, m.OldPath); err != nil {
			return fmt.Errorf("index: batch move delete %s: %w", m.OldPath, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO notes (path, title, checksum, tags, body, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			m.NewPath, title, cs, tagsJSON, body, updatedAt,
		); err != nil {
			return fmt.Errorf("index: batch move insert %s: %w", m.NewPath, err)
		}
		if err := ftsDelete(tx, m.OldPath); err != nil {
			return fmt.Errorf("index: batch move fts delete %s: %w", m.OldPath, err)
		}
		var tags []string
		_ = json.Unmarshal([]byte(tagsJSON), &tags)
		if err := ftsUpsert(tx, m.NewPath, title, body, tags); err != nil {
			return fmt.Errorf("index: batch move fts insert %s: %w", m.NewPath, err)
		}
		if _, err := tx.Exec(`UPDATE links SET source = ? WHERE source = ?`, m.NewPath, m.OldPath); err != nil {
			return fmt.Errorf("index: batch move links source %s: %w", m.OldPath, err)
		}
		if _, err := tx.Exec(`UPDATE links SET target = ? WHERE target = ?`, m.NewPath, m.OldPath); err != nil {
			return fmt.Errorf("index: batch move links target %s: %w", m.OldPath, err)
		}
		oldNoExt := strings.TrimSuffix(m.OldPath, ".md")
		newNoExt := strings.TrimSuffix(m.NewPath, ".md")
		if oldNoExt != m.OldPath {
			if _, err := tx.Exec(`UPDATE links SET target = ? WHERE target = ?`, newNoExt, oldNoExt); err != nil {
				return fmt.Errorf("index: batch move links target no-ext %s: %w", m.OldPath, err)
			}
		}
	}

	return tx.Commit()
}

// PathMove represents an old→new path mapping for batch moves.
type PathMove struct {
	OldPath string
	NewPath string
}

// NotesWithPrefix returns all notes whose path starts with the given prefix.
func (db *DB) NotesWithPrefix(prefix string) ([]NoteRow, error) {
	rows, err := db.conn.Query(
		`SELECT path, title, checksum, tags, updated_at FROM notes WHERE path LIKE ?`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("index: notes with prefix: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var out []NoteRow
	for rows.Next() {
		var n NoteRow
		var tagsJSON string
		if err := rows.Scan(&n.Path, &n.Title, &n.Checksum, &tagsJSON, &n.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
		n.Tags = nonNilSlice(n.Tags)
		out = append(out, n)
	}
	return out, rows.Err()
}

func nonNilSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

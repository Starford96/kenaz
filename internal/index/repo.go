package index

import (
	"encoding/json"
	"fmt"
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
	_, _ = tx.Exec(`DELETE FROM links WHERE source = ?`, n.Path)
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

	ftsDelete(tx, path)
	_, _ = tx.Exec(`DELETE FROM links WHERE source = ?`, path)
	_, _ = tx.Exec(`DELETE FROM notes WHERE path = ?`, path)

	return tx.Commit()
}

// GetChecksum returns the stored checksum for a note, or empty string if not found.
func (db *DB) GetChecksum(path string) (string, error) {
	var cs string
	err := db.conn.QueryRow(`SELECT checksum FROM notes WHERE path = ?`, path).Scan(&cs)
	if err != nil {
		return "", nil // not found is fine
	}
	return cs, nil
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
		return nil, nil // not found
	}
	_ = json.Unmarshal([]byte(tagsJSON), &n.Tags)
	if n.Tags == nil {
		n.Tags = []string{}
	}
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
	args := []interface{}{}
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
		if n.Tags == nil {
			n.Tags = []string{}
		}
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

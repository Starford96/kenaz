//go:build !sqlite_fts5

package index

import (
	"database/sql"
	"fmt"
)

func initFTS(_ *sql.DB) error {
	// FTS5 not available; full-text search uses LIKE fallback on the notes.body column.
	return nil
}

func ftsUpsert(_ *sql.Tx, _, _, _ string, _ []string) error {
	// Body is already stored in the notes table; nothing extra to do.
	return nil
}

func ftsDelete(_ *sql.Tx, _ string) error { return nil }

// Search performs a LIKE-based search (fallback when FTS5 is not compiled in).
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	like := "%" + query + "%"
	rows, err := db.conn.Query(`
		SELECT path, title, substr(body, 1, 200)
		FROM notes
		WHERE title LIKE ? OR body LIKE ? OR tags LIKE ?
		LIMIT ?
	`, like, like, like, limit)
	if err != nil {
		return nil, fmt.Errorf("index: search: %w", err)
	}
	defer rows.Close()

	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Path, &r.Title, &r.Snippet); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

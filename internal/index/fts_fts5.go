//go:build sqlite_fts5

package index

import (
	"database/sql"
	"fmt"
	"strings"
)

func initFTS(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
			path UNINDEXED,
			title,
			body,
			tags,
			tokenize = 'unicode61 remove_diacritics 2'
		);
	`)
	return err
}

func ftsUpsert(tx *sql.Tx, path, title, body string, tags []string) error {
	_, _ = tx.Exec(`DELETE FROM files_fts WHERE path = ?`, path)
	_, err := tx.Exec(`INSERT INTO files_fts (path, title, body, tags) VALUES (?, ?, ?, ?)`,
		path, title, body, strings.Join(tags, " "))
	if err != nil {
		return fmt.Errorf("index: upsert fts: %w", err)
	}
	return nil
}

func ftsDelete(tx *sql.Tx, path string) {
	_, _ = tx.Exec(`DELETE FROM files_fts WHERE path = ?`, path)
}

// Search performs an FTS5 full-text search and returns matching results with snippets.
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.conn.Query(`
		SELECT path,
		       title,
		       snippet(files_fts, 2, '<b>', '</b>', '...', 64)
		FROM files_fts
		WHERE files_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
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

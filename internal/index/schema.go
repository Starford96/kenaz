// Package index provides SQLite-backed note indexing with optional FTS5 full-text search.
package index

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const coreSchemaSQL = `
CREATE TABLE IF NOT EXISTS notes (
	path       TEXT PRIMARY KEY,
	title      TEXT NOT NULL DEFAULT '',
	checksum   TEXT NOT NULL DEFAULT '',
	tags       TEXT NOT NULL DEFAULT '[]',
	body       TEXT NOT NULL DEFAULT '',
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS links (
	source TEXT NOT NULL,
	target TEXT NOT NULL,
	type   TEXT NOT NULL DEFAULT 'inline',
	UNIQUE(source, target)
);

CREATE INDEX IF NOT EXISTS idx_links_source ON links(source);
CREATE INDEX IF NOT EXISTS idx_links_target ON links(target);
`

// DB wraps a sql.DB with index-specific operations.
type DB struct {
	conn *sql.DB
}

// Open opens (or creates) the SQLite database and applies the schema.
func Open(dsn string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dsn+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("index: open db: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("index: ping: %w", err)
	}
	if _, err := conn.Exec(coreSchemaSQL); err != nil {
		conn.Close()
		return nil, fmt.Errorf("index: apply core schema: %w", err)
	}
	if err := initFTS(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("index: apply fts schema: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

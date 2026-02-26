package index

// NoteIndex defines the interface for note indexing operations.
// Consumers should depend on this interface rather than the concrete *DB type
// to facilitate testing with mocks.
type NoteIndex interface {
	UpsertNote(n NoteRow, body string, links []string) error
	DeleteNote(path string) error
	GetChecksum(path string) (string, error)
	GetNote(path string) (*NoteRow, error)
	ListNotes(limit, offset int, tag, sort string) ([]NoteRow, int, error)
	Search(query string, limit int) ([]SearchResult, error)
	Graph() ([]GraphNode, []GraphLink, error)
	Backlinks(target string) ([]string, error)
	AllPaths() (map[string]struct{}, error)
	AllChecksums() (map[string]string, error)
	Close() error
}

// Verify *DB satisfies NoteIndex at compile time.
var _ NoteIndex = (*DB)(nil)

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/storage"
)

// testEnv sets up a temp vault, SQLite DB, service, and router for testing.
// authEnabled=false means disabled mode; authEnabled=true with non-empty token means token mode.
func testEnv(t *testing.T, authToken string) (*Service, http.Handler) {
	t.Helper()
	enabled := authToken != ""
	return testEnvFull(t, enabled, authToken)
}

func testEnvFull(t *testing.T, authEnabled bool, authToken string) (*Service, http.Handler) {
	t.Helper()

	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	dbFile, err := os.CreateTemp("", "kenaz-api-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbFile.Name()) })

	db, err := index.Open(dbFile.Name())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	svc := NewService(store, db)
	router := NewRouter(svc, authEnabled, authToken, nil)
	return svc, router
}

func TestCreateAndGetNote(t *testing.T) {
	_, router := testEnv(t, "")

	// Create note.
	body, _ := json.Marshal(map[string]string{"path": "hello.md", "content": "# Hello\nWorld"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", w.Code, w.Body.String())
	}

	// Get note.
	req = httptest.NewRequest(http.MethodGet, "/notes/hello.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d", w.Code)
	}
	var note NoteDetail
	_ = json.Unmarshal(w.Body.Bytes(), &note)
	if note.Path != "hello.md" {
		t.Errorf("path = %q", note.Path)
	}
	if note.Title != "Hello" {
		t.Errorf("title = %q, want Hello", note.Title)
	}
}

func TestCreateDuplicate(t *testing.T) {
	_, router := testEnv(t, "")

	body, _ := json.Marshal(map[string]string{"path": "dup.md", "content": "a"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create = %d", w.Code)
	}

	// Second create should 409.
	req = httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate create = %d, want 409", w.Code)
	}
}

func TestUpdateWithOptimisticLocking(t *testing.T) {
	_, router := testEnv(t, "")

	// Create.
	body, _ := json.Marshal(map[string]string{"path": "lock.md", "content": "v1"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d", w.Code)
	}
	var created NoteDetail
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Update with correct checksum.
	updateBody, _ := json.Marshal(map[string]string{"content": "v2"})
	req = httptest.NewRequest(http.MethodPut, "/notes/lock.md", bytes.NewReader(updateBody))
	req.Header.Set("If-Match", created.Checksum)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update with correct checksum = %d, body = %s", w.Code, w.Body.String())
	}

	// Update with stale checksum → 409.
	req = httptest.NewRequest(http.MethodPut, "/notes/lock.md", bytes.NewReader(updateBody))
	req.Header.Set("If-Match", created.Checksum) // stale now
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("update with stale checksum = %d, want 409", w.Code)
	}
}

func TestUpdateWithoutIfMatch(t *testing.T) {
	_, router := testEnv(t, "")

	body, _ := json.Marshal(map[string]string{"path": "nolock.md", "content": "v1"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Update without If-Match should succeed (no locking enforced).
	updateBody, _ := json.Marshal(map[string]string{"content": "v2"})
	req = httptest.NewRequest(http.MethodPut, "/notes/nolock.md", bytes.NewReader(updateBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("update without If-Match = %d, want 200", w.Code)
	}
}

func TestDeleteNote(t *testing.T) {
	_, router := testEnv(t, "")

	body, _ := json.Marshal(map[string]string{"path": "bye.md", "content": "gone"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodDelete, "/notes/bye.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete = %d, want 204", w.Code)
	}

	// GET should now 404.
	req = httptest.NewRequest(http.MethodGet, "/notes/bye.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", w.Code)
	}
}

func TestListNotes(t *testing.T) {
	_, router := testEnv(t, "")

	for _, name := range []string{"a.md", "b.md"} {
		body, _ := json.Marshal(map[string]string{"path": name, "content": "# " + name})
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/notes?limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list = %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	notes := resp["notes"].([]interface{})
	if len(notes) != 2 {
		t.Errorf("len(notes) = %d, want 2", len(notes))
	}
}

func TestSearchEndpoint(t *testing.T) {
	_, router := testEnv(t, "")

	body, _ := json.Marshal(map[string]string{"path": "find.md", "content": "uniquetoken here"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodGet, "/search?q=uniquetoken", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("search = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	results := resp["results"].([]interface{})
	if len(results) != 1 {
		t.Errorf("search results = %d, want 1", len(results))
	}
}

func TestGraphEndpoint(t *testing.T) {
	_, router := testEnv(t, "")

	for _, n := range []struct{ path, content string }{
		{"a.md", "links to [[b]]"},
		{"b.md", "links to [[a]]"},
	} {
		body, _ := json.Marshal(map[string]string{"path": n.path, "content": n.content})
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/graph", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graph = %d", w.Code)
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	nodes := resp["nodes"].([]interface{})
	links := resp["links"].([]interface{})
	if len(nodes) < 2 {
		t.Errorf("nodes = %d, want >= 2", len(nodes))
	}
	if len(links) < 2 {
		t.Errorf("links = %d, want >= 2", len(links))
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	_, router := testEnv(t, "secret123")

	body, _ := json.Marshal(map[string]string{"path": "auth.md", "content": "test"})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("authed create = %d, want 201", w.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	_, router := testEnv(t, "secret123")

	req := httptest.NewRequest(http.MethodGet, "/notes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unauthed = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_WrongToken(t *testing.T) {
	_, router := testEnv(t, "secret123")

	req := httptest.NewRequest(http.MethodGet, "/notes", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	_, router := testEnv(t, "")

	req := httptest.NewRequest(http.MethodGet, "/notes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("no auth = %d, want 200", w.Code)
	}
}

func TestGetNote_NotFound(t *testing.T) {
	_, router := testEnv(t, "")

	req := httptest.NewRequest(http.MethodGet, "/notes/nope.md", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("missing note = %d, want 404", w.Code)
	}
}

func TestUpdateNote_NotFound(t *testing.T) {
	_, router := testEnv(t, "")

	body, _ := json.Marshal(map[string]string{"content": "x"})
	req := httptest.NewRequest(http.MethodPut, "/notes/ghost.md", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("update missing = %d, want 404", w.Code)
	}
}

func TestSearchMissingQuery(t *testing.T) {
	_, router := testEnv(t, "")

	req := httptest.NewRequest(http.MethodGet, "/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("search no query = %d, want 400", w.Code)
	}
}

// SSE endpoint auth tests.

func TestSSEEvents_AuthProtected(t *testing.T) {
	_, router := testEnvWithSSE(t, true, "secret")

	// No token → 401.
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("SSE no auth = %d, want 401", w.Code)
	}
}

func TestSSEEvents_AuthDisabled(t *testing.T) {
	_, router := testEnvWithSSE(t, false, "")

	// Disabled mode → should not 401. SSE handler will write 200 and block,
	// so we cancel the context after a short time.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Error("SSE should not require auth when disabled")
	}
}

func TestSSEEvents_ValidToken(t *testing.T) {
	_, router := testEnvWithSSE(t, true, "tok")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Error("SSE with valid token should not 401")
	}
}

// testEnvWithSSE creates a router with a dummy SSE handler to test auth on /events.
func testEnvWithSSE(t *testing.T, authEnabled bool, token string) (*Service, http.Handler) {
	t.Helper()

	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}
	dbFile, err := os.CreateTemp("", "kenaz-sse-test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	dbFile.Close()
	t.Cleanup(func() { os.Remove(dbFile.Name()) })
	db, err := index.Open(dbFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	svc := NewService(store, db)

	// Minimal SSE handler stub — writes headers and blocks until context done.
	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	})

	router := NewRouter(svc, authEnabled, token, sseHandler)
	return svc, router
}

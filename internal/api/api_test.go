package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/noteservice"
	"github.com/starford/kenaz/internal/storage"
)

// testEnv sets up a temp vault, SQLite DB, service, and router for testing.
// authEnabled=false means disabled mode; authEnabled=true with non-empty token means token mode.
func testEnv(t *testing.T, authToken string) (*noteservice.Service, http.Handler) {
	t.Helper()
	enabled := authToken != ""
	return testEnvFull(t, enabled, authToken)
}

func testEnvFull(t *testing.T, authEnabled bool, authToken string) (*noteservice.Service, http.Handler) {
	t.Helper()
	svc, router, _ := testEnvWithVault(t, authEnabled, authToken)
	return svc, router
}

func testEnvWithVault(t *testing.T, authEnabled bool, authToken string) (*noteservice.Service, http.Handler, string) {
	t.Helper()

	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir, nil)
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

	svc := noteservice.NewService(store, db)
	router := NewRouter(svc, authEnabled, authToken, nil, vaultDir)
	return svc, router, vaultDir
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

func TestDeleteDir(t *testing.T) {
	_, router := testEnv(t, "")

	// Create two notes inside a subdirectory.
	for _, name := range []string{"sub/a.md", "sub/b.md"} {
		body, _ := json.Marshal(map[string]string{"path": name, "content": "# " + name})
		req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("create %s = %d, want 201", name, w.Code)
		}
	}

	// Delete directory via ?dir=true query param.
	req := httptest.NewRequest(http.MethodDelete, "/notes/sub?dir=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("delete dir = %d, want 204", w.Code)
	}

	// Both notes should now 404.
	for _, name := range []string{"sub/a.md", "sub/b.md"} {
		req = httptest.NewRequest(http.MethodGet, "/notes/"+name, nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("get %s after delete dir = %d, want 404", name, w.Code)
		}
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
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	notes := resp["notes"].([]any)
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
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	results := resp["results"].([]any)
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
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	nodes := resp["nodes"].([]any)
	links := resp["links"].([]any)
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
func testEnvWithSSE(t *testing.T, authEnabled bool, token string) (*noteservice.Service, http.Handler) {
	t.Helper()

	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir, nil)
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

	svc := noteservice.NewService(store, db)

	// Minimal SSE handler stub — writes headers and blocks until context done.
	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	})

	router := NewRouter(svc, authEnabled, token, sseHandler, vaultDir)
	return svc, router
}

// Attachment tests.

func uploadFile(t *testing.T, router http.Handler, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(part, bytes.NewReader(content))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestUploadAndServeAttachment(t *testing.T) {
	_, router, vaultDir := testEnvWithVault(t, false, "")

	// Upload.
	w := uploadFile(t, router, "test.png", []byte("fake-png-data"))
	if w.Code != http.StatusCreated {
		t.Fatalf("upload = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["filename"] != "test.png" {
		t.Errorf("filename = %v", resp["filename"])
	}

	// Verify file on disk.
	data, err := os.ReadFile(filepath.Join(vaultDir, "attachments", "test.png"))
	if err != nil {
		t.Fatalf("file not on disk: %v", err)
	}
	if string(data) != "fake-png-data" {
		t.Errorf("content mismatch")
	}
}

func TestServeAttachment_NotFound(t *testing.T) {
	ah := NewAttachmentHandler(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/attachments/nope.png", nil)

	// chi URL params need a router context; test the handler directly with a
	// chi router to get proper URL param extraction.
	r := chi.NewRouter()
	r.Get("/attachments/{filename}", ah.ServeFile)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("missing attachment = %d, want 404", w.Code)
	}
}

func TestServeAttachment_TraversalBlocked(t *testing.T) {
	ah := NewAttachmentHandler(t.TempDir())
	r := chi.NewRouter()
	r.Get("/attachments/{filename}", ah.ServeFile)

	for _, name := range []string{"../secret.md", "../../etc/passwd"} {
		req := httptest.NewRequest(http.MethodGet, "/attachments/"+name, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		// chi may not route the traversal paths at all (404), or our handler rejects (400).
		if w.Code == http.StatusOK {
			t.Errorf("traversal %q should not return 200", name)
		}
	}
}

func TestUploadAttachment_InvalidFilename(t *testing.T) {
	_, router, vaultDir := testEnvWithVault(t, false, "")
	// multipart headers may clean "../" so we also verify file doesn't land outside.
	w := uploadFile(t, router, "../escape.txt", []byte("bad"))
	// Either rejected (400) or the cleaned name lands safely inside attachments.
	if w.Code == http.StatusCreated {
		// Verify no file outside vault.
		if _, err := os.Stat(filepath.Join(vaultDir, "..", "escape.txt")); err == nil {
			t.Error("file escaped vault directory")
		}
	}
}

func TestUploadAttachment_AuthProtected(t *testing.T) {
	_, router, _ := testEnvWithVault(t, true, "secret")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "x.png")
	_, _ = part.Write([]byte("data"))
	mw.Close()

	// No token → 401.
	req := httptest.NewRequest(http.MethodPost, "/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("upload no auth = %d, want 401", w.Code)
	}
}

func TestUploadAttachment_MissingFileField(t *testing.T) {
	_, router, _ := testEnvWithVault(t, false, "")

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("wrong", "data")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/attachments", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing field = %d, want 400", w.Code)
	}
}

// Rename tests.

func createTestNote(t *testing.T, router http.Handler, path, content string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"path": path, "content": content})
	req := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %s: status = %d, body = %s", path, w.Code, w.Body.String())
	}
}

func postRename(router http.Handler, oldPath, newPath string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"old_path": oldPath, "new_path": newPath})
	req := httptest.NewRequest(http.MethodPost, "/notes/rename", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestRenameNote_API(t *testing.T) {
	_, router := testEnv(t, "")
	createTestNote(t, router, "old.md", "# Old")

	w := postRename(router, "old.md", "new.md")
	if w.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["path"] != "new.md" {
		t.Errorf("response path = %v, want new.md", resp["path"])
	}

	// New path accessible.
	req := httptest.NewRequest(http.MethodGet, "/notes/new.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET new.md = %d, want 200", w.Code)
	}

	// Old path gone.
	req = httptest.NewRequest(http.MethodGet, "/notes/old.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("GET old.md = %d, want 404", w.Code)
	}
}

func TestRenameNote_API_NotFound(t *testing.T) {
	_, router := testEnv(t, "")
	w := postRename(router, "nope.md", "x.md")
	if w.Code != http.StatusNotFound {
		t.Errorf("rename missing = %d, want 404", w.Code)
	}
}

func TestRenameNote_API_Conflict(t *testing.T) {
	_, router := testEnv(t, "")
	createTestNote(t, router, "a.md", "# A")
	createTestNote(t, router, "b.md", "# B")

	w := postRename(router, "a.md", "b.md")
	if w.Code != http.StatusConflict {
		t.Errorf("rename conflict = %d, want 409", w.Code)
	}
}

func TestRenameNote_API_MissingFields(t *testing.T) {
	_, router := testEnv(t, "")

	// Empty old_path.
	body, _ := json.Marshal(map[string]string{"old_path": "", "new_path": "x.md"})
	req := httptest.NewRequest(http.MethodPost, "/notes/rename", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty old_path = %d, want 400", w.Code)
	}

	// Empty body.
	req = httptest.NewRequest(http.MethodPost, "/notes/rename", bytes.NewReader([]byte("{}")))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty body = %d, want 400", w.Code)
	}
}

func TestRenameNote_API_SamePath(t *testing.T) {
	_, router := testEnv(t, "")
	w := postRename(router, "a.md", "a.md")
	if w.Code != http.StatusBadRequest {
		t.Errorf("same path = %d, want 400", w.Code)
	}
}

func TestRenameDir_API(t *testing.T) {
	_, router := testEnv(t, "")
	createTestNote(t, router, "dir/a.md", "# A")
	createTestNote(t, router, "dir/b.md", "# B")

	w := postRename(router, "dir/", "newdir/")
	if w.Code != http.StatusOK {
		t.Fatalf("rename dir status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	moved, ok := resp["moved"].([]any)
	if !ok || len(moved) != 2 {
		t.Fatalf("moved = %v, want 2 items", resp["moved"])
	}

	// Verify new paths accessible.
	req := httptest.NewRequest(http.MethodGet, "/notes/newdir/a.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET newdir/a.md = %d, want 200", w.Code)
	}
}

func TestRenameNote_API_WikilinkUpdate(t *testing.T) {
	_, router := testEnv(t, "")
	createTestNote(t, router, "target.md", "# Target")
	createTestNote(t, router, "ref.md", "See [[target.md]]")

	w := postRename(router, "target.md", "moved.md")
	if w.Code != http.StatusOK {
		t.Fatalf("rename = %d, body = %s", w.Code, w.Body.String())
	}

	// Check that ref.md content was updated.
	req := httptest.NewRequest(http.MethodGet, "/notes/ref.md", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET ref.md = %d", w.Code)
	}
	var note map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &note)
	content, _ := note["content"].(string)
	if !strings.Contains(content, "[[moved.md]]") {
		t.Errorf("ref.md content should contain [[moved.md]], got: %s", content)
	}
}

func TestRenameNote_API_AuthProtected(t *testing.T) {
	_, router := testEnv(t, "secret")

	// No token → 401.
	w := postRename(router, "a.md", "b.md")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("rename no auth = %d, want 401", w.Code)
	}

	// With token → should work (will 404 since note doesn't exist, but not 401).
	body, _ := json.Marshal(map[string]string{"old_path": "a.md", "new_path": "b.md"})
	req := httptest.NewRequest(http.MethodPost, "/notes/rename", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code == http.StatusUnauthorized {
		t.Errorf("rename with token should not 401, got %d", w.Code)
	}
}

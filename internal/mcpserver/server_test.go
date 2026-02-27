package mcpserver

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/noteservice"
	"github.com/starford/kenaz/internal/storage"
)

func testServer(t *testing.T) (*Server, storage.Provider) {
	t.Helper()

	vaultDir := t.TempDir()
	store, err := storage.NewFS(vaultDir)
	if err != nil {
		t.Fatal(err)
	}

	dbFile, err := os.CreateTemp("", "kenaz-mcp-test-*.db")
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
	srv := New(svc, store)
	return srv, store
}

func callTool(t *testing.T, srv *Server, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Method = "tools/call"
	req.Params.Name = name
	req.Params.Arguments = args

	// Find the handler via the MCPServer's tool list. We call the handler directly.
	// Since mcp-go doesn't expose a direct "call tool" test helper, we test
	// through the tool handler functions directly.
	var result *mcp.CallToolResult
	var err error

	switch name {
	case "search_notes":
		result, err = srv.searchNotes(ctx, req)
	case "read_note":
		result, err = srv.readNote(ctx, req)
	case "create_note":
		result, err = srv.createNote(ctx, req)
	case "list_notes":
		result, err = srv.listNotes(ctx, req)
	case "update_note":
		result, err = srv.updateNote(ctx, req)
	case "delete_note":
		result, err = srv.deleteNote(ctx, req)
	case "get_backlinks":
		result, err = srv.getBacklinks(ctx, req)
	case "get_note_contract":
		result, err = srv.getNoteContract(ctx, req)
	case "upload_asset":
		result, err = srv.uploadAsset(ctx, req)
	default:
		t.Fatalf("unknown tool: %s", name)
	}

	if err != nil {
		t.Fatalf("tool %s error: %v", name, err)
	}
	return result
}

func resultText(r *mcp.CallToolResult) string {
	if len(r.Content) > 0 {
		if tc, ok := r.Content[0].(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestCreateAndReadNote(t *testing.T) {
	srv, _ := testServer(t)

	r := callTool(t, srv, "create_note", map[string]any{
		"path":    "test.md",
		"content": "# Test\nHello",
	})
	text := resultText(r)
	if text != "created: test.md" {
		t.Errorf("create result = %q", text)
	}

	r = callTool(t, srv, "read_note", map[string]any{
		"path": "test.md",
	})
	text = resultText(r)
	if text != "# Test\nHello" {
		t.Errorf("read result = %q", text)
	}
}

func TestListNotes(t *testing.T) {
	srv, _ := testServer(t)
	_ = callTool(t, srv, "create_note", map[string]any{
		"path": "a.md", "content": "# A",
	})
	_ = callTool(t, srv, "create_note", map[string]any{
		"path": "b.md", "content": "# B",
	})

	r := callTool(t, srv, "list_notes", map[string]any{})
	text := resultText(r)
	if text == "" {
		t.Error("list returned empty")
	}
}

func TestReadNoteMissing(t *testing.T) {
	srv, _ := testServer(t)
	r := callTool(t, srv, "read_note", map[string]any{"path": "nope.md"})
	if !r.IsError {
		t.Error("expected error for missing note")
	}
}

func TestGetNoteContract(t *testing.T) {
	srv, _ := testServer(t)
	r := callTool(t, srv, "get_note_contract", map[string]any{})
	text := resultText(r)
	if text == "" {
		t.Fatal("contract is empty")
	}
	if !strings.Contains(text, "YAML frontmatter is mandatory") {
		t.Error("contract missing expected content")
	}
	if !strings.Contains(text, "[[wikilinks]]") {
		t.Error("contract missing wikilink guidance")
	}
}

func TestReadNoteFormatResource(t *testing.T) {
	srv, _ := testServer(t)
	ctx := context.Background()
	req := mcp.ReadResourceRequest{}
	req.Params.URI = "kenaz://note-format"

	contents, err := srv.readNoteFormatResource(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	tc, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatal("expected TextResourceContents")
	}
	if tc.URI != "kenaz://note-format" {
		t.Errorf("URI = %q", tc.URI)
	}
	if tc.MIMEType != "text/markdown" {
		t.Errorf("MIMEType = %q", tc.MIMEType)
	}
	if !strings.Contains(tc.Text, "title") {
		t.Error("resource text missing 'title'")
	}
}

func TestGetBacklinks(t *testing.T) {
	srv, _ := testServer(t)
	_ = callTool(t, srv, "create_note", map[string]any{
		"path":    "a.md",
		"content": "links to [[b]]",
	})

	r := callTool(t, srv, "get_backlinks", map[string]any{"path": "b"})
	text := resultText(r)
	if text != "a.md" {
		t.Errorf("backlinks = %q, want a.md", text)
	}
}

func TestUpdateNote(t *testing.T) {
	srv, _ := testServer(t)

	_ = callTool(t, srv, "create_note", map[string]any{
		"path":    "upd.md",
		"content": "# Original\nv1",
	})

	r := callTool(t, srv, "update_note", map[string]any{
		"path":    "upd.md",
		"content": "# Updated\nv2",
	})
	text := resultText(r)
	if text != "updated: upd.md" {
		t.Errorf("update result = %q", text)
	}

	r = callTool(t, srv, "read_note", map[string]any{"path": "upd.md"})
	text = resultText(r)
	if text != "# Updated\nv2" {
		t.Errorf("read after update = %q", text)
	}
}

func TestUpdateNoteNotFound(t *testing.T) {
	srv, _ := testServer(t)
	r := callTool(t, srv, "update_note", map[string]any{
		"path":    "missing.md",
		"content": "# Hello",
	})
	if !r.IsError {
		t.Error("expected error for updating non-existent note")
	}
}

func TestUpdateNoteChecksumConflict(t *testing.T) {
	srv, _ := testServer(t)

	_ = callTool(t, srv, "create_note", map[string]any{
		"path":    "cs.md",
		"content": "# CS\noriginal",
	})

	r := callTool(t, srv, "update_note", map[string]any{
		"path":     "cs.md",
		"content":  "# CS\nnew",
		"checksum": "0000000000000000000000000000000000000000000000000000000000000000",
	})
	if !r.IsError {
		t.Error("expected conflict error for wrong checksum")
	}
	text := resultText(r)
	if !strings.Contains(text, "conflict") {
		t.Errorf("expected conflict message, got %q", text)
	}
}

func TestDeleteNote(t *testing.T) {
	srv, _ := testServer(t)

	_ = callTool(t, srv, "create_note", map[string]any{
		"path":    "del.md",
		"content": "# Delete me",
	})

	r := callTool(t, srv, "delete_note", map[string]any{"path": "del.md"})
	text := resultText(r)
	if text != "deleted: del.md" {
		t.Errorf("delete result = %q", text)
	}

	r = callTool(t, srv, "read_note", map[string]any{"path": "del.md"})
	if !r.IsError {
		t.Error("expected error reading deleted note")
	}
}

func TestDeleteNoteMissing(t *testing.T) {
	srv, _ := testServer(t)
	r := callTool(t, srv, "delete_note", map[string]any{"path": "ghost.md"})
	if !r.IsError {
		t.Error("expected error deleting non-existent note")
	}
}

// --- upload_asset tests ---

// 1x1 red PNG pixel, base64-encoded.
const testPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

func TestUploadAssetBase64PNG(t *testing.T) {
	srv, store := testServer(t)

	dataURI := "data:image/png;base64," + testPNGBase64
	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "pixel.png",
	})
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}

	text := resultText(r)
	if !strings.Contains(text, `"/attachments/pixel.png"`) {
		t.Errorf("unexpected savedPath in result: %s", text)
	}
	if !strings.Contains(text, `![pixel.png](/attachments/pixel.png)`) {
		t.Errorf("unexpected markdownImage in result: %s", text)
	}

	data, err := store.Read("attachments/pixel.png")
	if err != nil {
		t.Fatalf("file not found on disk: %v", err)
	}
	if len(data) == 0 {
		t.Error("saved file is empty")
	}
}

func TestUploadAssetAutoFilename(t *testing.T) {
	srv, _ := testServer(t)

	dataURI := "data:image/png;base64," + testPNGBase64
	r := callTool(t, srv, "upload_asset", map[string]any{
		"url": dataURI,
	})
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}

	text := resultText(r)
	if !strings.Contains(text, "/attachments/") {
		t.Errorf("expected path under /attachments/, got: %s", text)
	}
	if !strings.Contains(text, ".png") {
		t.Errorf("expected .png extension, got: %s", text)
	}
}

func TestUploadAssetBadExtension(t *testing.T) {
	srv, _ := testServer(t)

	dataURI := "data:image/png;base64," + testPNGBase64
	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "malware.exe",
	})
	if !r.IsError {
		t.Error("expected error for .exe extension")
	}
	text := resultText(r)
	if !strings.Contains(text, "unsupported file extension") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestUploadAssetMagicBytesMismatch(t *testing.T) {
	srv, _ := testServer(t)

	dataURI := "data:image/png;base64," + testPNGBase64
	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "fake.pdf",
	})
	if !r.IsError {
		t.Error("expected error for magic bytes mismatch")
	}
	text := resultText(r)
	if !strings.Contains(text, "content does not match") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestUploadAssetOverwriteProtection(t *testing.T) {
	srv, store := testServer(t)

	if err := store.Write("attachments/image.png", []byte("existing")); err != nil {
		t.Fatal(err)
	}

	dataURI := "data:image/png;base64," + testPNGBase64
	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "image.png",
	})
	if !r.IsError {
		t.Error("expected error for overwriting existing file")
	}
	text := resultText(r)
	if !strings.Contains(text, "already exists") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestUploadAssetBlockedLoopback(t *testing.T) {
	srv, _ := testServer(t)

	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      "http://127.0.0.1:8080/secret.png",
		"filename": "secret.png",
	})
	if !r.IsError {
		t.Error("expected error for loopback address")
	}
	text := resultText(r)
	if !strings.Contains(text, "blocked host") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestUploadAssetBlockedMetadata(t *testing.T) {
	srv, _ := testServer(t)

	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      "http://169.254.169.254/latest/meta-data/",
		"filename": "metadata.pdf",
	})
	if !r.IsError {
		t.Error("expected error for metadata address")
	}
	text := resultText(r)
	if !strings.Contains(text, "blocked host") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestUploadAssetUnsupportedDataURIMime(t *testing.T) {
	srv, _ := testServer(t)

	r := callTool(t, srv, "upload_asset", map[string]any{
		"url": "data:application/octet-stream;base64,AAAA",
	})
	if !r.IsError {
		t.Error("expected error for unsupported MIME type")
	}
}

func TestUploadAssetSVGValid(t *testing.T) {
	srv, store := testServer(t)

	svgContent := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`)
	encoded := base64.StdEncoding.EncodeToString(svgContent)
	dataURI := "data:image/svg+xml;base64," + encoded

	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "icon.svg",
	})
	if r.IsError {
		t.Fatalf("unexpected error: %s", resultText(r))
	}

	data, err := store.Read("attachments/icon.svg")
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}
	if !strings.Contains(string(data), "<svg") {
		t.Error("saved file doesn't contain <svg")
	}
}

func TestUploadAssetSVGInvalid(t *testing.T) {
	srv, _ := testServer(t)

	encoded := base64.StdEncoding.EncodeToString([]byte("not an svg at all"))
	dataURI := "data:image/svg+xml;base64," + encoded

	r := callTool(t, srv, "upload_asset", map[string]any{
		"url":      dataURI,
		"filename": "fake.svg",
	})
	if !r.IsError {
		t.Error("expected error for invalid SVG content")
	}
}

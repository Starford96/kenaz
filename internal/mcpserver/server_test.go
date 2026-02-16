package mcpserver

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/starford/kenaz/internal/index"
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

	srv := New(store, db)
	return srv, store
}

func callTool(t *testing.T, srv *Server, name string, args map[string]interface{}) *mcp.CallToolResult {
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
	case "get_backlinks":
		result, err = srv.getBacklinks(ctx, req)
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

	r := callTool(t, srv, "create_note", map[string]interface{}{
		"path":    "test.md",
		"content": "# Test\nHello",
	})
	text := resultText(r)
	if text != "created: test.md" {
		t.Errorf("create result = %q", text)
	}

	r = callTool(t, srv, "read_note", map[string]interface{}{
		"path": "test.md",
	})
	text = resultText(r)
	if text != "# Test\nHello" {
		t.Errorf("read result = %q", text)
	}
}

func TestListNotes(t *testing.T) {
	srv, store := testServer(t)
	_ = store.Write("a.md", []byte("a"))
	_ = store.Write("b.md", []byte("b"))

	r := callTool(t, srv, "list_notes", map[string]interface{}{})
	text := resultText(r)
	if text == "" {
		t.Error("list returned empty")
	}
}

func TestReadNoteMissing(t *testing.T) {
	srv, _ := testServer(t)
	r := callTool(t, srv, "read_note", map[string]interface{}{"path": "nope.md"})
	if !r.IsError {
		t.Error("expected error for missing note")
	}
}

func TestGetBacklinks(t *testing.T) {
	srv, _ := testServer(t)
	_ = callTool(t, srv, "create_note", map[string]interface{}{
		"path":    "a.md",
		"content": "links to [[b]]",
	})

	r := callTool(t, srv, "get_backlinks", map[string]interface{}{"path": "b"})
	text := resultText(r)
	if text != "a.md" {
		t.Errorf("backlinks = %q, want a.md", text)
	}
}

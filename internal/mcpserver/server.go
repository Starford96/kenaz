// Package mcpserver provides an MCP (Model Context Protocol) server
// that exposes Kenaz tools for LLM integration via stdio transport.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/starford/kenaz/internal/index"
	"github.com/starford/kenaz/internal/parser"
	"github.com/starford/kenaz/internal/storage"
)

// Server wraps the MCP server with Kenaz tools.
type Server struct {
	mcp   *server.MCPServer
	store storage.Provider
	db    *index.DB
}

// New creates a new MCP server with all Kenaz tools registered.
func New(store storage.Provider, db *index.DB) *Server {
	s := &Server{store: store, db: db}

	s.mcp = server.NewMCPServer(
		"Kenaz",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),
	)

	s.mcp.AddTool(mcp.NewTool("search_notes",
		mcp.WithDescription("Full-text search through notes content and titles."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query string")),
	), s.searchNotes)

	s.mcp.AddTool(mcp.NewTool("read_note",
		mcp.WithDescription("Read the full content of a Markdown note."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the note (e.g. folder/note.md)")),
	), s.readNote)

	s.mcp.AddTool(mcp.NewTool("create_note",
		mcp.WithDescription("Create a new Markdown note at the specified path. "+
			"Content MUST follow the canonical note format (YAML frontmatter with title, "+
			"optional tags, Markdown body with [[wikilinks]]). Read the contract first via "+
			"the get_note_contract tool or the kenaz://note-format resource."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path for the new note (must end with .md)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown content following the Kenaz note format contract")),
	), s.createNote)

	s.mcp.AddTool(mcp.NewTool("get_note_contract",
		mcp.WithDescription("Returns the canonical Kenaz note format contract. "+
			"Call this before creating or updating notes to ensure correct structure."),
	), s.getNoteContract)

	s.mcp.AddTool(mcp.NewTool("list_notes",
		mcp.WithDescription("List all notes or notes in a specific folder."),
		mcp.WithString("folder", mcp.Description("Optional folder to list (empty for all)")),
	), s.listNotes)

	s.mcp.AddTool(mcp.NewTool("get_backlinks",
		mcp.WithDescription("Find all notes that link to the specified note."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path of the note to find backlinks for")),
	), s.getBacklinks)

	// Resource: note format contract.
	s.mcp.AddResource(
		mcp.NewResource("kenaz://note-format", "Note Format Contract",
			mcp.WithResourceDescription("Canonical Markdown note format that all notes must follow."),
			mcp.WithMIMEType("text/markdown"),
		),
		s.readNoteFormatResource,
	)

	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcp)
}

// MCPServer returns the underlying server for testing.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcp
}

func (s *Server) searchNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	results, err := s.db.Search(query, 20)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}

func (s *Server) readNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	data, err := s.store.Read(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("not found: %s", path)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) createNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check existence.
	if _, readErr := s.store.Read(path); readErr == nil {
		return mcp.NewToolResultError(fmt.Sprintf("note already exists: %s", path)), nil
	}

	data := []byte(content)
	if err := s.store.Write(path, data); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Index the new note.
	res, _ := parser.Parse(data)
	if res != nil {
		tags := res.Tags
		if tags == nil {
			tags = []string{}
		}
		_ = s.db.UpsertNote(index.NoteRow{
			Path:     path,
			Title:    res.Title,
			Checksum: "",
			Tags:     tags,
		}, res.Body, res.Links)
	}

	return mcp.NewToolResultText(fmt.Sprintf("created: %s", path)), nil
}

func (s *Server) listNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	folder := ""
	if f, err := req.RequireString("folder"); err == nil {
		folder = f
	}

	metas, err := s.store.List(folder)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var paths []string
	for _, m := range metas {
		paths = append(paths, m.Path)
	}
	return mcp.NewToolResultText(strings.Join(paths, "\n")), nil
}

func (s *Server) getNoteContract(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(NoteFormatContract), nil
}

func (s *Server) readNoteFormatResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "kenaz://note-format",
			MIMEType: "text/markdown",
			Text:     NoteFormatContract,
		},
	}, nil
}

func (s *Server) getBacklinks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	bl, err := s.db.Backlinks(path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(bl) == 0 {
		return mcp.NewToolResultText("no backlinks found"), nil
	}
	return mcp.NewToolResultText(strings.Join(bl, "\n")), nil
}

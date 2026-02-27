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

	"github.com/starford/kenaz/internal/noteservice"
	"github.com/starford/kenaz/internal/storage"
)

// Server wraps the MCP server with Kenaz tools.
type Server struct {
	mcp   *server.MCPServer
	svc   *noteservice.Service
	store storage.Provider
}

// New creates a new MCP server with all Kenaz tools registered.
func New(svc *noteservice.Service, store storage.Provider) *Server {
	s := &Server{svc: svc, store: store}

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
			"optional tags, Markdown body with [[wikilinks]]). "+
			"Language policy: file/directory names must be in English; frontmatter values and body content may use any language. "+
			"Read the contract first via the get_note_contract tool or the kenaz://note-format resource."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path for the new note (must end with .md)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown content following the Kenaz note format contract")),
	), s.createNote)

	s.mcp.AddTool(mcp.NewTool("update_note",
		mcp.WithDescription("Update an existing Markdown note at the specified path. "+
			"Content MUST follow the canonical note format. "+
			"Language policy: file/directory names must be in English; frontmatter values and body content may use any language. "+
			"Optionally provide a checksum for optimistic concurrency (SHA-256 of current content)."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the note")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Updated Markdown content")),
		mcp.WithString("checksum", mcp.Description("SHA-256 checksum of the current content for conflict detection")),
	), s.updateNote)

	s.mcp.AddTool(mcp.NewTool("delete_note",
		mcp.WithDescription("Delete an existing note at the specified path."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative path to the note to delete")),
	), s.deleteNote)

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

	s.mcp.AddTool(mcp.NewTool("upload_asset",
		mcp.WithDescription("Download a file from a URL or base64 data URI and save it as an attachment. "+
			"The file is stored in the shared attachments/ directory. "+
			"Returns savedPath and markdownImage ready to paste into a note. "+
			"Supported formats: png, jpg, jpeg, gif, webp, svg, pdf. Max size: 10 MB."),
		mcp.WithString("url", mcp.Required(), mcp.Description("HTTP/HTTPS URL or base64 data URI (e.g. data:image/png;base64,...)")),
		mcp.WithString("filename", mcp.Description("Optional filename; if omitted, extracted from URL or generated as UUID")),
	), s.uploadAsset)

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
	results, err := s.svc.Search(ctx, query, 20)
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
	note, err := s.svc.GetNote(ctx, path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("not found: %s", path)), nil
	}
	return mcp.NewToolResultText(note.Content), nil
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

	if _, err := s.svc.CreateNote(ctx, path, []byte(content)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("created: %s", path)), nil
}

func (s *Server) updateNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := req.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	cs := ""
	if v, csErr := req.RequireString("checksum"); csErr == nil {
		cs = v
	}

	if _, err := s.svc.UpdateNote(ctx, path, []byte(content), cs); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("updated: %s", path)), nil
}

func (s *Server) deleteNote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.svc.DeleteNote(ctx, path); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("not found: %s", path)), nil //nolint:nilerr
	}
	return mcp.NewToolResultText(fmt.Sprintf("deleted: %s", path)), nil
}

func (s *Server) listNotes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, _, err := s.svc.ListNotes(ctx, 1000, 0, "", "path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	folder := ""
	if f, fErr := req.RequireString("folder"); fErr == nil {
		folder = f
	}

	var paths []string
	for _, item := range items {
		if folder == "" || strings.HasPrefix(item.Path, folder) {
			paths = append(paths, item.Path)
		}
	}
	return mcp.NewToolResultText(strings.Join(paths, "\n")), nil
}

func (s *Server) getNoteContract(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(NoteFormatContract), nil
}

func (s *Server) readNoteFormatResource(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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
	bl, err := s.svc.Backlinks(ctx, path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(bl) == 0 {
		return mcp.NewToolResultText("no backlinks found"), nil
	}
	return mcp.NewToolResultText(strings.Join(bl, "\n")), nil
}

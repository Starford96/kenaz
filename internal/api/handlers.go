package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starford/kenaz/internal/apperr"
	"github.com/starford/kenaz/internal/noteservice"
)

// Handler holds API route handlers.
type Handler struct {
	svc *noteservice.Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *noteservice.Service) *Handler {
	return &Handler{svc: svc}
}

// notePath extracts the note path from the URL (everything after /api/notes/).
// Supports encoded slashes from OpenAPI clients (e.g. topics%2Fnote.md).
func notePath(r *http.Request) string {
	raw := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if raw == "" {
		return ""
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// ListNotes handles GET /api/notes.
//
//	@Summary		List notes with optional pagination and filtering
//	@Tags			notes
//	@Produce		json
//	@Param			limit	query		int		false	"Page size"
//	@Param			offset	query		int		false	"Page offset"
//	@Param			tag		query		string	false	"Filter by tag"
//	@Param			sort	query		string	false	"Sort field"	Enums(updated_at, title, path)
//	@Success		200		{object}	NoteListResponse
//	@Security		BearerAuth
//	@Router			/notes [get]
func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	tag := q.Get("tag")
	sort := q.Get("sort")

	items, total, err := h.svc.ListNotes(r.Context(), limit, offset, tag, sort)
	if err != nil {
		slog.Error("list notes failed", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notes": items,
		"total": total,
	})
}

// GetNote handles GET /api/notes/*.
//
//	@Summary		Get a single note by path
//	@Tags			notes
//	@Produce		json
//	@Param			path	path		string	true	"Note path"
//	@Success		200		{object}	NoteDetail
//	@Failure		404		{object}	errResponse
//	@Security		BearerAuth
//	@Router			/notes/{path} [get]
func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	path := notePath(r)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path is required"))
		return
	}
	note, err := h.svc.GetNote(r.Context(), path)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, errorBody("not found"))
		} else {
			slog.Error("get note failed", slog.String("path", path), slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// CreateNote handles POST /api/notes.
//
//	@Summary		Create a new note
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateNoteRequest	true	"Note to create"
//	@Success		201		{object}	NoteDetail
//	@Failure		400		{object}	errResponse
//	@Failure		409		{object}	errResponse
//	@Security		BearerAuth
//	@Router			/notes [post]
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON body"))
		return
	}
	if req.Path == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path and content are required"))
		return
	}
	note, err := h.svc.CreateNote(r.Context(), req.Path, []byte(req.Content))
	if err != nil {
		if errors.Is(err, apperr.ErrAlreadyExists) {
			writeJSON(w, http.StatusConflict, errorBody("note already exists"))
		} else {
			slog.Error("create note failed", slog.String("path", req.Path), slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		}
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

// UpdateNote handles PUT /api/notes/*.
//
//	@Summary		Update a note with optimistic concurrency
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Param			path	path		string				true	"Note path"
//	@Param			If-Match	header	string				false	"SHA-256 checksum for optimistic concurrency"
//	@Param			body	body		UpdateNoteRequest	true	"Updated content"
//	@Success		200		{object}	NoteDetail
//	@Failure		400		{object}	errResponse
//	@Failure		404		{object}	errResponse
//	@Failure		409		{object}	errResponse
//	@Security		BearerAuth
//	@Router			/notes/{path} [put]
func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	path := notePath(r)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path is required"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("failed to read body"))
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON body"))
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("content is required"))
		return
	}

	ifMatch := r.Header.Get("If-Match")
	// Strip surrounding quotes if present (standard ETag format).
	ifMatch = strings.Trim(ifMatch, `"`)

	note, err := h.svc.UpdateNote(r.Context(), path, []byte(req.Content), ifMatch)
	if err != nil {
		switch {
		case errors.Is(err, apperr.ErrNotFound):
			writeJSON(w, http.StatusNotFound, errorBody("not found"))
		case errors.Is(err, apperr.ErrConflict):
			writeJSON(w, http.StatusConflict, errorBody("checksum mismatch"))
		default:
			slog.Error("update note failed", slog.String("path", path), slog.String("error", err.Error()))
			writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		}
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// DeleteNote handles DELETE /api/notes/*.
//
//	@Summary		Delete a note
//	@Tags			notes
//	@Param			path	path	string	true	"Note path"
//	@Success		204		"Note deleted"
//	@Failure		404		{object}	errResponse
//	@Security		BearerAuth
//	@Router			/notes/{path} [delete]
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	path := notePath(r)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path is required"))
		return
	}
	if err := h.svc.DeleteNote(r.Context(), path); err != nil {
		slog.Error("delete note failed", slog.String("path", path), slog.String("error", err.Error()))
		writeJSON(w, http.StatusNotFound, errorBody("not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Search handles GET /api/search.
//
//	@Summary		Full-text search across notes
//	@Tags			search
//	@Produce		json
//	@Param			q		query		string	true	"Search query"
//	@Param			limit	query		int		false	"Max results"
//	@Success		200		{object}	SearchResponse
//	@Failure		400		{object}	errResponse
//	@Security		BearerAuth
//	@Router			/search [get]
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("query parameter 'q' is required"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.svc.Search(r.Context(), q, limit)
	if err != nil {
		slog.Error("search failed", slog.String("query", q), slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"results": results,
	})
}

// Graph handles GET /api/graph.
//
//	@Summary		Get the knowledge graph
//	@Tags			graph
//	@Produce		json
//	@Success		200	{object}	GraphResponse
//	@Security		BearerAuth
//	@Router			/graph [get]
func (h *Handler) Graph(w http.ResponseWriter, r *http.Request) {
	nodes, links, err := h.svc.Graph(r.Context())
	if err != nil {
		slog.Error("graph failed", slog.String("error", err.Error()))
		writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": nodes,
		"links": links,
	})
}

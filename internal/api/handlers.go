package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handler holds API route handlers.
type Handler struct {
	svc *Service
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// notePath extracts the note path from the URL (everything after /api/notes/).
func notePath(r *http.Request) string {
	return strings.TrimPrefix(chi.URLParam(r, "*"), "/")
}

// ListNotes handles GET /api/notes.
func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	tag := q.Get("tag")
	sort := q.Get("sort")

	items, total, err := h.svc.ListNotes(limit, offset, tag, sort)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notes": items,
		"total": total,
	})
}

// GetNote handles GET /api/notes/*.
func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	path := notePath(r)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path is required"))
		return
	}
	note, err := h.svc.GetNote(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorBody("not found"))
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// CreateNote handles POST /api/notes.
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
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
	note, err := h.svc.CreateNote(req.Path, []byte(req.Content))
	if err != nil {
		if err.Error() == "already exists" {
			writeJSON(w, http.StatusConflict, errorBody("note already exists"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, note)
}

// UpdateNote handles PUT /api/notes/*.
func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
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

	note, err := h.svc.UpdateNote(path, []byte(req.Content), ifMatch)
	if err != nil {
		switch err.Error() {
		case "not found":
			writeJSON(w, http.StatusNotFound, errorBody("not found"))
		case "conflict":
			writeJSON(w, http.StatusConflict, errorBody("checksum mismatch"))
		default:
			writeJSON(w, http.StatusInternalServerError, errorBody(err.Error()))
		}
		return
	}
	writeJSON(w, http.StatusOK, note)
}

// DeleteNote handles DELETE /api/notes/*.
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	path := notePath(r)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("path is required"))
		return
	}
	if err := h.svc.DeleteNote(path); err != nil {
		writeJSON(w, http.StatusNotFound, errorBody("not found"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Search handles GET /api/search.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, http.StatusBadRequest, errorBody("query parameter 'q' is required"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.svc.Search(q, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// Graph handles GET /api/graph.
func (h *Handler) Graph(w http.ResponseWriter, _ *http.Request) {
	nodes, links, err := h.svc.Graph()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"links": links,
	})
}

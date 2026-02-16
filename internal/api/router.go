package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter creates a chi router with all API routes mounted.
// authEnabled controls whether Bearer token auth is enforced.
// sseHandler, if non-nil, is mounted at GET /events inside the auth group.
func NewRouter(svc *Service, authEnabled bool, token string, sseHandler http.Handler) chi.Router {
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(AuthMiddleware(authEnabled, token))

	// Notes CRUD.
	r.Get("/notes", h.ListNotes)
	r.Post("/notes", h.CreateNote)
	r.Get("/notes/*", h.GetNote)
	r.Put("/notes/*", h.UpdateNote)
	r.Delete("/notes/*", h.DeleteNote)

	// Search.
	r.Get("/search", h.Search)

	// Graph.
	r.Get("/graph", h.Graph)

	// SSE endpoint (protected by same auth middleware).
	if sseHandler != nil {
		r.Get("/events", sseHandler.ServeHTTP)
	}

	return r
}

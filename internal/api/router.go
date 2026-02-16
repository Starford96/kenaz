package api

import (
	"github.com/go-chi/chi/v5"
)

// NewRouter creates a chi router with all API routes mounted.
// The token parameter controls Bearer auth; empty string disables authentication.
func NewRouter(svc *Service, token string) chi.Router {
	h := NewHandler(svc)

	r := chi.NewRouter()
	r.Use(AuthMiddleware(token))

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

	return r
}

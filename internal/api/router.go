package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starford/kenaz/internal/noteservice"
)

// NewRouter creates a chi router with all API routes mounted.
// authEnabled controls whether Bearer token auth is enforced.
// sseHandler, if non-nil, is mounted at GET /events inside the auth group.
// vaultRoot is used to resolve the attachments directory.
func NewRouter(svc *noteservice.Service, authEnabled bool, token string, sseHandler http.Handler, vaultRoot string) chi.Router {
	h := NewHandler(svc)
	ah := NewAttachmentHandler(vaultRoot)

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

	// Attachments upload (auth-protected).
	r.Post("/attachments", ah.Upload)

	// SSE endpoint (protected by same auth middleware).
	if sseHandler != nil {
		r.Get("/events", sseHandler.ServeHTTP)
	}

	return r
}

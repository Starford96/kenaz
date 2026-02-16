// Package api implements the Kenaz REST API using chi.
package api

import (
	"net/http"
	"strings"
)

// AuthMiddleware returns middleware that validates a Bearer token.
// If token is empty, authentication is disabled (all requests pass).
func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
				writeJSON(w, http.StatusUnauthorized, errorBody("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

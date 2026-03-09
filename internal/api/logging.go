package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// SlogRequestLogger is an HTTP middleware that logs requests using slog,
// producing structured JSON output consistent with the application logger.
func SlogRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		status := ww.Status()
		lvl := slog.LevelInfo
		if status >= 500 {
			lvl = slog.LevelWarn
		}

		slog.Log(r.Context(), lvl, "http request",
			slog.String("method", r.Method),
			slog.String("path", r.RequestURI),
			slog.Int("status", status),
			slog.Int("bytes", ww.BytesWritten()),
			slog.String("duration", time.Since(start).String()),
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

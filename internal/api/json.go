package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("json encode failed", slog.String("error", err.Error()))
	}
}

type errResponse struct {
	Error string `json:"error" validate:"required"`
}

func errorBody(msg string) errResponse {
	return errResponse{Error: msg}
}

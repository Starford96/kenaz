package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

const (
	attachDir      = "attachments"
	maxUploadBytes = 50 << 20 // 50 MB
)

// AttachmentHandler serves and accepts attachment files.
type AttachmentHandler struct {
	vaultRoot string
}

// NewAttachmentHandler creates a handler rooted at the vault directory.
func NewAttachmentHandler(vaultRoot string) *AttachmentHandler {
	return &AttachmentHandler{vaultRoot: vaultRoot}
}

// attachDir returns the absolute path to the attachments directory.
func (h *AttachmentHandler) attachPath() string {
	return filepath.Join(h.vaultRoot, attachDir)
}

// safeName validates that the filename is a plain name (no path separators,
// no traversal) and returns the absolute path under the attachments dir.
func (h *AttachmentHandler) safeName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("filename is required")
	}
	// Reject anything with path separators or traversal.
	cleaned := filepath.Clean(name)
	if cleaned != filepath.Base(cleaned) || strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid filename: %s", name)
	}
	abs := filepath.Join(h.attachPath(), cleaned)
	// Double-check the resolved path is under attachments dir.
	if !strings.HasPrefix(abs, h.attachPath()+string(os.PathSeparator)) && abs != h.attachPath() {
		return "", fmt.Errorf("path escapes attachments directory")
	}
	return abs, nil
}

// ServeFile handles GET /attachments/{filename}.
func (h *AttachmentHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	abs, err := h.safeName(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, statErr := os.Stat(abs); os.IsNotExist(statErr) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, abs)
}

// Upload handles POST /api/attachments (multipart/form-data, field "file").
func (h *AttachmentHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("file too large or invalid multipart"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("missing 'file' field in multipart form"))
		return
	}
	defer file.Close()

	abs, err := h.safeName(header.Filename)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}

	// Ensure attachments directory exists.
	if err := os.MkdirAll(h.attachPath(), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody("failed to create attachments dir"))
		return
	}

	dst, err := os.Create(abs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody("failed to create file"))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody("failed to write file"))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"filename": header.Filename,
		"size":     written,
		"url":      "/attachments/" + header.Filename,
	})
}

package mcpserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

const maxAssetSize = 10 << 20 // 10 MB

var (
	allowedExtensions = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".webp": true, ".svg": true, ".pdf": true,
	}

	mimeToExt = map[string]string{
		"image/png":     ".png",
		"image/jpeg":    ".jpg",
		"image/gif":     ".gif",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"application/pdf": ".pdf",
	}

	safeFilenameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)
)

type uploadResult struct {
	SavedPath     string `json:"savedPath"`
	MarkdownImage string `json:"markdownImage"`
}

func (s *Server) uploadAsset(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rawURL, err := req.RequireString("url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	filename := ""
	if v, fErr := req.RequireString("filename"); fErr == nil {
		filename = v
	}

	var data []byte
	var detectedExt string

	if strings.HasPrefix(rawURL, "data:") {
		data, detectedExt, err = decodeDataURI(rawURL)
	} else {
		data, detectedExt, err = fetchHTTP(rawURL)
	}
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(data) > maxAssetSize {
		return mcp.NewToolResultError(fmt.Sprintf("file too large: %d bytes (max %d)", len(data), maxAssetSize)), nil
	}

	if filename == "" {
		filename = filenameFromURL(rawURL, detectedExt)
	}
	filename = sanitizeFilename(filename)

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExtensions[ext] {
		return mcp.NewToolResultError(fmt.Sprintf("unsupported file extension: %s (allowed: png, jpg, jpeg, gif, webp, svg, pdf)", ext)), nil
	}

	if err := validateMagicBytes(data, ext); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	savePath := filepath.Join("attachments", filename)

	if _, readErr := s.store.Read(savePath); readErr == nil {
		return mcp.NewToolResultError(fmt.Sprintf("file already exists: %s", savePath)), nil
	}

	if err := s.store.Write(savePath, data); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save attachment: %v", err)), nil
	}

	urlPath := "/attachments/" + filename
	out, _ := json.Marshal(uploadResult{
		SavedPath:     urlPath,
		MarkdownImage: fmt.Sprintf("![%s](%s)", filename, urlPath),
	})
	return mcp.NewToolResultText(string(out)), nil
}

// decodeDataURI parses a data:[<mediatype>][;base64],<data> URI.
func decodeDataURI(uri string) ([]byte, string, error) {
	rest := strings.TrimPrefix(uri, "data:")
	commaIdx := strings.Index(rest, ",")
	if commaIdx < 0 {
		return nil, "", fmt.Errorf("invalid data URI: missing comma separator")
	}

	meta := rest[:commaIdx]
	encoded := rest[commaIdx+1:]

	if !strings.Contains(meta, ";base64") {
		return nil, "", fmt.Errorf("only base64 data URIs are supported")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		data, err = base64.RawStdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, "", fmt.Errorf("invalid base64 data: %w", err)
		}
	}

	mime := strings.Split(strings.TrimSuffix(meta, ";base64"), ";")[0]
	ext := mimeToExt[mime]
	if ext == "" {
		return nil, "", fmt.Errorf("unsupported MIME type in data URI: %s", mime)
	}
	return data, ext, nil
}

// fetchHTTP downloads a file from an HTTP/HTTPS URL with security checks.
func fetchHTTP(rawURL string) ([]byte, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, "", fmt.Errorf("unsupported scheme: %s (only http/https)", parsed.Scheme)
	}

	if err := checkBlockedHost(parsed.Hostname()); err != nil {
		return nil, "", err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects (max 5)")
			}
			return checkBlockedHost(req.URL.Hostname())
		},
	}

	resp, err := client.Get(rawURL) //nolint:noctx
	if err != nil {
		return nil, "", fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxAssetSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("read body failed: %w", err)
	}
	if len(data) > maxAssetSize {
		return nil, "", fmt.Errorf("file too large: exceeds %d bytes", maxAssetSize)
	}

	ct := resp.Header.Get("Content-Type")
	ext := mimeToExt[strings.Split(ct, ";")[0]]
	return data, ext, nil
}

// checkBlockedHost rejects loopback and cloud metadata addresses.
func checkBlockedHost(host string) error {
	if host == "metadata.google.internal" {
		return fmt.Errorf("blocked host: %s", host)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		ips, lookupErr := net.LookupIP(host)
		if lookupErr != nil || len(ips) == 0 {
			return nil //nolint:nilerr // let http.Client handle DNS failures
		}
		ip = ips[0]
	}

	if ip.IsLoopback() {
		return fmt.Errorf("blocked host: loopback address %s", host)
	}
	// AWS/GCP/Azure metadata endpoint.
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return fmt.Errorf("blocked host: cloud metadata address %s", host)
	}
	return nil
}

// filenameFromURL tries to extract a filename from a URL, falling back to UUID.
func filenameFromURL(rawURL string, fallbackExt string) string {
	if strings.HasPrefix(rawURL, "data:") {
		ext := fallbackExt
		if ext == "" {
			ext = ".bin"
		}
		return uuid.New().String() + ext
	}

	parsed, err := url.Parse(rawURL)
	if err == nil {
		base := path.Base(parsed.Path)
		if base != "" && base != "." && base != "/" && strings.Contains(base, ".") {
			return base
		}
	}

	ext := fallbackExt
	if ext == "" {
		ext = ".bin"
	}
	return uuid.New().String() + ext
}

// sanitizeFilename strips path separators and unsafe characters.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = safeFilenameRe.ReplaceAllString(name, "_")
	if name == "" || name == "." {
		name = uuid.New().String()
	}
	return name
}

// validateMagicBytes verifies file content matches the declared extension.
func validateMagicBytes(data []byte, ext string) error {
	if ext == ".svg" {
		prefix := data
		if len(prefix) > 1024 {
			prefix = prefix[:1024]
		}
		if !bytes.Contains(prefix, []byte("<svg")) {
			return fmt.Errorf("content does not appear to be a valid SVG (missing <svg tag)")
		}
		return nil
	}

	detected := http.DetectContentType(data)
	expectedExts := mimeToExt[strings.Split(detected, ";")[0]]

	switch ext {
	case ".jpg", ".jpeg":
		if expectedExts != ".jpg" && expectedExts != ".jpeg" {
			return fmt.Errorf("content does not match extension %s (detected: %s)", ext, detected)
		}
	default:
		if expectedExts != ext {
			return fmt.Errorf("content does not match extension %s (detected: %s)", ext, detected)
		}
	}
	return nil
}

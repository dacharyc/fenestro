package main

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LocalFileHandler serves files from the local filesystem for relative paths
// It intercepts requests to /localfile/* and serves them from the current file's directory
type LocalFileHandler struct {
	app *App
}

// NewLocalFileHandler creates a new handler for serving local files
func NewLocalFileHandler(app *App) *LocalFileHandler {
	return &LocalFileHandler{app: app}
}

// ServeHTTP handles requests for local files
func (h *LocalFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only handle GET requests to /localfile/*
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	if !strings.HasPrefix(path, "/localfile/") {
		http.NotFound(w, r)
		return
	}

	// Get the relative path (everything after /localfile/)
	relativePath := strings.TrimPrefix(path, "/localfile/")
	if relativePath == "" {
		http.NotFound(w, r)
		return
	}

	// Get the base path from the current file
	basePath := h.app.GetCurrentBasePath()
	if basePath == "" {
		// No base path (stdin content), can't serve local files
		http.NotFound(w, r)
		return
	}

	// Construct the full file path
	fullPath := filepath.Join(basePath, relativePath)

	// Security check: ensure the resolved path is within the base directory
	// This prevents directory traversal attacks (e.g., ../../../etc/passwd)
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Don't serve directories
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Open and serve the file
	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Set content type based on file extension
	ext := filepath.Ext(fullPath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		// Default to octet-stream for unknown types
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)

	// Copy the file content to the response
	io.Copy(w, file)
}

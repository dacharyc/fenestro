package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalFileHandler_ServeHTTP(t *testing.T) {
	// Create a temp directory with test files
	tmpDir, err := os.MkdirTemp("", "fenestro-handler-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	cssDir := filepath.Join(tmpDir, "assets", "css")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		t.Fatal(err)
	}
	cssContent := "body { color: red; }"
	if err := os.WriteFile(filepath.Join(cssDir, "style.css"), []byte(cssContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create app with a file in the temp directory
	app := NewApp(FileEntry{
		Name:    "test.html",
		Path:    filepath.Join(tmpDir, "test.html"),
		Content: "<html></html>",
	}, "")

	handler := NewLocalFileHandler(app)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "serves CSS file",
			path:           "/localfile/assets/css/style.css",
			expectedStatus: http.StatusOK,
			expectedBody:   cssContent,
		},
		{
			name:           "returns 404 for non-existent file",
			path:           "/localfile/nonexistent.css",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "returns 404 for non-localfile paths",
			path:           "/other/path",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "returns 404 for empty relative path",
			path:           "/localfile/",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && w.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestLocalFileHandler_DirectoryTraversal(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "fenestro-traversal-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a "secret" file outside the html directory
	secretDir := filepath.Join(tmpDir, "secret")
	if err := os.MkdirAll(secretDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "password.txt"), []byte("secret123"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create HTML directory
	htmlDir := filepath.Join(tmpDir, "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create app with a file in the html directory
	app := NewApp(FileEntry{
		Name:    "test.html",
		Path:    filepath.Join(htmlDir, "test.html"),
		Content: "<html></html>",
	}, "")

	handler := NewLocalFileHandler(app)

	// Try to access the secret file via directory traversal
	req := httptest.NewRequest(http.MethodGet, "/localfile/../secret/password.txt", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for directory traversal, got %d", http.StatusForbidden, w.Code)
	}
}

func TestLocalFileHandler_NoBasePath(t *testing.T) {
	// Create app with stdin content (no path)
	app := NewApp(FileEntry{
		Name:    "stdin",
		Path:    "", // stdin has no path
		Content: "<html></html>",
	}, "")

	handler := NewLocalFileHandler(app)

	req := httptest.NewRequest(http.MethodGet, "/localfile/style.css", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d when no base path, got %d", http.StatusNotFound, w.Code)
	}
}

func TestLocalFileHandler_MethodNotAllowed(t *testing.T) {
	app := NewApp(FileEntry{
		Name:    "test.html",
		Path:    "/tmp/test.html",
		Content: "<html></html>",
	}, "")

	handler := NewLocalFileHandler(app)

	req := httptest.NewRequest(http.MethodPost, "/localfile/style.css", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status %d for POST, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestLocalFileHandler_ContentType(t *testing.T) {
	// Create a temp directory with test files
	tmpDir, err := os.MkdirTemp("", "fenestro-content-type-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with different extensions
	files := map[string]string{
		"style.css":  ".test { }",
		"script.js":  "console.log('test');",
		"image.svg":  "<svg></svg>",
		"data.json":  "{}",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	app := NewApp(FileEntry{
		Name:    "test.html",
		Path:    filepath.Join(tmpDir, "test.html"),
		Content: "<html></html>",
	}, "")

	handler := NewLocalFileHandler(app)

	tests := []struct {
		file        string
		contentType string
	}{
		{"style.css", "text/css"},
		{"script.js", "text/javascript"},
		{"image.svg", "image/svg+xml"},
		{"data.json", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/localfile/"+tt.file, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			// Content-Type may include charset parameter (e.g., "text/css; charset=utf-8")
			if !strings.HasPrefix(contentType, tt.contentType) {
				t.Errorf("Expected Content-Type to start with %q, got %q", tt.contentType, contentType)
			}
		})
	}
}

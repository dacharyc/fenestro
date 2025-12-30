package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct holds the application state
type App struct {
	ctx          context.Context
	files        []FileEntry
	currentIndex int
	windowID     string
	config       Config
	mu           sync.RWMutex
	// Initial window position to set on startup (if shouldSetPosition is true)
	initialX          int
	initialY          int
	initialWidth      int
	initialHeight     int
	shouldSetPosition bool
	// Cached geometry to avoid redundant saves
	lastSavedGeometry WindowState
}

// NewApp creates a new App with the given initial file
func NewApp(file FileEntry, windowID string) *App {
	return &App{
		files:        []FileEntry{file},
		currentIndex: 0,
		windowID:     windowID,
		config:       LoadConfig(),
	}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Set window position if we have saved state or config defaults
	if a.shouldSetPosition {
		ValidateAndSetWindowPosition(ctx, a.initialX, a.initialY, a.initialWidth, a.initialHeight)
	}
}

// macOS title bar height in pixels
// This is needed because Wails Width/Height options set content size,
// but WindowGetSize returns frame size (including title bar)
const macOSTitleBarHeight = 28

// GetWindowGeometry returns the current window geometry for saving
func (a *App) GetWindowGeometry() WindowState {
	if a.ctx == nil {
		return WindowState{}
	}
	w, h := runtime.WindowGetSize(a.ctx)
	x, y := runtime.WindowGetPosition(a.ctx)

	// Subtract title bar height since Wails options expect content height
	// but WindowGetSize returns frame height
	contentHeight := h - macOSTitleBarHeight
	if contentHeight < MinWindowHeight {
		contentHeight = MinWindowHeight
	}

	return WindowState{
		Width:  w,
		Height: contentHeight,
		X:      x,
		Y:      y,
	}
}

// SaveWindowGeometry saves the current window geometry if it has changed.
// Called from frontend when window is moved or resized.
func (a *App) SaveWindowGeometry() {
	geometry := a.GetWindowGeometry()
	if !geometry.IsValid() {
		return
	}

	// Only save if geometry has changed
	if geometry.Width == a.lastSavedGeometry.Width &&
		geometry.Height == a.lastSavedGeometry.Height &&
		geometry.X == a.lastSavedGeometry.X &&
		geometry.Y == a.lastSavedGeometry.Y {
		return
	}

	if err := SaveWindowState(geometry); err == nil {
		a.lastSavedGeometry = geometry
	}
}

// GetHTMLContent returns the HTML content of the currently selected file
// This is called from the frontend to get the initial content
func (a *App) GetHTMLContent() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.files) == 0 || a.currentIndex < 0 || a.currentIndex >= len(a.files) {
		return ""
	}
	return a.files[a.currentIndex].Content
}

// GetCurrentBasePath returns the directory containing the current file
// Used by frontend to set <base> tag for resolving relative URLs
// Returns empty string for stdin content (no file path)
func (a *App) GetCurrentBasePath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.files) == 0 || a.currentIndex < 0 || a.currentIndex >= len(a.files) {
		return ""
	}
	path := a.files[a.currentIndex].Path
	if path == "" {
		return ""
	}
	// Return the directory containing the file
	dir := filepath.Dir(path)
	return dir
}

// GetFiles returns all files for the sidebar
func (a *App) GetFiles() []FileEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()
	// Return a copy to avoid race conditions
	result := make([]FileEntry, len(a.files))
	copy(result, a.files)
	return result
}

// GetCurrentIndex returns the index of the currently selected file
func (a *App) GetCurrentIndex() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentIndex
}

// SelectFile switches to the file at the given index and returns its content
func (a *App) SelectFile(index int) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.files) {
		return ""
	}
	a.currentIndex = index
	return a.files[index].Content
}

// AddFile adds a new file to the sidebar and emits an event to the frontend
func (a *App) AddFile(entry FileEntry) {
	a.mu.Lock()
	a.files = append(a.files, entry)
	sortFilesByName(a.files)
	// Find the new index after sorting
	newIndex := 0
	for i, f := range a.files {
		if f.Path == entry.Path && f.Name == entry.Name {
			newIndex = i
			break
		}
	}
	// Copy files while holding the lock to avoid race condition
	filesCopy := make([]FileEntry, len(a.files))
	copy(filesCopy, a.files)
	a.mu.Unlock()

	// Emit event to frontend
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "file-added", map[string]interface{}{
			"files": filesCopy,
			"index": newIndex,
		})
	}
}

// ReplaceFileContent replaces the content of a file by path, selects it, and emits an event
// If the path is not found, adds it as a new file
func (a *App) ReplaceFileContent(path, content, name string) {
	a.mu.Lock()
	found := false
	for i, f := range a.files {
		if f.Path == path {
			a.files[i].Content = content
			if name != "" {
				a.files[i].Name = name
			}
			a.currentIndex = i
			found = true
			break
		}
	}
	if !found {
		// Add as new file
		a.files = append(a.files, FileEntry{
			Name:    name,
			Path:    path,
			Content: content,
		})
		sortFilesByName(a.files)
		// Find index after sorting
		for i, f := range a.files {
			if f.Path == path {
				a.currentIndex = i
				break
			}
		}
	}
	// Copy data while holding the lock to avoid race condition
	filesCopy := make([]FileEntry, len(a.files))
	copy(filesCopy, a.files)
	currentIndex := a.currentIndex
	a.mu.Unlock()

	// Emit event to frontend
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "content-replaced", map[string]interface{}{
			"files":        filesCopy,
			"currentIndex": currentIndex,
		})
	}
}

// GetWindowID returns the window ID
func (a *App) GetWindowID() string {
	return a.windowID
}

// GetConfig returns the application configuration
func (a *App) GetConfig() Config {
	return a.config
}

// GetChromeCSS returns the content of the custom chrome CSS file
// Returns empty string if no file is configured or file can't be read
func (a *App) GetChromeCSS() string {
	if a.config.ChromeCSS == "" {
		return ""
	}
	content, err := os.ReadFile(a.config.ChromeCSS)
	if err != nil {
		return ""
	}
	return string(content)
}

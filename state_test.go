package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowStateIsValid(t *testing.T) {
	tests := []struct {
		name     string
		state    *WindowState
		expected bool
	}{
		{"nil state", nil, false},
		{"zero dimensions", &WindowState{Width: 0, Height: 0}, false},
		{"zero width", &WindowState{Width: 0, Height: 700}, false},
		{"zero height", &WindowState{Width: 900, Height: 0}, false},
		{"negative width", &WindowState{Width: -100, Height: 700}, false},
		{"negative height", &WindowState{Width: 900, Height: -100}, false},
		{"valid state", &WindowState{Width: 900, Height: 700}, true},
		{"valid with position", &WindowState{Width: 900, Height: 700, X: 100, Y: 100}, true},
		{"valid with negative position", &WindowState{Width: 900, Height: 700, X: -50, Y: -50}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLoadWindowStateNoFile(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	// Point to a directory that doesn't exist
	os.Setenv("XDG_CONFIG_HOME", "/nonexistent/path")
	state := LoadWindowState()

	if state != nil {
		t.Errorf("Expected nil state when file doesn't exist, got %+v", state)
	}
}

func TestSaveAndLoadWindowState(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-state-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Save state
	state := WindowState{
		Width:  1200,
		Height: 800,
		X:      150,
		Y:      75,
	}
	err = SaveWindowState(state)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify file was created
	statePath := filepath.Join(tmpDir, "fenestro", "state.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created at %s", statePath)
	}

	// Load state
	loaded := LoadWindowState()
	if loaded == nil {
		t.Fatalf("LoadWindowState returned nil")
	}

	if loaded.Width != state.Width {
		t.Errorf("Width = %d, expected %d", loaded.Width, state.Width)
	}
	if loaded.Height != state.Height {
		t.Errorf("Height = %d, expected %d", loaded.Height, state.Height)
	}
	if loaded.X != state.X {
		t.Errorf("X = %d, expected %d", loaded.X, state.X)
	}
	if loaded.Y != state.Y {
		t.Errorf("Y = %d, expected %d", loaded.Y, state.Y)
	}
}

func TestSaveWindowStateInvalid(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-state-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Try to save invalid state
	state := WindowState{Width: 0, Height: 0}
	err = SaveWindowState(state)
	if err != nil {
		t.Errorf("SaveWindowState should not return error for invalid state, got: %v", err)
	}

	// File should not exist
	statePath := filepath.Join(tmpDir, "fenestro", "state.json")
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Errorf("State file should not be created for invalid state")
	}
}

func TestLoadWindowStateInvalidJSON(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-state-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create state directory and file with invalid JSON
	stateDir := filepath.Join(tmpDir, "fenestro")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Could not create state dir: %v", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, []byte("invalid json {{{"), 0644); err != nil {
		t.Fatalf("Could not write state file: %v", err)
	}

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	state := LoadWindowState()

	if state != nil {
		t.Errorf("Expected nil state on invalid JSON, got %+v", state)
	}
}

func TestLoadWindowStateInvalidValues(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-state-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create state directory and file with zero dimensions
	stateDir := filepath.Join(tmpDir, "fenestro")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("Could not create state dir: %v", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	content := `{"width": 0, "height": 0, "x": 100, "y": 100}`
	if err := os.WriteFile(statePath, []byte(content), 0644); err != nil {
		t.Fatalf("Could not write state file: %v", err)
	}

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	state := LoadWindowState()

	if state != nil {
		t.Errorf("Expected nil state when dimensions are zero, got %+v", state)
	}
}

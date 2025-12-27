package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WindowState holds the saved window geometry
type WindowState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	X      int `json:"x"`
	Y      int `json:"y"`
}

// IsValid returns true if the state has valid dimensions
func (s *WindowState) IsValid() bool {
	return s != nil && s.Width > 0 && s.Height > 0
}

// getStatePath returns the path to the state file
func getStatePath() string {
	configDir := getConfigDir()
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "state.json")
}

// LoadWindowState loads the window state from the state file
// Returns nil if no state exists or can't be read
func LoadWindowState() *WindowState {
	statePath := getStatePath()
	if statePath == "" {
		return nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		// File doesn't exist or can't be read - that's fine
		return nil
	}

	var state WindowState
	if err := json.Unmarshal(data, &state); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse state file %s: %v\n", statePath, err)
		return nil
	}

	if !state.IsValid() {
		return nil
	}

	return &state
}

// SaveWindowState saves the window state to the state file
func SaveWindowState(state WindowState) error {
	if !state.IsValid() {
		return nil // Don't save invalid state
	}

	statePath := getStatePath()
	if statePath == "" {
		return fmt.Errorf("could not determine state file path")
	}

	// Ensure config directory exists
	configDir := filepath.Dir(statePath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

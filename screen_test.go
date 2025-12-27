package main

import (
	"testing"
)

func TestGetWindowDimensions(t *testing.T) {
	tests := []struct {
		name           string
		state          *WindowState
		config         Config
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "defaults when no state or config",
			state:          nil,
			config:         Config{},
			expectedWidth:  DefaultWindowWidth,
			expectedHeight: DefaultWindowHeight,
		},
		{
			name:           "config overrides defaults",
			state:          nil,
			config:         Config{DefaultWidth: 1200, DefaultHeight: 800},
			expectedWidth:  1200,
			expectedHeight: 800,
		},
		{
			name:           "state overrides config",
			state:          &WindowState{Width: 1400, Height: 900, X: 0, Y: 0},
			config:         Config{DefaultWidth: 1200, DefaultHeight: 800},
			expectedWidth:  1400,
			expectedHeight: 900,
		},
		{
			name:           "state overrides defaults",
			state:          &WindowState{Width: 1000, Height: 600, X: 0, Y: 0},
			config:         Config{},
			expectedWidth:  1000,
			expectedHeight: 600,
		},
		{
			name:           "partial config - only width",
			state:          nil,
			config:         Config{DefaultWidth: 1200},
			expectedWidth:  1200,
			expectedHeight: DefaultWindowHeight,
		},
		{
			name:           "partial config - only height",
			state:          nil,
			config:         Config{DefaultHeight: 800},
			expectedWidth:  DefaultWindowWidth,
			expectedHeight: 800,
		},
		{
			name:           "enforces minimum width",
			state:          &WindowState{Width: 100, Height: 600, X: 0, Y: 0},
			config:         Config{},
			expectedWidth:  MinWindowWidth,
			expectedHeight: 600,
		},
		{
			name:           "enforces minimum height",
			state:          &WindowState{Width: 900, Height: 100, X: 0, Y: 0},
			config:         Config{},
			expectedWidth:  900,
			expectedHeight: MinWindowHeight,
		},
		{
			name:           "config below minimum is enforced",
			state:          nil,
			config:         Config{DefaultWidth: 200, DefaultHeight: 150},
			expectedWidth:  MinWindowWidth,
			expectedHeight: MinWindowHeight,
		},
		{
			name:           "invalid state uses config",
			state:          &WindowState{Width: 0, Height: 0},
			config:         Config{DefaultWidth: 1200, DefaultHeight: 800},
			expectedWidth:  1200,
			expectedHeight: 800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := GetWindowDimensions(tt.state, tt.config)
			if width != tt.expectedWidth {
				t.Errorf("width = %d, expected %d", width, tt.expectedWidth)
			}
			if height != tt.expectedHeight {
				t.Errorf("height = %d, expected %d", height, tt.expectedHeight)
			}
		})
	}
}

func TestGetWindowPosition(t *testing.T) {
	tests := []struct {
		name         string
		state        *WindowState
		config       Config
		expectedX    int
		expectedY    int
		shouldSet    bool
	}{
		{
			name:      "no state or config - don't set",
			state:     nil,
			config:    Config{},
			expectedX: 0,
			expectedY: 0,
			shouldSet: false,
		},
		{
			name:      "config sets position",
			state:     nil,
			config:    Config{DefaultX: 100, DefaultY: 200},
			expectedX: 100,
			expectedY: 200,
			shouldSet: true,
		},
		{
			name:      "config with only X",
			state:     nil,
			config:    Config{DefaultX: 100},
			expectedX: 100,
			expectedY: 0,
			shouldSet: true,
		},
		{
			name:      "config with only Y",
			state:     nil,
			config:    Config{DefaultY: 200},
			expectedX: 0,
			expectedY: 200,
			shouldSet: true,
		},
		{
			name:      "state overrides config",
			state:     &WindowState{Width: 900, Height: 700, X: 300, Y: 400},
			config:    Config{DefaultX: 100, DefaultY: 200},
			expectedX: 300,
			expectedY: 400,
			shouldSet: true,
		},
		{
			name:      "state at origin still sets position",
			state:     &WindowState{Width: 900, Height: 700, X: 0, Y: 0},
			config:    Config{},
			expectedX: 0,
			expectedY: 0,
			shouldSet: true,
		},
		{
			name:      "invalid state uses config",
			state:     &WindowState{Width: 0, Height: 0, X: 300, Y: 400},
			config:    Config{DefaultX: 100, DefaultY: 200},
			expectedX: 100,
			expectedY: 200,
			shouldSet: true,
		},
		{
			name:      "invalid state no config - don't set",
			state:     &WindowState{Width: 0, Height: 0},
			config:    Config{},
			expectedX: 0,
			expectedY: 0,
			shouldSet: false,
		},
		{
			name:      "negative position from state",
			state:     &WindowState{Width: 900, Height: 700, X: -100, Y: -50},
			config:    Config{},
			expectedX: -100,
			expectedY: -50,
			shouldSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y, shouldSet := GetWindowPosition(tt.state, tt.config)
			if x != tt.expectedX {
				t.Errorf("x = %d, expected %d", x, tt.expectedX)
			}
			if y != tt.expectedY {
				t.Errorf("y = %d, expected %d", y, tt.expectedY)
			}
			if shouldSet != tt.shouldSet {
				t.Errorf("shouldSet = %v, expected %v", shouldSet, tt.shouldSet)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have reasonable values
	if DefaultWindowWidth < MinWindowWidth {
		t.Errorf("DefaultWindowWidth (%d) should be >= MinWindowWidth (%d)",
			DefaultWindowWidth, MinWindowWidth)
	}
	if DefaultWindowHeight < MinWindowHeight {
		t.Errorf("DefaultWindowHeight (%d) should be >= MinWindowHeight (%d)",
			DefaultWindowHeight, MinWindowHeight)
	}
	if MinWindowWidth <= 0 {
		t.Errorf("MinWindowWidth should be positive, got %d", MinWindowWidth)
	}
	if MinWindowHeight <= 0 {
		t.Errorf("MinWindowHeight should be positive, got %d", MinWindowHeight)
	}
}

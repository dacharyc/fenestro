package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// Default window dimensions
	DefaultWindowWidth  = 900
	DefaultWindowHeight = 700
	MinWindowWidth      = 400
	MinWindowHeight     = 300
)

// GetWindowDimensions returns the window width and height to use based on
// saved state, config defaults, and hardcoded defaults (in priority order)
func GetWindowDimensions(state *WindowState, config Config) (width, height int) {
	// Start with hardcoded defaults
	width = DefaultWindowWidth
	height = DefaultWindowHeight

	// Apply config defaults if specified
	if config.DefaultWidth > 0 {
		width = config.DefaultWidth
	}
	if config.DefaultHeight > 0 {
		height = config.DefaultHeight
	}

	// Apply saved state if valid (takes precedence)
	if state != nil && state.IsValid() {
		width = state.Width
		height = state.Height
	}

	// Ensure minimum dimensions
	if width < MinWindowWidth {
		width = MinWindowWidth
	}
	if height < MinWindowHeight {
		height = MinWindowHeight
	}

	return width, height
}

// GetWindowPosition returns the window X and Y position to use based on
// saved state and config defaults. Returns (0, 0, false) if no position
// should be explicitly set (let OS decide).
func GetWindowPosition(state *WindowState, config Config) (x, y int, shouldSet bool) {
	// Check saved state first
	if state != nil && state.IsValid() {
		return state.X, state.Y, true
	}

	// Check config defaults
	if config.DefaultX != 0 || config.DefaultY != 0 {
		return config.DefaultX, config.DefaultY, true
	}

	// Let OS decide position
	return 0, 0, false
}

// ValidateAndSetWindowPosition sets the window position.
// Validates that at least part of the window would be visible on the current
// screen setup. This handles the case where an external monitor was disconnected.
func ValidateAndSetWindowPosition(ctx context.Context, x, y, width, height int) {
	screens, err := runtime.ScreenGetAll(ctx)
	if err != nil || len(screens) == 0 {
		// Can't get screen info, just set the position
		runtime.WindowSetPosition(ctx, x, y)
		return
	}

	// Calculate total screen dimensions
	// For multi-monitor: screens can be arranged horizontally, vertically, or mixed
	// We estimate total bounds as sum of widths and max height (common horizontal layout)
	// This is imperfect but handles the main case: external monitor disconnected
	var totalWidth, maxHeight int
	for _, screen := range screens {
		totalWidth += screen.Size.Width
		if screen.Size.Height > maxHeight {
			maxHeight = screen.Size.Height
		}
	}

	// Check if window would be at least partially visible
	// Allow window to be partially off-screen but require some portion visible
	const minVisible = 100
	windowRight := x + width
	windowBottom := y + height

	// Window must have at least minVisible pixels potentially on-screen
	// For X: window's right edge must be > minVisible, left edge must be < totalWidth - minVisible
	// For Y: window's bottom must be > minVisible, top must be < maxHeight - minVisible
	if windowRight < minVisible || x > totalWidth-minVisible ||
		windowBottom < minVisible || y > maxHeight-minVisible {
		// Position would be mostly/entirely off-screen, let OS decide
		return
	}

	runtime.WindowSetPosition(ctx, x, y)
}

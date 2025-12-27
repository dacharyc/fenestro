package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration
type Config struct {
	// FontSize is the default font size in pixels (e.g., 16, 18, 24)
	FontSize int `toml:"font_size" json:"font_size"`
	// ChromeCSS is the path to a custom CSS file for styling fenestro UI
	ChromeCSS string `toml:"chrome_css" json:"chrome_css"`
}

// DefaultConfig returns the default configuration values
func DefaultConfig() Config {
	return Config{
		FontSize: 0, // 0 means use browser default
	}
}

// getConfigDir returns the config directory following XDG Base Directory standard
func getConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "fenestro")
	}
	// Fall back to ~/.config/fenestro
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fenestro")
}

// getConfigPath returns the full path to the config file
func getConfigPath() string {
	configDir := getConfigDir()
	if configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "config.toml")
}

// LoadConfig loads the configuration from the config file
// Returns default config if file doesn't exist or can't be read
func LoadConfig() Config {
	config := DefaultConfig()

	configPath := getConfigPath()
	if configPath == "" {
		return config
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config
	}

	// Parse the config file
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		// Log error to stderr but continue with defaults
		// Don't fail startup due to config issues
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse config file %s: %v\n", configPath, err)
		fmt.Fprintf(os.Stderr, "Using default configuration. Check TOML syntax (string values must be quoted).\n")
		return DefaultConfig()
	}

	return config
}

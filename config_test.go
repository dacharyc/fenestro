package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.FontSize != 0 {
		t.Errorf("Expected default FontSize to be 0, got %d", config.FontSize)
	}
}

func TestGetConfigDirWithXDGConfigHome(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	dir := getConfigDir()
	expected := "/custom/config/fenestro"
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestGetConfigDirWithoutXDGConfigHome(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Unsetenv("XDG_CONFIG_HOME")
	dir := getConfigDir()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Could not get home dir: %v", err)
	}
	expected := filepath.Join(home, ".config", "fenestro")
	if dir != expected {
		t.Errorf("Expected %s, got %s", expected, dir)
	}
}

func TestLoadConfigNoFile(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	// Point to a directory that doesn't exist
	os.Setenv("XDG_CONFIG_HOME", "/nonexistent/path")
	config := LoadConfig()

	// Should return defaults
	if config.FontSize != 0 {
		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-config-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory and file
	configDir := filepath.Join(tmpDir, "fenestro")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `font_size = 24`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	config := LoadConfig()

	if config.FontSize != 24 {
		t.Errorf("Expected FontSize 24, got %d", config.FontSize)
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-config-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory and file with invalid TOML
	configDir := filepath.Join(tmpDir, "fenestro")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	configContent := `this is not valid toml [`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	config := LoadConfig()

	// Should return defaults on invalid TOML
	if config.FontSize != 0 {
		t.Errorf("Expected default FontSize 0 on invalid TOML, got %d", config.FontSize)
	}
}

func TestLoadConfigPartialConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fenestro-config-test")
	if err != nil {
		t.Fatalf("Could not create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config directory and file with only some options
	configDir := filepath.Join(tmpDir, "fenestro")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Could not create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	// Empty config file - should use defaults
	configContent := `# empty config`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Could not write config file: %v", err)
	}

	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	config := LoadConfig()

	// Should use default for unspecified options
	if config.FontSize != 0 {
		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
	}
}

func TestGetConfig(t *testing.T) {
	// Save and restore XDG_CONFIG_HOME
	original := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", original)

	// Point to nonexistent dir so we get defaults
	os.Setenv("XDG_CONFIG_HOME", "/nonexistent/path")

	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")
	config := app.GetConfig()

	if config.FontSize != 0 {
		t.Errorf("Expected default FontSize 0, got %d", config.FontSize)
	}
}

// TestConfigJSONSerialization verifies that Config serializes to JSON with
// the correct field names that the frontend expects (snake_case, not PascalCase).
// This test ensures the json struct tags are present and correct.
func TestConfigJSONSerialization(t *testing.T) {
	config := Config{
		FontSize:  18,
		ChromeCSS: "/path/to/chrome.css",
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Verify snake_case field names (what frontend expects)
	if !contains(jsonStr, `"font_size"`) {
		t.Errorf("JSON should contain 'font_size' field, got: %s", jsonStr)
	}
	if !contains(jsonStr, `"chrome_css"`) {
		t.Errorf("JSON should contain 'chrome_css' field, got: %s", jsonStr)
	}

	// Verify PascalCase field names are NOT present (would break frontend)
	if contains(jsonStr, `"FontSize"`) {
		t.Errorf("JSON should NOT contain 'FontSize' field (missing json tag), got: %s", jsonStr)
	}
	if contains(jsonStr, `"ChromeCSS"`) {
		t.Errorf("JSON should NOT contain 'ChromeCSS' field (missing json tag), got: %s", jsonStr)
	}

	// Verify the values are correct
	var decoded map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if decoded["font_size"] != float64(18) {
		t.Errorf("Expected font_size=18, got %v", decoded["font_size"])
	}
	if decoded["chrome_css"] != "/path/to/chrome.css" {
		t.Errorf("Expected chrome_css='/path/to/chrome.css', got %v", decoded["chrome_css"])
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

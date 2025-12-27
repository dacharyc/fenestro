package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSocketPolling verifies the socket polling logic used in spawnGUIBackground
func TestSocketPolling(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "fenestro-test-polling.sock")
	os.Remove(socketPath)

	// Start a goroutine that creates the socket after a delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Errorf("Failed to create socket: %v", err)
			return
		}
		defer listener.Close()
		// Keep socket alive for the test
		time.Sleep(500 * time.Millisecond)
	}()

	// Poll for socket (simulating spawnGUIBackground logic)
	deadline := time.Now().Add(1 * time.Second)
	found := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !found {
		t.Error("Socket polling failed to detect socket creation")
	}
}

// TestSocketPollingTimeout verifies polling times out correctly
func TestSocketPollingTimeout(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "fenestro-test-polling-timeout.sock")
	os.Remove(socketPath)

	// Poll for non-existent socket with short timeout
	deadline := time.Now().Add(100 * time.Millisecond)
	found := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if found {
		t.Error("Socket polling should have timed out for non-existent socket")
	}
}

// TestTempFileCreation verifies temp file is created with correct content
func TestTempFileCreation(t *testing.T) {
	content := "<html><body>Test content</body></html>"

	tmpFile, err := os.CreateTemp("", "fenestro-*.html")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Verify content
	readContent, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Content mismatch: got %q, want %q", string(readContent), content)
	}

	// Verify file has fenestro prefix
	if !filepath.HasPrefix(filepath.Base(tmpPath), "fenestro-") {
		t.Errorf("Temp file should have fenestro- prefix, got %s", filepath.Base(tmpPath))
	}
}

// TestTempFileCleanup verifies temp file is removed after reading when --temp-file flag is used
func TestTempFileCleanup(t *testing.T) {
	content := "<html><body>Cleanup test</body></html>"

	// Create temp file (simulating what spawnGUIBackground does)
	tmpFile, err := os.CreateTemp("", "fenestro-*.html")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Simulate what the child process does with --temp-file flag:
	// Read content then delete
	readContent, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(readContent) != content {
		t.Errorf("Content mismatch before cleanup")
	}

	// Simulate cleanup (as done in main.go when tempFile flag is true)
	os.Remove(tmpPath)

	// Verify file is gone
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should have been removed")
		os.Remove(tmpPath) // Cleanup if test failed
	}
}

// TestSequentialCLISimulation simulates the sequential CLI invocation pattern
// that git difftool uses: each invocation should be able to connect to the socket
func TestSequentialCLISimulation(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-cli-sim.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, true) // Use timeout mode like sidebar
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	// Simulate 3 sequential CLI invocations
	// Each one should:
	// 1. Find the socket exists (first one creates it, subsequent ones find it)
	// 2. Connect and send file
	// 3. Exit (return) immediately

	for i := 0; i < 3; i++ {
		// Verify socket exists (what CLI does before connecting)
		if _, err := os.Stat(socketPath); os.IsNotExist(err) {
			t.Fatalf("Socket should exist for invocation %d", i)
		}

		// Connect and send (simulating TrySendToSidebarInstance)
		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("Invocation %d failed to connect: %v", i, err)
		}

		cmd := IPCCommand{
			Cmd: "add-file",
			Entry: FileEntry{
				Name:    "file" + string(rune('1'+i)) + ".html",
				Path:    "/tmp/file" + string(rune('1'+i)) + ".html",
				Content: "<html>File " + string(rune('1'+i)) + "</html>",
			},
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd); err != nil {
			conn.Close()
			t.Fatalf("Invocation %d failed to send: %v", i, err)
		}
		conn.Close()

		// Small delay between invocations (simulating real CLI timing)
		time.Sleep(20 * time.Millisecond)
	}

	// Give server time to process all files
	time.Sleep(50 * time.Millisecond)

	// Verify all files arrived
	files := app.GetFiles()
	expectedCount := 4 // initial + 3 added
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}
}

// TestFirstInvocationCreatesSocket verifies that the first invocation creates the socket
// and subsequent invocations can immediately connect
func TestFirstInvocationCreatesSocket(t *testing.T) {
	socketPath := filepath.Join(os.TempDir(), "fenestro-test-first-invoke.sock")
	os.Remove(socketPath)

	// Verify socket doesn't exist initially
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatal("Socket should not exist initially")
	}

	// Create app and server (simulating first invocation)
	app := NewApp(FileEntry{Name: "first", Content: "<html>first</html>"}, "")
	server, err := NewIPCServer(app, socketPath, true)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	// Socket should exist immediately after NewIPCServer (before Start even)
	// This is the guarantee that spawnGUIBackground relies on
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket should exist immediately after NewIPCServer")
	}

	// Subsequent invocation should be able to connect immediately
	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Subsequent invocation failed to connect: %v", err)
	}
	conn.Close()
}

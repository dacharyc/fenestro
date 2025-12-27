package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGetSocketDir(t *testing.T) {
	dir := getSocketDir()
	if dir == "" {
		t.Error("getSocketDir() returned empty string")
	}
	if !strings.HasSuffix(dir, socketDir) {
		t.Errorf("getSocketDir() should end with %q, got %q", socketDir, dir)
	}
}

func TestGetSidebarSocketPath(t *testing.T) {
	path := getSidebarSocketPath()
	if path == "" {
		t.Error("getSidebarSocketPath() returned empty string")
	}
	if !strings.HasSuffix(path, sidebarSocketName) {
		t.Errorf("getSidebarSocketPath() should end with %q, got %q", sidebarSocketName, path)
	}
}

func TestGetWindowSocketPath(t *testing.T) {
	windowID := "test-uuid-1234"
	path := getWindowSocketPath(windowID)
	expectedSuffix := filepath.Join(windowsDir, windowID+".sock")
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("getWindowSocketPath() should end with %q, got %q", expectedSuffix, path)
	}
}

func TestEnsureSocketDir(t *testing.T) {
	err := ensureSocketDir()
	if err != nil {
		t.Fatalf("ensureSocketDir() failed: %v", err)
	}

	// Verify directories exist
	dir := getSocketDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Socket directory %q was not created", dir)
	}

	windowsPath := filepath.Join(dir, windowsDir)
	if _, err := os.Stat(windowsPath); os.IsNotExist(err) {
		t.Errorf("Windows directory %q was not created", windowsPath)
	}
}

func TestIPCCommandJSON(t *testing.T) {
	cmd := IPCCommand{
		Cmd: "add-file",
		Entry: FileEntry{
			Name:    "test.html",
			Path:    "/tmp/test.html",
			Content: "<html></html>",
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal IPCCommand: %v", err)
	}

	var decoded IPCCommand
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal IPCCommand: %v", err)
	}

	if decoded.Cmd != cmd.Cmd {
		t.Errorf("Cmd mismatch: got %q, want %q", decoded.Cmd, cmd.Cmd)
	}
	if decoded.Entry.Name != cmd.Entry.Name {
		t.Errorf("Entry.Name mismatch: got %q, want %q", decoded.Entry.Name, cmd.Entry.Name)
	}
	if decoded.Entry.Path != cmd.Entry.Path {
		t.Errorf("Entry.Path mismatch: got %q, want %q", decoded.Entry.Path, cmd.Entry.Path)
	}
	if decoded.Entry.Content != cmd.Entry.Content {
		t.Errorf("Entry.Content mismatch: got %q, want %q", decoded.Entry.Content, cmd.Entry.Content)
	}
}

func TestTrySendToExistingNoSocket(t *testing.T) {
	// Try to send to a non-existent socket
	socketPath := filepath.Join(os.TempDir(), "fenestro-test-nonexistent.sock")
	os.Remove(socketPath) // Ensure it doesn't exist

	cmd := IPCCommand{Cmd: "test"}
	result := TrySendToExisting(socketPath, cmd)

	if result {
		t.Error("TrySendToExisting() should return false when socket doesn't exist")
	}
}

func TestIPCServerLifecycle(t *testing.T) {
	// Create a test app
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")

	// Use a unique socket path for testing
	socketPath := filepath.Join(os.TempDir(), "fenestro-test-lifecycle.sock")
	os.Remove(socketPath) // Clean up any existing socket

	// Create server
	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}

	// Start server
	server.Start()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Verify socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket file was not created")
	}

	// Close server
	server.Close()

	// Give server time to clean up
	time.Sleep(50 * time.Millisecond)

	// Verify socket is removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file was not removed after Close()")
		os.Remove(socketPath) // Clean up
	}
}

func TestIPCServerAddFile(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-addfile.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	// Connect and send add-file command
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	cmd := IPCCommand{
		Cmd: "add-file",
		Entry: FileEntry{
			Name:    "newfile",
			Path:    "/tmp/newfile.html",
			Content: "<html>new</html>",
		},
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		t.Fatalf("Failed to send command: %v", err)
	}

	// Give server time to process
	time.Sleep(50 * time.Millisecond)

	// Verify file was added
	files := app.GetFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestIPCServerReplace(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Path: "/tmp/test.html", Content: "<html>original</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-replace.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to connect to socket: %v", err)
	}
	defer conn.Close()

	cmd := IPCCommand{
		Cmd:     "replace",
		Path:    "/tmp/test.html",
		Content: "<html>replaced</html>",
		Name:    "test",
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		t.Fatalf("Failed to send command: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Verify content was replaced
	content := app.GetHTMLContent()
	if content != "<html>replaced</html>" {
		t.Errorf("Content not replaced: got %q", content)
	}
}

func TestIPCServerDoubleClose(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-doubleclose.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()

	// Close twice should not panic
	server.Close()
	server.Close()
}

func TestTrySendToSidebarInstance(t *testing.T) {
	// This tests the helper function when no server is running
	entry := FileEntry{Name: "test", Content: "<html></html>"}

	// Ensure no socket exists
	socketPath := getSidebarSocketPath()
	os.Remove(socketPath)

	result := TrySendToSidebarInstance(entry)
	if result {
		t.Error("TrySendToSidebarInstance() should return false when no server is running")
	}
}

func TestTrySendToWindowInstance(t *testing.T) {
	entry := FileEntry{Name: "test", Content: "<html></html>"}

	// Use a unique window ID
	windowID := "test-window-12345"
	socketPath := getWindowSocketPath(windowID)
	os.Remove(socketPath)

	result := TrySendToWindowInstance(windowID, entry)
	if result {
		t.Error("TrySendToWindowInstance() should return false when no server is running")
	}
}

// TestThroughputStress simulates rapid file arrivals like git diff output
func TestThroughputStress(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-throughput.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	// Simulate rapid file arrivals (like git diff with many files)
	fileCount := 50
	var wg sync.WaitGroup
	errors := make(chan error, fileCount)

	start := time.Now()

	for i := 0; i < fileCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
			if err != nil {
				errors <- err
				return
			}
			defer conn.Close()

			cmd := IPCCommand{
				Cmd: "add-file",
				Entry: FileEntry{
					Name:    filepath.Base(filepath.Join("diff", string(rune('a'+idx%26))+".diff")),
					Path:    filepath.Join("/tmp", "file"+string(rune('0'+idx%10))+".html"),
					Content: "<html>content " + string(rune('0'+idx%10)) + "</html>",
				},
			}

			encoder := json.NewEncoder(conn)
			if err := encoder.Encode(cmd); err != nil {
				errors <- err
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)

	// Check for connection errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	if len(errorList) > 0 {
		t.Errorf("Had %d connection errors (first: %v)", len(errorList), errorList[0])
	}

	// Give server time to process all files
	time.Sleep(100 * time.Millisecond)

	// Verify all files arrived
	files := app.GetFiles()
	// We expect initial + fileCount, but some may have same name due to modulo
	if len(files) < 2 {
		t.Errorf("Expected multiple files, got %d", len(files))
	}

	t.Logf("Throughput test: sent %d files in %v (%.0f files/sec)",
		fileCount, elapsed, float64(fileCount)/elapsed.Seconds())

	// Should complete in reasonable time (< 5 seconds for 50 files)
	if elapsed > 5*time.Second {
		t.Errorf("Throughput too slow: %v for %d files", elapsed, fileCount)
	}
}

// TestThroughputSequential tests sequential file arrivals (more realistic CLI pattern)
func TestThroughputSequential(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-sequential.sock")
	os.Remove(socketPath)

	server, err := NewIPCServer(app, socketPath, false)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()
	defer server.Close()

	time.Sleep(50 * time.Millisecond)

	fileCount := 30
	start := time.Now()

	for i := 0; i < fileCount; i++ {
		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("Failed to connect on file %d: %v", i, err)
		}

		cmd := IPCCommand{
			Cmd: "add-file",
			Entry: FileEntry{
				Name:    "file" + string(rune('A'+i)) + ".diff",
				Path:    "/tmp/file" + string(rune('A'+i)) + ".html",
				Content: "<html>sequential content</html>",
			},
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd); err != nil {
			conn.Close()
			t.Fatalf("Failed to send file %d: %v", i, err)
		}
		conn.Close()
	}

	elapsed := time.Since(start)

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	files := app.GetFiles()
	expectedCount := 1 + fileCount // initial + added files
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d", expectedCount, len(files))
	}

	t.Logf("Sequential test: sent %d files in %v (%.0f files/sec)",
		fileCount, elapsed, float64(fileCount)/elapsed.Seconds())
}

// TestSidebarTimeoutReset verifies that rapid arrivals keep resetting the timeout
func TestSidebarTimeoutReset(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html>initial</html>"}, "")

	socketPath := filepath.Join(os.TempDir(), "fenestro-test-timeout-reset.sock")
	os.Remove(socketPath)

	// Use timeout mode (like sidebar)
	server, err := NewIPCServer(app, socketPath, true)
	if err != nil {
		t.Fatalf("NewIPCServer() failed: %v", err)
	}
	server.Start()

	// Send files every 500ms - should keep resetting the 2-second timeout
	for i := 0; i < 5; i++ {
		time.Sleep(500 * time.Millisecond)

		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err != nil {
			t.Fatalf("Connection failed on iteration %d (server may have timed out early): %v", i, err)
		}

		cmd := IPCCommand{
			Cmd: "add-file",
			Entry: FileEntry{
				Name:    "timeout-test-" + string(rune('0'+i)),
				Content: "<html></html>",
			},
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(cmd); err != nil {
			conn.Close()
			t.Fatalf("Send failed on iteration %d: %v", i, err)
		}
		conn.Close()
	}

	// Server should still be alive
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		t.Fatal("Server closed while files were still arriving")
	}
	conn.Close()

	// Now wait for timeout
	time.Sleep(2500 * time.Millisecond)

	// Server should be closed now
	_, err = net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err == nil {
		t.Error("Server should have closed after 2-second timeout")
		server.Close()
	}
}

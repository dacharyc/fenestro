package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	socketDir         = ".fenestro"
	sidebarSocketName = "fenestro.sock"
	windowsDir        = "windows"
	groupingTimeout   = 2 * time.Second
)

// IPCCommand represents a command sent via IPC
type IPCCommand struct {
	Cmd     string    `json:"cmd"`     // "add-file" or "replace"
	Entry   FileEntry `json:"entry"`   // for add-file
	Path    string    `json:"path"`    // for replace
	Content string    `json:"content"` // for replace
	Name    string    `json:"name"`    // for replace
}

// IPCServer manages the Unix socket server for receiving commands
type IPCServer struct {
	listener     net.Listener
	socketPath   string
	app          *App
	mu           sync.Mutex
	closed       bool
	timeoutTimer *time.Timer
	useTimeout   bool // false for window ID mode (persistent)
}

// getSocketDir returns the socket directory path
func getSocketDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	return filepath.Join(homeDir, socketDir)
}

// getSidebarSocketPath returns the path for the sidebar mode socket
func getSidebarSocketPath() string {
	return filepath.Join(getSocketDir(), sidebarSocketName)
}

// getWindowSocketPath returns the path for a specific window ID socket
func getWindowSocketPath(windowID string) string {
	return filepath.Join(getSocketDir(), windowsDir, windowID+".sock")
}

// ensureSocketDir creates the socket directory if it doesn't exist
func ensureSocketDir() error {
	dir := getSocketDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	windowsPath := filepath.Join(dir, windowsDir)
	return os.MkdirAll(windowsPath, 0700)
}

// TrySendToExisting tries to connect to an existing instance and send a command
// Returns true if successful (caller should exit), false if no instance running
func TrySendToExisting(socketPath string, cmd IPCCommand) bool {
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		// Connection failed - socket might be stale, clean it up
		os.Remove(socketPath)
		return false
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return false
	}

	return true
}

// TrySendToSidebarInstance tries to send a file to an existing sidebar instance
func TrySendToSidebarInstance(entry FileEntry) bool {
	cmd := IPCCommand{
		Cmd:   "add-file",
		Entry: entry,
	}
	return TrySendToExisting(getSidebarSocketPath(), cmd)
}

// TrySendToWindowInstance tries to send content to a specific window
func TrySendToWindowInstance(windowID string, entry FileEntry) bool {
	cmd := IPCCommand{
		Cmd:     "replace",
		Path:    entry.Path,
		Content: entry.Content,
		Name:    entry.Name,
	}
	return TrySendToExisting(getWindowSocketPath(windowID), cmd)
}

// NewIPCServer creates a new IPC server
func NewIPCServer(app *App, socketPath string, useTimeout bool) (*IPCServer, error) {
	if err := ensureSocketDir(); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket file if it exists
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	server := &IPCServer{
		listener:   listener,
		socketPath: socketPath,
		app:        app,
		useTimeout: useTimeout,
	}

	// Start timeout timer if in sidebar mode
	if useTimeout {
		server.resetTimeout()
	}

	return server, nil
}

// resetTimeout resets the grouping timeout timer
func (s *IPCServer) resetTimeout() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
	}

	s.timeoutTimer = time.AfterFunc(groupingTimeout, func() {
		s.Close()
	})
}

// Start begins accepting connections
func (s *IPCServer) Start() {
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				s.mu.Lock()
				closed := s.closed
				s.mu.Unlock()
				if closed {
					return
				}
				continue
			}

			go s.handleConnection(conn)
		}
	}()
}

// handleConnection processes a single IPC connection
func (s *IPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Reset timeout on each new file (sidebar mode only)
	if s.useTimeout {
		s.resetTimeout()
	}

	decoder := json.NewDecoder(conn)
	var cmd IPCCommand
	if err := decoder.Decode(&cmd); err != nil {
		return
	}

	switch cmd.Cmd {
	case "add-file":
		s.app.AddFile(cmd.Entry)
	case "replace":
		s.app.ReplaceFileContent(cmd.Path, cmd.Content, cmd.Name)
	}
}

// Close shuts down the IPC server and removes the socket file
func (s *IPCServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	if s.timeoutTimer != nil {
		s.timeoutTimer.Stop()
	}

	if s.listener != nil {
		s.listener.Close()
	}

	// Clean up socket file
	os.Remove(s.socketPath)
}

// StartSidebarServer starts an IPC server for sidebar mode with timeout
func StartSidebarServer(app *App) (*IPCServer, error) {
	server, err := NewIPCServer(app, getSidebarSocketPath(), true)
	if err != nil {
		return nil, err
	}
	server.Start()
	return server, nil
}

// StartWindowServer starts an IPC server for a specific window (no timeout)
func StartWindowServer(app *App, windowID string) (*IPCServer, error) {
	server, err := NewIPCServer(app, getWindowSocketPath(windowID), false)
	if err != nil {
		return nil, err
	}
	server.Start()
	return server, nil
}

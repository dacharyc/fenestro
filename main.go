package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	flag "github.com/spf13/pflag"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed frontend/*
var assets embed.FS

const Version = "2.0.0"

var (
	filePath    string
	displayName string
	windowID    string
	showVersion bool
	internalGUI bool // Hidden flag: run as GUI subprocess
	tempFile    bool // Hidden flag: delete file after reading (for stdin content)
)

func init() {
	flag.StringVarP(&filePath, "path", "p", "", "Path to HTML file to display")
	flag.StringVarP(&displayName, "name", "n", "", "Display name for the window title")
	flag.StringVar(&windowID, "id", "", "Window ID: use 'new' to generate ID, or provide existing UUID to target that window")
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version")
	flag.BoolVar(&internalGUI, "internal-gui", false, "Internal: run as GUI subprocess")
	flag.BoolVar(&tempFile, "temp-file", false, "Internal: delete file after reading")
	flag.CommandLine.MarkHidden("internal-gui")
	flag.CommandLine.MarkHidden("temp-file")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("fenestro %s\n", Version)
		os.Exit(0)
	}

	// Determine content source and create FileEntry
	var entry FileEntry
	var fromStdin bool

	if filePath != "" {
		// Load from file path
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}
		content, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		entry = FileEntry{
			Name:    displayName,
			Path:    absPath,
			Content: string(content),
		}
		if entry.Name == "" {
			entry.Name = filepath.Base(filePath)
		}
		// If this was a temp file (from stdin in parent), clean it up after reading
		if tempFile {
			os.Remove(absPath)
		}
	} else if !isTerminal(os.Stdin) {
		// Read from stdin
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		entry = FileEntry{
			Name:    displayName,
			Path:    "", // stdin has no path
			Content: string(content),
		}
		if entry.Name == "" {
			entry.Name = "stdin"
		}
		fromStdin = true
	} else {
		// No input provided
		fmt.Println("Usage: fenestro [-p path] [-n name] [-id [window-id]]")
		fmt.Println("       echo '<html>...</html>' | fenestro")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -p, --path    Path to HTML file to display")
		fmt.Println("  -n, --name    Display name for the window title")
		fmt.Println("  -id           Window ID mode ('new' = generate ID, '<uuid>' = target window)")
		fmt.Println("  -v, --version Show version")
		fmt.Println()
		fmt.Println("Sidebar mode (default):")
		fmt.Println("  Files opened within 2 seconds are grouped in the same window.")
		fmt.Println()
		fmt.Println("Window ID mode (-id):")
		fmt.Println("  fenestro -p file.html -id new    # Create window, print UUID")
		fmt.Println("  fenestro -p file.html -id <uuid> # Replace content in window")
		os.Exit(0)
	}

	// Check if we're using window ID mode
	isWindowIDMode := windowID != ""

	// Handle window ID "new" - generate UUID before any IPC or spawning
	if isWindowIDMode && windowID == "new" {
		windowID = uuid.New().String()
		fmt.Println(windowID)
	}

	// If this is the GUI subprocess, run the GUI directly
	if internalGUI {
		runGUI(entry, windowID, isWindowIDMode)
		return
	}

	// CLI invocation - try to send to existing instance first
	if isWindowIDMode {
		if windowID != "" {
			// Validate UUID format (skip if we just generated it above)
			if _, err := uuid.Parse(windowID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: Invalid window ID format (expected UUID): %s\n", windowID)
				os.Exit(1)
			}
			// Try to send to existing window
			if TrySendToWindowInstance(windowID, entry) {
				os.Exit(0)
			}
		}
	} else {
		// Sidebar mode - try to send to existing instance
		if TrySendToSidebarInstance(entry) {
			os.Exit(0)
		}
	}

	// No existing instance - spawn GUI in background and exit
	if err := spawnGUIBackground(entry, windowID, fromStdin); err != nil {
		fmt.Fprintf(os.Stderr, "Error spawning GUI: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// spawnGUIBackground spawns the GUI as a background process and waits for the socket to be ready
func spawnGUIBackground(entry FileEntry, windowID string, fromStdin bool) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	args := []string{"--internal-gui"}

	// Handle content: if from stdin, write to temp file; otherwise use original path
	if fromStdin {
		tmpFile, err := os.CreateTemp("", "fenestro-*.html")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		if _, err := tmpFile.WriteString(entry.Content); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		args = append(args, "-p", tmpFile.Name(), "--temp-file")
	} else {
		args = append(args, "-p", entry.Path)
	}

	// Pass display name if it was explicitly set
	if displayName != "" {
		args = append(args, "-n", displayName)
	}

	// Pass window ID if set
	if windowID != "" {
		args = append(args, "-id", windowID)
	}

	// Spawn the child process detached
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session so child survives parent exit
	}
	// Don't inherit stdin (child reads from file), but keep stderr for errors
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start GUI process: %w", err)
	}

	// Wait for socket to be created (guarantees subsequent invocations can connect)
	var socketPath string
	if windowID != "" {
		socketPath = getWindowSocketPath(windowID)
	} else {
		socketPath = getSidebarSocketPath()
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return nil // Socket exists, child is ready
		}
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for GUI to start")
}

// runGUI runs the Wails application (called from GUI subprocess)
func runGUI(entry FileEntry, windowID string, isWindowIDMode bool) {
	// Create app with the file entry
	app := NewApp(entry, windowID)

	// Load saved window state
	state := LoadWindowState()
	config := app.config

	// Determine window dimensions
	width, height := GetWindowDimensions(state, config)

	// Determine window position (to be set after startup)
	x, y, shouldSetPosition := GetWindowPosition(state, config)
	app.initialX = x
	app.initialY = y
	app.initialWidth = width
	app.initialHeight = height
	app.shouldSetPosition = shouldSetPosition

	// Start IPC server
	var ipcServer *IPCServer
	var err error
	if isWindowIDMode {
		ipcServer, err = StartWindowServer(app, windowID)
	} else {
		ipcServer, err = StartSidebarServer(app)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not start IPC server: %v\n", err)
	}

	// Run Wails application
	err = wails.Run(&options.App{
		Title:     entry.Name,
		Width:     width,
		Height:    height,
		MinWidth:  MinWindowWidth,
		MinHeight: MinWindowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		OnShutdown: func(ctx context.Context) {
			if ipcServer != nil {
				ipcServer.Close()
			}
		},
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

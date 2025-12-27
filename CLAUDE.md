# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Fenestro is a macOS app that renders HTML in a native window. It accepts HTML via stdin or file path from the command line, then displays it in a WebView. Built with Go using Wails for native macOS WebView integration.

## Build Commands

```bash
# Build the production binary
make build

# Run in development mode with hot reload
make dev

# Install to /usr/local/bin
make install

# Run with test HTML
make run

# Run Go tests
go test -v ./...

# Run frontend tests
cd frontend && npm test

# Clean build artifacts
make clean
```

## Architecture

Wails application with Go backend and HTML/JS frontend:

- **main.go**: Entry point, CLI flag parsing, IPC check, Wails app initialization
- **app.go**: Application struct with multi-file support, methods exposed to frontend
- **files.go**: FileEntry struct, utility functions
- **ipc.go**: Unix domain socket IPC for sidebar grouping and window ID mode
- **frontend/index.html**: HTML wrapper with find bar, sidebar, and content container
- **frontend/main.js**: Find-in-page, sidebar logic, backend event handling
- **frontend/html-renderer.js**: DOMParser-based HTML rendering that preserves scripts/styles from `<head>`
- **frontend/style.css**: Find bar and sidebar styling with dark mode support

## Key Dependencies

- **github.com/wailsapp/wails/v2**: Go framework for desktop apps using web technologies
- **github.com/google/uuid**: UUID generation for window ID mode

## Wails Patterns

- Frontend assets are embedded via `//go:embed frontend/*`
- Backend methods are exposed to frontend via `Bind` in app options
- Frontend calls Go methods via `window.go.main.App.MethodName()`
- Wails handles the WebView and window management

## Features

- **Cmd+F Find**: JavaScript-based find-in-page with highlight and navigation
- **Stdin support**: Pipe HTML content directly
- **File path support**: Load HTML from file with `-p` flag
- **Sidebar**: Files opened within 2 seconds are grouped in same window with sidebar
- **Window ID mode**: `-id new` creates window with UUID, `-id <uuid>` updates existing window
- **Dark mode**: Automatic styling for UI elements based on system preference

## IPC System

Uses Unix domain sockets for inter-process communication:
- Sidebar mode socket: `~/.fenestro/fenestro.sock` (2-second timeout)
- Window ID sockets: `~/.fenestro/windows/<uuid>.sock` (persistent)
- Stale sockets are auto-cleaned on failed connection attempts

## Development Notes

- Build output goes to `build/bin/fenestro.app`
- Use `make dev` for hot reload during development
- The binary inside the .app bundle can be run directly for CLI usage

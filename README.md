# Fenestro

A lightweight macOS app that renders HTML in a native window. Built with Go using [Wails](https://wails.io/).

## Features

- Display HTML from files or stdin
- Native macOS WebView
- Cmd+F find-in-page with highlight navigation
- Zoom in/out with Cmd+/Cmd-
- Configurable default font size and custom chrome CSS
- Sidebar for multiple files (files opened within 2 seconds are grouped together)
- Window ID mode for live content updates
- Dark mode support for UI elements
- Minimal footprint - single binary app bundle

## Installation

### Prerequisites

- macOS 11+
- Go 1.24+
- Xcode Command Line Tools (`xcode-select --install`)
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Build & Install

```bash
git clone https://github.com/dacharyc/fenestro.git
cd fenestro
make install
```

This installs `fenestro.app` to `/Applications`. To use the `fenestro` command from your terminal, either add it to your PATH or create a symlink:

```bash
# Option 1: Add to your shell profile (.zshrc, .bashrc, etc.)
export PATH="/Applications/fenestro.app/Contents/MacOS:$PATH"

# Option 2: Create a symlink
ln -s /Applications/fenestro.app/Contents/MacOS/fenestro /usr/local/bin/fenestro
```

## Usage

### Display an HTML file

```bash
fenestro -p report.html
fenestro --path=report.html
```

### Pipe HTML from stdin

```bash
echo '<html><body><h1>Hello!</h1></body></html>' | fenestro

# Colorize terminal output and display
ack --color foo src/ | aha | fenestro

# View command output with syntax highlighting
cat code.py | pygmentize -f html | fenestro
```

### Custom display name

```bash
fenestro -p output.html -n "Search Results"
```

### Show version

```bash
fenestro -v
```

### Window ID Mode

Target a specific window for live content updates:

```bash
# Create a new window and get its ID
fenestro -p report.html -id new > /tmp/window_id.txt &
WINDOW_ID=$(cat /tmp/window_id.txt)

# Update that window with new content later
fenestro -p updated_report.html -id $WINDOW_ID

# The window finds the file by path, updates its content, and displays it
```

This is useful for:
- Live-reloading documentation as you edit
- Updating build output in real-time
- Monitoring log files converted to HTML

## Keyboard Shortcuts

- **Cmd+F** - Find in page
- **Cmd+Plus** - Zoom in
- **Cmd+Minus** - Zoom out
- **Cmd+0** - Reset zoom to 100%
- **Cmd+W** - Close window
- **Cmd+Q** - Quit

## Configuration

Fenestro supports a TOML configuration file following the [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/latest/) standard.

**Location:** `$XDG_CONFIG_HOME/fenestro/config.toml` (defaults to `~/.config/fenestro/config.toml`)

### Example Files

The `examples/` directory contains ready-to-use templates:

- **`examples/config.toml`** - Documented configuration with all options
- **`examples/chrome.css`** - Comprehensive CSS template with all stylable elements and example themes (Nord, Solarized Light, Dracula)

To get started quickly:

```bash
mkdir -p ~/.config/fenestro
cp examples/config.toml ~/.config/fenestro/
cp examples/chrome.css ~/.config/fenestro/
```

Then edit the files to customize font size and UI styling.

### Available Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `font_size` | integer | 0 | Base font size in pixels. Set to 0 to use browser default. |
| `chrome_css` | string | "" | Path to a CSS file for customizing fenestro UI (find bar, sidebar, etc.). |

The font size setting works alongside zoom (Cmd+/Cmd-) for additional flexibility.

### Custom Chrome CSS

The `chrome_css` option lets you style fenestro's UI elements (the "chrome") separately from your HTML content. Create a CSS file and reference it in your config:

```css
/* Example: ~/.config/fenestro/chrome.css */

/* Style the find bar */
#find-bar {
    background: #1a1a2e;
    border-color: #16213e;
}

#find-input {
    background: #0f0f23;
    color: #e0e0e0;
}

/* Style the sidebar */
#sidebar {
    background: #1a1a2e;
}

.file-item {
    color: #a0a0a0;
}

.file-item:hover {
    background: #16213e;
}

.file-item.selected {
    background: #0f3460;
    color: #ffffff;
}

/* Style search highlights */
.find-highlight {
    background: #e2b714;
}

.find-highlight.current {
    background: #ff6b6b;
}
```

Available CSS selectors:
- `#find-bar` - Find bar container
- `#find-input` - Find text input
- `#find-count` - Match count display
- `#sidebar` - Sidebar container
- `#file-list` - File list within sidebar
- `.file-item` - Individual file entries
- `.file-item.selected` - Currently selected file
- `#content` - Main content area
- `.find-highlight` - Search match highlights
- `.find-highlight.current` - Current search match

## Development

```bash
# Build without installing
make build

# Build and run with test HTML
make run

# Run tests
make test

# Clean build artifacts
make clean

# Uninstall from /Applications
make uninstall
```

## Troubleshooting

### Stale Sockets

Fenestro uses Unix domain sockets for inter-process communication (sidebar grouping and window ID mode). Sockets are stored in `~/.fenestro/`.

If a fenestro process is force-killed (e.g., via `kill -9`), its socket may not be cleaned up. Stale sockets are automatically removed when a new connection attempt fails, but you can also clean them up manually:

```bash
# Remove all fenestro sockets
rm -rf ~/.fenestro/
```

## Architecture

### Background Process Model

Fenestro uses a background process model to ensure the CLI always exits immediately - essential for integration with tools like `git difftool` that wait for each command to complete before proceeding.

When you run `fenestro`:
1. The CLI checks for an existing fenestro window (via Unix socket)
2. If found, it sends the file via IPC and exits immediately
3. If not found, it spawns the GUI as a background process, waits for the socket to be ready, then exits

This means every `fenestro` invocation returns immediately, while windows run independently in the background.

### Wails v2 Limitation

Wails v2 only supports a single window per application process. This means each "window group" (files opened within 2 seconds) runs in its own process with its own Wails stack.

The original Swift version (Fenestro) used a single-process architecture where one app managed multiple windows via NSDocument. This was more memory-efficient but macOS-specific.

### Future: Wails v3

[Wails v3](https://v3alpha.wails.io/whats-new/) introduces native multi-window support, which would allow fenestro to use a single-process daemon architecture:
- One persistent process managing all windows
- Lower memory footprint
- Simpler IPC (all windows in one process)

When Wails v3 reaches stable release, consider refactoring to this architecture. The relevant tracking issue is [wailsapp/wails#1480](https://github.com/wailsapp/wails/issues/1480).

## History

Fenestro is a rewrite of [Fenestro](https://github.com/masukomi/fenestro), originally written in Swift. This Go version uses Wails for native macOS WebView integration.

## License

MIT

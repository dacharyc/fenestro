BINARY = fenestro
VERSION = 2.0.0
INSTALL_PATH ?= /Applications
WAILS ?= $(shell command -v wails || echo ~/go/bin/wails)
APP_BUNDLE = build/bin/$(BINARY).app
APP_BINARY = $(APP_BUNDLE)/Contents/MacOS/$(BINARY)

.PHONY: build dev install uninstall clean test run

# Build the production binary
build:
	$(WAILS) build

# Run in development mode with hot reload
dev:
	$(WAILS) dev

# Install app bundle to /Applications
install: build
	@if [ -w $(INSTALL_PATH) ]; then \
		rm -rf $(INSTALL_PATH)/$(BINARY).app; \
		cp -r $(APP_BUNDLE) $(INSTALL_PATH)/; \
		echo "Installed $(BINARY).app to $(INSTALL_PATH)"; \
	else \
		sudo rm -rf $(INSTALL_PATH)/$(BINARY).app; \
		sudo cp -r $(APP_BUNDLE) $(INSTALL_PATH)/; \
		echo "Installed $(BINARY).app to $(INSTALL_PATH) (with sudo)"; \
	fi

# Remove from /Applications
uninstall:
	@if [ -w $(INSTALL_PATH)/$(BINARY).app ]; then \
		rm -rf $(INSTALL_PATH)/$(BINARY).app; \
	else \
		sudo rm -rf $(INSTALL_PATH)/$(BINARY).app; \
	fi
	@echo "Uninstalled $(BINARY).app"

# Clean build artifacts
clean:
	rm -rf build/bin
	go clean

# Run tests
test:
	go test -v ./...

# Build and run with a test file
run: build
	echo '<html><body><h1>Hello from Fenestro!</h1><p>This is a test paragraph with some searchable text.</p><p>Press Cmd+F to find text in this page.</p></body></html>' | ./$(APP_BINARY)

# Show help
help:
	@echo "Fenestro - HTML viewer for macOS"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the production binary"
	@echo "  dev       - Run in development mode with hot reload"
	@echo "  install   - Install app bundle to $(INSTALL_PATH)"
	@echo "  uninstall - Remove app bundle from $(INSTALL_PATH)"
	@echo "  clean     - Remove build artifacts"
	@echo "  test      - Run tests"
	@echo "  run       - Build and run with test HTML"
	@echo "  help      - Show this help"

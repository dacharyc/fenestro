package main

import (
	"sync"
	"testing"
)

func TestNewApp(t *testing.T) {
	entry := FileEntry{
		Name:    "test.html",
		Path:    "/tmp/test.html",
		Content: "<html>test</html>",
	}
	windowID := "test-window-id"

	app := NewApp(entry, windowID)

	if app == nil {
		t.Fatal("NewApp() returned nil")
	}

	if len(app.files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(app.files))
	}

	if app.files[0].Name != entry.Name {
		t.Errorf("File name mismatch: got %q, want %q", app.files[0].Name, entry.Name)
	}

	if app.currentIndex != 0 {
		t.Errorf("Expected currentIndex 0, got %d", app.currentIndex)
	}

	if app.windowID != windowID {
		t.Errorf("Window ID mismatch: got %q, want %q", app.windowID, windowID)
	}
}

func TestGetHTMLContent(t *testing.T) {
	content := "<html><body>Hello</body></html>"
	app := NewApp(FileEntry{Name: "test", Content: content}, "")

	got := app.GetHTMLContent()
	if got != content {
		t.Errorf("GetHTMLContent() = %q, want %q", got, content)
	}
}

func TestGetHTMLContentEmpty(t *testing.T) {
	app := &App{
		files:        []FileEntry{},
		currentIndex: 0,
	}

	got := app.GetHTMLContent()
	if got != "" {
		t.Errorf("GetHTMLContent() with empty files should return empty string, got %q", got)
	}
}

func TestGetHTMLContentInvalidIndex(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")
	app.currentIndex = 5 // Invalid index

	got := app.GetHTMLContent()
	if got != "" {
		t.Errorf("GetHTMLContent() with invalid index should return empty string, got %q", got)
	}
}

func TestGetFiles(t *testing.T) {
	app := NewApp(FileEntry{Name: "test1", Content: "<html>1</html>"}, "")
	app.files = append(app.files, FileEntry{Name: "test2", Content: "<html>2</html>"})

	files := app.GetFiles()

	if len(files) != 2 {
		t.Errorf("GetFiles() returned %d files, want 2", len(files))
	}

	// Verify it's a copy by modifying the returned slice
	files[0].Name = "modified"
	if app.files[0].Name == "modified" {
		t.Error("GetFiles() should return a copy, not the original slice")
	}
}

func TestGetCurrentIndex(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")
	app.currentIndex = 3

	if got := app.GetCurrentIndex(); got != 3 {
		t.Errorf("GetCurrentIndex() = %d, want 3", got)
	}
}

func TestSelectFile(t *testing.T) {
	app := NewApp(FileEntry{Name: "file1", Content: "<html>1</html>"}, "")
	app.files = append(app.files, FileEntry{Name: "file2", Content: "<html>2</html>"})
	app.files = append(app.files, FileEntry{Name: "file3", Content: "<html>3</html>"})

	content := app.SelectFile(1)
	if content != "<html>2</html>" {
		t.Errorf("SelectFile(1) returned %q, want %q", content, "<html>2</html>")
	}

	if app.currentIndex != 1 {
		t.Errorf("currentIndex should be 1, got %d", app.currentIndex)
	}
}

func TestSelectFileInvalidIndex(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")

	// Test negative index
	content := app.SelectFile(-1)
	if content != "" {
		t.Errorf("SelectFile(-1) should return empty string, got %q", content)
	}

	// Test out of bounds index
	content = app.SelectFile(10)
	if content != "" {
		t.Errorf("SelectFile(10) should return empty string, got %q", content)
	}
}

func TestAddFile(t *testing.T) {
	app := NewApp(FileEntry{Name: "beta", Content: "<html>beta</html>"}, "")

	// Add file that should sort before "beta"
	app.AddFile(FileEntry{Name: "alpha", Path: "/tmp/alpha.html", Content: "<html>alpha</html>"})

	files := app.GetFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Files should be sorted alphabetically
	if files[0].Name != "alpha" {
		t.Errorf("First file should be 'alpha', got %q", files[0].Name)
	}
	if files[1].Name != "beta" {
		t.Errorf("Second file should be 'beta', got %q", files[1].Name)
	}
}

func TestReplaceFileContentExisting(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Path: "/tmp/test.html", Content: "<html>original</html>"}, "")

	app.ReplaceFileContent("/tmp/test.html", "<html>replaced</html>", "newname")

	files := app.GetFiles()
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if files[0].Content != "<html>replaced</html>" {
		t.Errorf("Content not replaced: got %q", files[0].Content)
	}

	if files[0].Name != "newname" {
		t.Errorf("Name not updated: got %q", files[0].Name)
	}
}

func TestReplaceFileContentNew(t *testing.T) {
	app := NewApp(FileEntry{Name: "existing", Path: "/tmp/existing.html", Content: "<html>existing</html>"}, "")

	app.ReplaceFileContent("/tmp/new.html", "<html>new</html>", "newfile")

	files := app.GetFiles()
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Find the new file
	var found bool
	for _, f := range files {
		if f.Path == "/tmp/new.html" {
			found = true
			if f.Content != "<html>new</html>" {
				t.Errorf("New file content wrong: got %q", f.Content)
			}
			if f.Name != "newfile" {
				t.Errorf("New file name wrong: got %q", f.Name)
			}
		}
	}
	if !found {
		t.Error("New file not found in files list")
	}
}

func TestReplaceFileContentPreservesNameIfEmpty(t *testing.T) {
	app := NewApp(FileEntry{Name: "original-name", Path: "/tmp/test.html", Content: "<html>original</html>"}, "")

	// Replace with empty name - should preserve original name
	app.ReplaceFileContent("/tmp/test.html", "<html>replaced</html>", "")

	files := app.GetFiles()
	if files[0].Name != "original-name" {
		t.Errorf("Name should be preserved when replacement name is empty, got %q", files[0].Name)
	}
}

func TestGetWindowID(t *testing.T) {
	windowID := "test-uuid-12345"
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, windowID)

	if got := app.GetWindowID(); got != windowID {
		t.Errorf("GetWindowID() = %q, want %q", got, windowID)
	}
}

func TestGetWindowIDEmpty(t *testing.T) {
	app := NewApp(FileEntry{Name: "test", Content: "<html></html>"}, "")

	if got := app.GetWindowID(); got != "" {
		t.Errorf("GetWindowID() should return empty string, got %q", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	app := NewApp(FileEntry{Name: "initial", Content: "<html></html>"}, "")

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = app.GetHTMLContent()
			_ = app.GetFiles()
			_ = app.GetCurrentIndex()
		}()
	}

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			app.AddFile(FileEntry{
				Name:    "concurrent",
				Path:    "/tmp/concurrent.html",
				Content: "<html></html>",
			})
		}()
	}

	// Concurrent selects
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = app.SelectFile(0)
		}()
	}

	wg.Wait()

	// If we get here without a race condition panic, the test passes
}

func TestSortFilesByName(t *testing.T) {
	files := []FileEntry{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "middle"},
	}

	sortFilesByName(files)

	expected := []string{"alpha", "middle", "zebra"}
	for i, name := range expected {
		if files[i].Name != name {
			t.Errorf("Position %d: got %q, want %q", i, files[i].Name, name)
		}
	}
}

func TestSortFilesByNameEmpty(t *testing.T) {
	files := []FileEntry{}
	sortFilesByName(files) // Should not panic
}

func TestSortFilesByNameSingle(t *testing.T) {
	files := []FileEntry{{Name: "single"}}
	sortFilesByName(files)

	if files[0].Name != "single" {
		t.Errorf("Single file sort changed name: got %q", files[0].Name)
	}
}

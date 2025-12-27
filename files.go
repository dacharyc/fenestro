package main

import (
	"os"
	"sort"
)

// FileEntry represents a file in the sidebar
type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // empty for stdin
	Content string `json:"content"`
}

// sortFilesByName sorts files alphabetically by name
func sortFilesByName(files []FileEntry) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
}

// isTerminal returns true if the given file is a terminal (not a pipe/redirect).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

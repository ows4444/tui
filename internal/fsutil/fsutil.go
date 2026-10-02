// Package fsutil is a shared filesystem-listing helper for widgets that
// browse directories (filepicker, treeview's DirectoryTree builder), so
// neither duplicates os.ReadDir traversal nor depends on the other. See
// decision #13.
package fsutil

import (
	"os"
	"path/filepath"
	"sort"
)

// Entry is one directory entry as returned by ListDir.
type Entry struct {
	Name  string // base name, not the full path
	IsDir bool
}

// ListDir reads the entries of dir and returns them sorted with
// directories first, then alphabetically within each group. A dir that
// can't be read (missing, no permission) returns a nil slice and the
// underlying error rather than panicking.
func ListDir(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(des))
	for i, de := range des {
		entries[i] = Entry{Name: de.Name(), IsDir: de.IsDir()}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

// HasExt reports whether name's extension (case-sensitive, including the
// leading dot, e.g. ".go") is one of exts. An empty exts always matches.
func HasExt(name string, exts []string) bool {
	if len(exts) == 0 {
		return true
	}
	ext := filepath.Ext(name)
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

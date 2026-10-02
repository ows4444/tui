package treeview

import (
	"path/filepath"

	"github.com/ows4444/tui/internal/fsutil"
)

// DirectoryTree walks the filesystem starting at root and returns a Node
// tree for New/Model, one root Node labeled with root's base name. Only
// directories are recursed into eagerly (their Children are populated up
// front, unlike Model's own lazy expand state, which is purely a
// rendering concern); files are included as leaves. An entry that can't
// be listed (permission error, or a symlink loop) is skipped rather than
// failing the whole walk.
func DirectoryTree(root string) Node {
	return Node{Label: filepath.Base(root), Children: directoryChildren(root)}
}

func directoryChildren(dir string) []Node {
	entries, err := fsutil.ListDir(dir)
	if err != nil {
		return nil
	}

	children := make([]Node, 0, len(entries))
	for _, e := range entries {
		path := filepath.Join(dir, e.Name)
		if e.IsDir {
			children = append(children, Node{Label: e.Name, Children: directoryChildren(path)})
			continue
		}
		children = append(children, Node{Label: e.Name})
	}
	return children
}

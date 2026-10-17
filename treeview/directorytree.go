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

// LazyDirectoryTree is DirectoryTree that lists a directory only when it is
// opened: each directory is a Node with a Load, so opening a big tree reads one
// directory at a time instead of the whole disk first. Pair it with a Model's
// Update, which runs the Loads and applies their LoadedMsg.
func LazyDirectoryTree(root string) Node {
	return Node{Label: filepath.Base(root), Load: func() []Node { return lazyChildren(root) }}
}

func lazyChildren(dir string) []Node {
	entries, err := fsutil.ListDir(dir)
	if err != nil {
		return nil
	}
	children := make([]Node, 0, len(entries))
	for _, e := range entries {
		path := filepath.Join(dir, e.Name)
		if e.IsDir {
			children = append(children, Node{Label: e.Name, Load: func() []Node { return lazyChildren(path) }})
			continue
		}
		children = append(children, Node{Label: e.Name})
	}
	return children
}

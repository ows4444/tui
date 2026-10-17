package main

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/treeview"
)

// item is one entry of a directory.
type item struct {
	name string
	dir  bool
	size int64
	mod  time.Time
}

// list reads dir from fsys: directories first, then files, each sorted by name
// ignoring case. Names starting with a dot are left out unless hidden is set.
func list(fsys fs.FS, dir string, hidden bool) ([]item, error) {
	ents, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var items []item
	for _, e := range ents {
		if !hidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		it := item{name: e.Name(), dir: e.IsDir()}
		if info, err := e.Info(); err == nil {
			it.size, it.mod = info.Size(), info.ModTime()
		}
		items = append(items, it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].dir != items[j].dir {
			return items[i].dir
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})
	return items, nil
}

// humanSize prints n bytes as 1.5 KiB, 3.0 MiB and so on.
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit && exp < 4; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTP"[exp])
}

// dirTree is a directory and the directories inside it, to a bounded depth.
type dirTree struct {
	name, path string
	kids       []*dirTree
}

// buildTree reads the directories below dir, at most depth levels down and at
// most *budget directories in all, so a huge tree cannot stall start-up. A
// deeper directory is still reachable from the listing; only the tree stops.
func buildTree(fsys fs.FS, dir, name string, depth int, hidden bool, budget *int) *dirTree {
	t := &dirTree{name: name, path: dir}
	if depth == 0 || *budget <= 0 {
		return t
	}
	items, err := list(fsys, dir, hidden)
	if err != nil {
		return t
	}
	for _, it := range items {
		if !it.dir || *budget <= 0 {
			continue
		}
		*budget--
		t.kids = append(t.kids, buildTree(fsys, path.Join(dir, it.name), it.name, depth-1, hidden, budget))
	}
	return t
}

// node converts the tree to treeview nodes and records, for each tree path
// ("0", "0.2", "0.2.1"), the file system path it stands for.
func (t *dirTree) node(treePath string, paths map[string]string) treeview.Node {
	paths[treePath] = t.path
	n := treeview.Node{Label: t.name}
	for i, k := range t.kids {
		n.Children = append(n.Children, k.node(fmt.Sprintf("%s.%d", treePath, i), paths))
	}
	return n
}

const previewMax = 32 << 10

// preview reads the start of the file at p for display. It returns the text,
// or a one-line description when the file is empty, binary or unreadable.
func preview(fsys fs.FS, p string, size int64) string {
	f, err := fsys.Open(p)
	if err != nil {
		return "Cannot open: " + err.Error()
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, previewMax))
	if err != nil {
		return "Cannot read: " + err.Error()
	}
	if len(b) == 0 {
		return "Empty file."
	}
	for len(b) > 0 && !utf8.Valid(b) && !utf8.FullRune(b[len(b)-1:]) {
		b = b[:len(b)-1] // a rune cut by the limit
	}
	if !utf8.Valid(b) || strings.IndexByte(string(b), 0) >= 0 {
		return fmt.Sprintf("Binary file, %s.", humanSize(size))
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = ansi.Sanitize(strings.ReplaceAll(l, "\t", "    "))
	}
	if size > previewMax {
		lines = append(lines, "", fmt.Sprintf("… %s more", humanSize(size-previewMax)))
	}
	return strings.Join(lines, "\n")
}

// crumbs splits a path into its places, with root as the first.
func crumbs(root, dir string) []string {
	parts := []string{root}
	if dir != "." {
		parts = append(parts, strings.Split(dir, "/")...)
	}
	return parts
}

// dirAt is the directory the first n places of a trail stand for.
func dirAt(parts []string, n int) string {
	if n <= 1 {
		return "."
	}
	return path.Join(parts[1:n]...)
}

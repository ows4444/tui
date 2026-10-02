package treeview

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryTree(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  Node
	}{
		{
			name:  "empty dir",
			setup: func(dir string) {},
			want:  Node{Label: "root"}, // Children nil, set below via renaming
		},
		{
			name: "flat files and dirs",
			setup: func(dir string) {
				must(t, os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o644))
				must(t, os.Mkdir(filepath.Join(dir, "a"), 0o755))
			},
			want: Node{Label: "root", Children: []Node{
				{Label: "a"},
				{Label: "b.txt"},
			}},
		},
		{
			name: "nested dir recurses",
			setup: func(dir string) {
				must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
				must(t, os.WriteFile(filepath.Join(dir, "sub", "nested.txt"), nil, 0o644))
			},
			want: Node{Label: "root", Children: []Node{
				{Label: "sub", Children: []Node{{Label: "nested.txt"}}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got := DirectoryTree(dir)
			got.Label = "root" // t.TempDir() base name is random; normalize for comparison
			if !nodeEqual(got, tt.want) {
				t.Errorf("DirectoryTree() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDirectoryTreeUnreadable(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "noperm")
	must(t, os.Mkdir(sub, 0o000))
	defer os.Chmod(sub, 0o755) // restore so TempDir cleanup can remove it

	got := DirectoryTree(dir)
	if got.Label != filepath.Base(dir) {
		t.Fatalf("DirectoryTree panicked or mislabeled: %+v", got)
	}
}

func TestDirectoryTreeFeedsModel(t *testing.T) {
	dir := t.TempDir()
	must(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))

	m := New(DirectoryTree(dir))
	view := m.View()
	if view == "" {
		t.Fatal("Model built from DirectoryTree rendered empty View()")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func nodeEqual(a, b Node) bool {
	if a.Label != b.Label || len(a.Children) != len(b.Children) {
		return false
	}
	for i := range a.Children {
		if !nodeEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}
	return true
}

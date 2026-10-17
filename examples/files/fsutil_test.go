package main

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

var stamp = time.Date(2026, 10, 1, 9, 30, 0, 0, time.UTC)

// sample is the tree every test and the golden screens use.
func sample() fs.FS {
	f := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s), ModTime: stamp} }
	return fstest.MapFS{
		"README.md":           f("# Sample\n\nA small project.\n"),
		"main.go":             f("package main\n\nfunc main() {}\n"),
		".gitignore":          f("bin/\n"),
		"go.mod":              f("module sample\n"),
		"docs/guide.md":       f("# Guide\n\nStart here.\n"),
		"docs/api/index.md":   f("# API\n"),
		"docs/api/types.md":   f("# Types\n"),
		"src/app/app.go":      f("package app\n\ttab\n"),
		"src/app/app_test.go": f("package app\n"),
		"src/util/strings.go": f("package util\n"),
		"assets/logo.bin":     {Data: []byte{0x89, 'P', 'N', 'G', 0, 1, 2}, ModTime: stamp},
		"assets/empty.txt":    {ModTime: stamp},
		"Zebra.txt":           f("zebra\n"),
		"apple.txt":           f("apple\n"),
	}
}

func TestListSortsDirectoriesFirstIgnoringCase(t *testing.T) {
	items, err := list(sample(), ".", false)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range items {
		names = append(names, it.name)
	}
	want := "assets docs src apple.txt go.mod main.go README.md Zebra.txt"
	if got := strings.Join(names, " "); got != want {
		t.Fatalf("order = %s\nwant    %s", got, want)
	}
}

func TestListHidden(t *testing.T) {
	items, _ := list(sample(), ".", true)
	if items[3].name != ".gitignore" {
		t.Fatalf("hidden file not listed first among files: %v", items)
	}
}

func TestListMissingDirectory(t *testing.T) {
	if _, err := list(sample(), "nope", false); err == nil {
		t.Fatal("no error for a missing directory")
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KiB", 1536: "1.5 KiB", 5 << 20: "5.0 MiB", 3 << 30: "3.0 GiB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestBuildTreeDepthAndBudget(t *testing.T) {
	budget := 100
	tree := buildTree(sample(), ".", "sample", 5, false, &budget)
	paths := map[string]string{}
	tree.node("0", paths)
	if paths["0"] != "." || paths["0.0"] != "assets" || paths["0.1.0"] != "docs/api" {
		t.Fatalf("paths = %v", paths)
	}
	shallow := 100
	if got := buildTree(sample(), ".", "sample", 1, false, &shallow); len(got.kids) != 3 || len(got.kids[1].kids) != 0 {
		t.Fatalf("depth 1 tree = %+v", got)
	}
	tiny := 2
	if got := buildTree(sample(), ".", "sample", 5, false, &tiny); len(got.kids) != 2 {
		t.Fatalf("budget 2 gave %d top-level directories", len(got.kids))
	}
}

func TestPreview(t *testing.T) {
	s := sample()
	if got := preview(s, "README.md", 0); !strings.HasPrefix(got, "# Sample") {
		t.Errorf("text preview = %q", got)
	}
	if got := preview(s, "assets/logo.bin", 7); got != "Binary file, 7 B." {
		t.Errorf("binary preview = %q", got)
	}
	if got := preview(s, "assets/empty.txt", 0); got != "Empty file." {
		t.Errorf("empty preview = %q", got)
	}
	if got := preview(s, "src/app/app.go", 0); !strings.Contains(got, "    tab") {
		t.Errorf("tabs not expanded: %q", got)
	}
	if got := preview(s, "missing", 0); !strings.HasPrefix(got, "Cannot open") {
		t.Errorf("missing file preview = %q", got)
	}
}

func TestPreviewStripsEscapes(t *testing.T) {
	s := fstest.MapFS{"x.txt": {Data: []byte("a\x1b[31mred\x1b[0m\n")}}
	if got := preview(s, "x.txt", 0); strings.Contains(got, "\x1b") {
		t.Fatalf("escape sequence reached the screen: %q", got)
	}
}

func TestCrumbs(t *testing.T) {
	parts := crumbs("sample", "docs/api")
	if strings.Join(parts, "|") != "sample|docs|api" {
		t.Fatalf("crumbs = %v", parts)
	}
	for n, want := range map[int]string{1: ".", 2: "docs", 3: "docs/api"} {
		if got := dirAt(parts, n); got != want {
			t.Errorf("dirAt(%d) = %q, want %q", n, got, want)
		}
	}
	if got := crumbs("sample", "."); len(got) != 1 {
		t.Fatalf("root crumbs = %v", got)
	}
}

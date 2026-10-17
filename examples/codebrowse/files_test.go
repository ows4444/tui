package main

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func TestReadDirOrdersAndHidesDotFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"b.txt": {Data: []byte("b")}, "A.txt": {Data: []byte("a")}, ".hidden": {Data: []byte("h")},
		"zdir/x": {Data: []byte("x")}, "Adir/y": {Data: []byte("y")}, ".git/config": {Data: []byte("c")},
	}
	es, _, err := readDir(fsys, ".", false)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range es {
		names = append(names, e.name)
	}
	if got := strings.Join(names, ","); got != "Adir,zdir,A.txt,b.txt" {
		t.Fatalf("order = %s", got)
	}
	es, _, _ = readDir(fsys, ".", true)
	if len(es) != 6 {
		t.Fatalf("with dot files: %d entries", len(es))
	}
	if es[0].path != ".git" || es[len(es)-1].path != "b.txt" {
		t.Fatalf("paths = %v", es)
	}
	sub, _, _ := readDir(fsys, "zdir", false)
	if len(sub) != 1 || sub[0].path != "zdir/x" {
		t.Fatalf("sub = %+v", sub)
	}
}

func TestReadDirIsBounded(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := 0; i < maxEntries+25; i++ {
		fsys[fmt.Sprintf("f%04d.txt", i)] = &fstest.MapFile{Data: []byte("x")}
	}
	es, cut, err := readDir(fsys, ".", false)
	if err != nil || len(es) != maxEntries || !cut {
		t.Fatalf("entries=%d cut=%v err=%v", len(es), cut, err)
	}
}

func TestReadDirMissing(t *testing.T) {
	if _, _, err := readDir(fstest.MapFS{}, "nope", false); err == nil {
		t.Fatal("a missing folder listed")
	}
}

func TestLanguageByName(t *testing.T) {
	for p, want := range map[string]string{
		"a/main.go": "go", "x.JSON": "json", "Makefile": "makefile", "sub/dockerfile": "dockerfile",
		"run.sh": "sh", "notes.md": "markdown", "readme": "", "data.bin": "", "style.css": "css",
	} {
		if got := language(p); got != want {
			t.Errorf("language(%q) = %q, want %q", p, got, want)
		}
	}
}

func TestLoadColoursKnownLanguages(t *testing.T) {
	fsys := fstest.MapFS{"a.go": {Data: []byte("package main\n\nfunc main() {\n\treturn\n}\n")}}
	f := load(fsys, "a.go", 36)
	if f.problem != "" || len(f.lines) != 5 || f.spans == nil || len(f.spans) != 5 {
		t.Fatalf("file = %+v", f)
	}
	if f.lines[3] != "    return" {
		t.Fatalf("tab not expanded: %q", f.lines[3])
	}
	plain := load(fstest.MapFS{"a.txt": {Data: []byte("one\ntwo")}}, "a.txt", 7)
	if plain.spans != nil || len(plain.lines) != 2 {
		t.Fatalf("plain = %+v", plain)
	}
}

func TestLoadProblems(t *testing.T) {
	fsys := fstest.MapFS{
		"empty": {Data: nil}, "bin.dat": {Data: []byte("a\x00b")}, "bad.txt": {Data: []byte{0xff, 0xfe, 'x'}},
	}
	for p, want := range map[string]string{"empty": "Empty file.", "bin.dat": "Binary file", "bad.txt": "Binary file", "missing": "Cannot open"} {
		if got := load(fsys, p, 3).problem; !strings.Contains(got, want) {
			t.Errorf("%s: problem = %q, want %q", p, got, want)
		}
	}
}

func TestLoadCutsBigFilesAndKeepsGoodUTF8(t *testing.T) {
	big := strings.Repeat("é", maxBytes) // two bytes each, so the limit falls inside a character
	f := load(fstest.MapFS{"big.txt": {Data: []byte(big)}}, "big.txt", int64(len(big)))
	if f.problem != "" || !f.truncated {
		t.Fatalf("problem=%q truncated=%v", f.problem, f.truncated)
	}
}

func TestLoadSanitisesEscapes(t *testing.T) {
	f := load(fstest.MapFS{"e.txt": {Data: []byte("a\x1b[2Jb\x1b]52;c;eA==\x07c\r\nnext\n")}}, "e.txt", 30)
	for _, l := range f.lines {
		if strings.ContainsAny(l, "\x1b\x07\r") {
			t.Fatalf("escape reached a line: %q", l)
		}
	}
	if len(f.lines) != 2 {
		t.Fatalf("lines = %q", f.lines)
	}
}

func TestFindIgnoresCase(t *testing.T) {
	f := file{lines: []string{"Alpha", "beta", "ALPHA beta", ""}}
	if got := f.find("alpha"); len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("find = %v", got)
	}
	if f.find("") != nil || len(f.find("zzz")) != 0 {
		t.Fatal("empty or missing query matched")
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KiB", 1536: "1.5 KiB", 5 << 20: "5.0 MiB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

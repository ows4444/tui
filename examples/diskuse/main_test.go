package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

// tree makes: big.bin (3000), small.txt (10), sub/ with a.dat (500) and b.dat (700).
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel string, n int) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, make([]byte, n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("big.bin", 3000)
	write("small.txt", 10)
	write("sub/a.dat", 500)
	write("sub/b.dat", 700)
	return root
}

func screen(s *tuitest.Session) string {
	return ansi.StripANSI(strings.Join(s.Screen(), "\n"))
}

func TestScanSumsDirectoriesAndSortsLargestFirst(t *testing.T) {
	ents, err := scan(tree(t))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.name)
	}
	if strings.Join(names, ",") != "big.bin,sub,small.txt" {
		t.Fatalf("order = %v", names)
	}
	if ents[1].bytes != 1200 || !ents[1].dir {
		t.Errorf("sub = %+v, want a directory of 1200 bytes", ents[1])
	}
}

func TestHuman(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KB", 1536: "1.5 KB", 5 << 20: "5.0 MB", 3 << 30: "3.0 GB"} {
		if got := human(n); got != want {
			t.Errorf("human(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestOpenGoUpAndFilter(t *testing.T) {
	root := tree(t)
	s := tuitest.New(newModel(root), 70, 14)
	defer s.Close()
	got := screen(s)
	if !strings.Contains(got, "big.bin") || !strings.Contains(got, "2.9 KB") {
		t.Fatalf("first screen:\n%s", got)
	}
	s.Keys("down", "enter") // second row: sub
	got = screen(s)
	if !strings.Contains(got, "b.dat") || !strings.Contains(got, "a.dat") || strings.Contains(got, "big.bin") {
		t.Fatalf("inside sub:\n%s", got)
	}
	s.Keys("backspace")
	if got = screen(s); !strings.Contains(got, "big.bin") {
		t.Fatalf("after going up:\n%s", got)
	}
	s.Keys("/", "s", "m", "a", "l", "l", "enter")
	got = screen(s)
	if !strings.Contains(got, "small.txt") || strings.Contains(got, "big.bin") {
		t.Errorf("filter did not narrow:\n%s", got)
	}
	s.Keys("ctrl+c")
}

func TestAnUnreadableDirectoryShowsTheError(t *testing.T) {
	m := newModel(filepath.Join(t.TempDir(), "nope"))
	if m.err == nil || !strings.Contains(m.View(), "nope") {
		t.Errorf("err %v view %q", m.err, m.View())
	}
}

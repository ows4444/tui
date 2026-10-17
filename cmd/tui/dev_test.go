package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWatchedFiles(t *testing.T) {
	for name, want := range map[string]bool{
		"main.go": true, "go.mod": true, "go.sum": true, "view.tmpl": true,
		"main_test.go": false, "README.md": false, "notes.txt": false, "image.png": false,
	} {
		if watched(name) != want {
			t.Errorf("watched(%q) = %v", name, !want)
		}
	}
}

func TestSnapshotSeesEditsAndSkipsNoise(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.go", "package a")
	write("sub/b.go", "package b")
	write("a_test.go", "package a")
	write("testdata/c.go", "package c")
	write(".git/d.go", "package d")
	before := snapshot([]string{dir})
	if len(before) != 2 {
		t.Fatalf("snapshot has %d files: %v", len(before), before)
	}
	if got := changedFiles(before, snapshot([]string{dir})); len(got) != 0 {
		t.Fatalf("nothing changed but %v differs", got)
	}
	write("a.go", "package a // edited")
	write("new.go", "package n")
	if err := os.Remove(filepath.Join(dir, "sub/b.go")); err != nil {
		t.Fatal(err)
	}
	write("a_test.go", "package a // not a source change")
	got := changedFiles(before, snapshot([]string{dir}))
	if len(got) != 3 {
		t.Fatalf("changed = %v; want the edit, the new file and the removal", got)
	}
}

func TestRunDevArguments(t *testing.T) {
	stop := make(chan struct{})
	close(stop)
	for _, args := range [][]string{{"-poll", "0"}, {"-poll", "-1s"}, {"pkg", "extra"}, {"-bogus"}} {
		if err := runDev(args, &bytes.Buffer{}, stop); err == nil {
			t.Errorf("runDev(%v) did not fail", args)
		}
	}
}

func TestDevRebuildsAndRestarts(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Go programs")
	}
	dir := t.TempDir()
	write := func(msg string) {
		t.Helper()
		src := "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"time\"\n)\n\nfunc main() {\n\tfmt.Println(\"" + msg + "\", os.Args[1:])\n\ttime.Sleep(time.Minute)\n}\n"
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module devtest\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	write("version-one")
	var out, status syncBuf
	stop := make(chan struct{})
	done := make(chan error, 1)
	cfg := devConfig{dir: dir, pkg: ".", args: []string{"-x", "y"}, watch: []string{dir}, poll: 50 * time.Millisecond,
		goCmd: "go", stdin: strings.NewReader(""), stdout: &out, status: &status}
	go func() { done <- dev(cfg, stop) }()
	waitFor := func(b *syncBuf, text string) {
		t.Helper()
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			if strings.Contains(b.String(), text) {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("never saw %q\nprogram:\n%s\ntui dev:\n%s", text, out.String(), status.String())
	}
	waitFor(&out, "version-one [-x y]")

	// A source change rebuilds and restarts.
	write("version-two!")
	waitFor(&out, "version-two! [-x y]")

	// A build that fails shows the compiler's errors and keeps waiting.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { oops }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitFor(&status, "build failed")
	if !strings.Contains(status.String(), "undefined: oops") {
		t.Errorf("the compiler's error was not shown:\n%s", status.String())
	}
	write("version-three!!")
	waitFor(&out, "version-three!! [-x y]")

	close(stop)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("dev: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("dev did not stop")
	}
}

func TestDevWaitsAfterTheProgramExits(t *testing.T) {
	if testing.Short() {
		t.Skip("builds Go programs")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module devtest\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"ran once\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, status syncBuf
	stop := make(chan struct{})
	done := make(chan error, 1)
	cfg := devConfig{dir: dir, pkg: ".", watch: []string{dir}, poll: 50 * time.Millisecond,
		goCmd: "go", stdin: strings.NewReader(""), stdout: &out, status: &status}
	go func() { done <- dev(cfg, stop) }()
	deadline := time.Now().Add(60 * time.Second)
	for !strings.Contains(status.String(), "waiting for changes") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(status.String(), "waiting for changes") {
		t.Fatalf("status:\n%s", status.String())
	}
	if strings.Count(out.String(), "ran once") != 1 {
		t.Fatalf("the program ran %d times without a change:\n%s", strings.Count(out.String(), "ran once"), out.String())
	}
	close(stop)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("dev did not stop while idle")
	}
}

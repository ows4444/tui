package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesTheProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myapp")
	var out bytes.Buffer
	if err := run([]string{"new", "-module", "example.com/me/myapp", dir}, &out); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"go.mod", "main.go", "main_test.go", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	mod, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	if !strings.HasPrefix(string(mod), "module example.com/me/myapp\n") {
		t.Errorf("go.mod = %q", mod)
	}
	src, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(src), `title.Render("myapp")`) {
		t.Errorf("main.go does not carry the name:\n%s", src)
	}
	if !strings.Contains(out.String(), "go mod tidy") {
		t.Errorf("no next steps printed: %q", out.String())
	}
}

func TestNewRefusesBadInput(t *testing.T) {
	var out bytes.Buffer
	for _, args := range [][]string{
		nil,
		{"build"},
		{"new"},
		{"new", "a", "b"},
		{"new", filepath.Join(t.TempDir(), "9lives")},
		{"new", "-module", "has space", filepath.Join(t.TempDir(), "ok")},
		{"new", "-nope", "x"},
	} {
		if err := run(args, &out); err == nil {
			t.Errorf("run(%q) did not fail", args)
		}
	}
	// A directory that already holds files is not written over.
	d := filepath.Join(t.TempDir(), "taken")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"new", d}, &out); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("non-empty dir: %v", err)
	}
}

// The scaffold builds and its own test passes, against this checkout of tui.
func TestGeneratedProjectBuildsAndPasses(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a module")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "demo")
	if err := run([]string{"new", dir}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	mod, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	mod = append(mod, []byte("\nrequire github.com/ows4444/tui v0.0.0\n\nreplace github.com/ows4444/tui => "+root+"\n")...)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), mod, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test in the generated project: %v\n%s", err, b)
	}
}

// Where the directory cannot be made, or written, the error is returned and
// nothing is half-described as created.
func TestNewReportsFilesystemErrors(t *testing.T) {
	var out bytes.Buffer
	file := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// The parent is a file, so the directory cannot be made.
	if err := run([]string{"new", filepath.Join(file, "app")}, &out); err == nil {
		t.Error("a directory under a file was made")
	}
	if strings.Contains(out.String(), "created") {
		t.Errorf("said created after failing: %q", out.String())
	}
	if os.Getuid() == 0 {
		t.Skip("root writes anywhere")
	}
	ro := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(ro, 0o755)
	if err := run([]string{"new", ro}, &out); err == nil {
		t.Error("wrote into a read-only directory")
	}
}

// Example scaffolds a program in a new directory and prints what to do next.
func Example() {
	dir := filepath.Join(os.TempDir(), "tui-example-newapp")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)
	var out bytes.Buffer
	if err := run([]string{"new", dir}, &out); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.ReplaceAll(out.String(), dir, "newapp"))
	// Output:
	// created newapp
	//
	//   cd newapp
	//   go mod tidy
	//   go run .
}

func TestCliReturnsTheExitCode(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := cli([]string{"new"}, &out, &errOut); code != 1 || !strings.Contains(errOut.String(), "usage:") {
		t.Errorf("bad args: code %d, stderr %q", code, errOut.String())
	}
	dir := filepath.Join(t.TempDir(), "okapp")
	if code := cli([]string{"new", dir}, &out, &errOut); code != 0 {
		t.Errorf("good args: code %d, stderr %q", code, errOut.String())
	}
}

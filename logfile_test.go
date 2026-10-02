package tui

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogToFileWritesOnlyToFile(t *testing.T) {
	var term bytes.Buffer
	log.SetOutput(&term)
	log.SetPrefix("")
	t.Cleanup(func() { log.SetOutput(os.Stderr); log.SetPrefix("") })

	path := filepath.Join(t.TempDir(), "x.log")
	f, err := LogToFile(path, "app: ")
	if err != nil {
		t.Fatal(err)
	}
	log.Print("hello")
	f.Close()

	if term.Len() != 0 {
		t.Fatalf("bytes leaked to terminal: %q", term.String())
	}
	b, _ := os.ReadFile(path)
	if !strings.Contains(string(b), "app: ") || !strings.Contains(string(b), "hello") {
		t.Fatalf("file = %q", b)
	}
	if _, err := LogToFile(filepath.Join(path, "nodir", "y"), ""); err == nil {
		t.Fatal("expected error for bad path")
	}
}

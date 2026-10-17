package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

func TestKeyViewerShowsDecodedEvents(t *testing.T) {
	s := tuitest.New(keyViewer{}, 90, 12)
	defer s.Close()
	s.Keys("a", "ctrl+x", "alt+left")
	s.Click(2, 3)
	s.Paste("hello")
	screen := strings.Join(s.Screen(), "\n")
	for _, want := range []string{keyViewerHeader, `key    a `, "ctrl+x", "alt+left", "mouse  button=1", "paste  5 bytes"} {
		if !strings.Contains(screen, want) {
			t.Errorf("missing %q:\n%s", want, screen)
		}
	}
	s.Keys("ctrl+q")
	if !s.Done() {
		time.Sleep(100 * time.Millisecond)
	}
	if !s.Done() {
		t.Fatal("Ctrl+Q did not quit")
	}
}

func TestKeyViewerKeepsOnlyWhatFits(t *testing.T) {
	v := keyViewer{h: 8}
	var m tui.Model = v
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "x"})
	}
	if n := len(m.(keyViewer).events); n != 6 {
		t.Fatalf("kept %d events for a window of 8 rows; want 6", n)
	}
	m, _ = m.Update(tui.FocusEvent{Focused: true})
	m, _ = m.Update(struct{}{})
	if !strings.Contains(m.View(), "focus  focused=true") {
		t.Fatal("focus event not shown")
	}
	evil := keyViewer{}
	var e tui.Model = evil
	e, _ = e.Update(tui.PasteEvent{Text: "x\x1b]52;c;ZXZpbA==\x07y"})
	if strings.Contains(e.View(), "\x1b") {
		t.Fatalf("an escape reached the view: %q", e.View())
	}
}

func writeLog(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLogTailAndGrep(t *testing.T) {
	p := writeLog(t, "a start", "b error one", "c info", "d error two", "e info")
	var out bytes.Buffer
	if err := runLog([]string{"-n", "2", p}, &out, nil); err != nil || out.String() != "d error two\ne info\n" {
		t.Fatalf("tail: %q %v", out.String(), err)
	}
	out.Reset()
	if err := runLog([]string{"-n", "-1", "-grep", "error", p}, &out, nil); err != nil || out.String() != "b error one\nd error two\n" {
		t.Fatalf("grep: %q %v", out.String(), err)
	}
	out.Reset()
	if err := runLog([]string{"-grep", "error", "-grep", "two", p}, &out, nil); err != nil || out.String() != "d error two\n" {
		t.Fatalf("two greps: %q", out.String())
	}
	out.Reset()
	if err := runLog([]string{"-n", "0", p}, &out, nil); err != nil || out.Len() != 0 {
		t.Fatalf("-n 0: %q", out.String())
	}
	if runLog(nil, &out, nil) == nil || runLog([]string{filepath.Join(t.TempDir(), "missing")}, &out, nil) == nil || runLog([]string{"-zzz", p}, &out, nil) == nil {
		t.Fatal("bad arguments must fail")
	}
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func TestLogFollowPrintsAppendedLinesAndStops(t *testing.T) {
	p := writeLog(t, "first")
	out := &syncBuf{}
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- runLog([]string{"-f", "-grep", "keep", p}, out, stop) }()
	time.Sleep(150 * time.Millisecond)
	fh, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fh.WriteString("skip this\nkeep \x1b[31mred\x1b[0m")
	time.Sleep(250 * time.Millisecond)
	_, _ = fh.WriteString(" line\n")
	fh.Close()
	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(out.String(), "line") && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	close(stop)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if got != "keep red line\n" {
		t.Fatalf("followed output %q; want the matching line once, whole, with escapes removed", got)
	}
}

func TestColorsBordersEasings(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"colors", "nord"}, &out); err != nil {
		t.Fatal(err)
	}
	plain := ansi.StripANSI(out.String())
	for _, want := range []string{"nord", "primary", "text_inverse", ":1"} {
		if !strings.Contains(plain, want) {
			t.Errorf("colors lacks %q:\n%s", want, plain)
		}
	}
	out.Reset()
	if err := run([]string{"colors"}, &out); err != nil || strings.Count(ansi.StripANSI(out.String()), "primary") < 10 {
		t.Fatalf("all presets: %v", err)
	}
	if run([]string{"colors", "nope"}, &out) == nil || run([]string{"colors", "a", "b"}, &out) == nil {
		t.Fatal("bad colors arguments")
	}
	out.Reset()
	if err := run([]string{"borders"}, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rounded", "outer-half-block", "markdown", "sample"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("borders lacks %q", want)
		}
	}
	if run([]string{"borders", "x"}, &out) == nil {
		t.Fatal("borders takes no arguments")
	}
	out.Reset()
	if err := run([]string{"easings", "out-bounce"}, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if lines[0] != "out-bounce" || len(lines) != 7 || !strings.Contains(out.String(), "*") {
		t.Fatalf("easings:\n%s", out.String())
	}
	out.Reset()
	if err := run([]string{"easings"}, &out); err != nil || strings.Count(out.String(), "  |") != 31*6 {
		t.Fatalf("all easings: %v", err)
	}
	if run([]string{"easings", "wobble"}, &out) == nil || run([]string{"easings", "a", "b"}, &out) == nil {
		t.Fatal("bad easings arguments")
	}
}

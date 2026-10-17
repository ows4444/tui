package main

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func readAll(t *testing.T, input string) []string {
	t.Helper()
	ch := make(chan item, 16)
	var no atomic.Bool
	go readLines(context.Background(), strings.NewReader(input), no.Load, ch)
	var out []string
	for it := range ch {
		if it.err != nil {
			t.Fatal(it.err)
		}
		out = append(out, it.line)
	}
	return out
}

func TestReadLinesSplitsAndKeepsTheLastPartialLine(t *testing.T) {
	for in, want := range map[string]string{
		"a\nb\nc\n": "a|b|c", "a\nb": "a|b", "": "", "\n\n": "|", "a\r\nb\r\n": "a|b",
	} {
		if got := strings.Join(readAll(t, in), "|"); got != want {
			t.Errorf("readLines(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReadLinesExpandsTabsAndStripsEscapes(t *testing.T) {
	got := readAll(t, "a\tb\n\x1b[2Jx\x1b]52;c;eA==\x07y\n")
	if got[0] != "a    b" || strings.ContainsAny(got[1], "\x1b\x07") {
		t.Fatalf("lines = %q", got)
	}
}

func TestReadLinesCutsAHugeLine(t *testing.T) {
	got := readAll(t, strings.Repeat("x", 3*maxLineBytes)+"\nafter\n")
	if len(got) != 2 || len(got[0]) > maxLineBytes+4096 || got[1] != "after" {
		t.Fatalf("%d lines, first %d bytes", len(got), len(got[0]))
	}
}

func TestFollowWaitsForMoreThenStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan item, 16)
	var follow atomic.Bool
	follow.Store(true)
	// A reader that reports EOF now and then more data later is a file being written.
	src := &growing{}
	src.add("first\n")
	go readLines(ctx, src, follow.Load, ch)
	if it := <-ch; it.line != "first" {
		t.Fatalf("first line = %q", it.line)
	}
	src.add("second\npart")
	if it := <-ch; it.line != "second" {
		t.Fatalf("second line = %q", it.line)
	}
	select {
	case it := <-ch:
		t.Fatalf("a partial line was sent before its newline: %q", it.line)
	case <-time.After(3 * pollEvery):
	}
	src.add("ial\n")
	if it := <-ch; it.line != "partial" {
		t.Fatalf("partial line = %q", it.line)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("more data after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the reader did not stop after cancel")
	}
}

// growing is a reader that returns what has been added, then io.EOF until more
// is added: the way reading a file that another program appends to behaves.
type growing struct {
	mu atomic.Pointer[[]byte]
}

func (g *growing) add(s string) {
	for {
		old := g.mu.Load()
		var cur []byte
		if old != nil {
			cur = *old
		}
		next := append(append([]byte(nil), cur...), s...)
		if g.mu.CompareAndSwap(old, &next) {
			return
		}
	}
}

func (g *growing) Read(p []byte) (int, error) {
	for {
		old := g.mu.Load()
		if old == nil || len(*old) == 0 {
			return 0, io.EOF
		}
		n := copy(p, *old)
		rest := (*old)[n:]
		if g.mu.CompareAndSwap(old, &rest) {
			return n, nil
		}
	}
}

func TestSampleIsLongAndWide(t *testing.T) {
	lines := strings.Split(sample(), "\n")
	if len(lines) < 100 {
		t.Fatalf("%d lines", len(lines))
	}
	wide := 0
	for _, l := range lines {
		if len(l) > 100 {
			wide++
		}
	}
	if wide < 50 {
		t.Fatalf("only %d wide lines", wide)
	}
}

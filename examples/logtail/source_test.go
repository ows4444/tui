package main

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func collect(t *testing.T, input string, follow bool) []string {
	t.Helper()
	ch := make(chan item, 16)
	go readLines(context.Background(), strings.NewReader(input), follow && false, ch)
	var out []string
	for it := range ch {
		if it.err != nil {
			t.Fatal(it.err)
		}
		out = append(out, it.line)
	}
	return out
}

func TestReadLines(t *testing.T) {
	for in, want := range map[string]string{"a\nb\nc\n": "a|b|c", "a\r\nb": "a|b", "": "", "\n": ""} {
		if got := strings.Join(collect(t, in, false), "|"); got != want {
			t.Errorf("readLines(%q) = %q, want %q", in, got, want)
		}
	}
	got := collect(t, "a\tb\n\x1b[2Jx\n", false)
	if got[0] != "a    b" || strings.Contains(got[1], "\x1b") {
		t.Errorf("tabs and escapes: %q", got)
	}
}

type growing struct{ mu atomic.Pointer[[]byte] }

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

func TestFollowWaitsForNewLinesAndPartialLines(t *testing.T) {
	src := &growing{}
	src.add("one\n")
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan item, 8)
	go readLines(ctx, src, true, ch)
	if it := <-ch; it.line != "one" {
		t.Fatalf("first = %q", it.line)
	}
	src.add("tw")
	select {
	case it := <-ch:
		t.Fatalf("a half line was sent: %q", it.line)
	case <-time.After(3 * pollEvery):
	}
	src.add("o\n")
	if it := <-ch; it.line != "two" {
		t.Fatalf("second = %q", it.line)
	}
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("data after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the reader did not stop")
	}
}

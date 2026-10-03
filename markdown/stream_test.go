package markdown

import (
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/theme"
)

var streamDocs = []string{
	"# Title\n\nSome *text* here.\n\nAnother paragraph\nwith two lines.\n\n```go\nx := 1\n\nfunc f() {}\n```\n\nafter",
	"- a\n- b\n\n- c\n\n  para in item\n\nplain\n\n1. x\n2. y\n\n> quote\n> more\n\n> second\n\n---\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\ntail",
	"a\r\n\r\nb\r\n\r\n~~~\r\n\r\ncode\r\n~~~\r\n\r\nc\x1b[31m red\x1b[0m\n\n\n\nd",
	"Setext\n===\n\ntext\n\n    indented\n\n- a\n\n    nested fence\n\n- ```\n\n  x\n  ```\n\nend\n\n",
}

// When the stream completes, the output equals Render on the full text; this
// checks every prefix, not only the last.
func TestStreamEqualsRenderAtEveryPrefix(t *testing.T) {
	th := theme.DarkTheme()
	for di, doc := range streamDocs {
		for _, w := range []int{8, 40} {
			s := NewStream(w, th, Options{})
			for i := 0; i < len(doc); i++ {
				s.Write(doc[i : i+1])
				if got, want := s.View(), Render(doc[:i+1], w, th); got != want {
					t.Fatalf("doc %d w %d prefix %d:\n got %q\nwant %q", di, w, i+1, got, want)
				}
			}
		}
	}
}

func TestStreamRandomChunks(t *testing.T) {
	th := theme.DarkTheme()
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 200; n++ {
		doc := strings.Join(streamDocs, "\n\n")
		s := NewStream(30, th, Options{})
		for p := 0; p < len(doc); {
			q := p + 1 + rng.Intn(25)
			if q > len(doc) {
				q = len(doc)
			}
			s.Write(doc[p:q])
			p = q
			if rng.Intn(3) == 0 {
				s.View()
			}
		}
		if got, want := s.View(), Render(doc, 30, th); got != want {
			t.Fatalf("final mismatch:\n got %q\nwant %q", got, want)
		}
	}
}

func TestStreamResizeAndReset(t *testing.T) {
	th := theme.DarkTheme()
	doc := streamDocs[0]
	s := NewStream(40, th, Options{})
	s.Write(doc)
	s.View()
	s.Resize(12, th)
	if got, want := s.View(), Render(doc, 12, th); got != want {
		t.Fatalf("after resize:\n got %q\nwant %q", got, want)
	}
	if s.Source() != doc {
		t.Fatal("Source")
	}
	s.Reset()
	if s.View() != "" || s.Source() != "" {
		t.Fatal("Reset")
	}
}

func streamTokens(n int) []string {
	var toks []string
	for i := 0; len(toks) < n; i++ {
		toks = append(toks, "word ", "*em* ", "more\n", "\n")
		if i%20 == 0 {
			toks = append(toks, "```\ncode\n```\n\n")
		}
	}
	return toks[:n]
}

func runStream(toks []string) {
	s := NewStream(80, theme.DarkTheme(), Options{})
	for _, tk := range toks {
		s.Write(tk)
		_, _ = s.Frozen(), s.Open()
	}
}

// TestStreamLinear checks that 10x the appends cost well under 100x the time: a
// linear stream gives about 10x, a quadratic one about 100x. The limit is 30x so a
// busy machine (parallel packages) cannot fail it.
func TestStreamLinear(t *testing.T) {
	if raceEnabled || testing.Short() {
		t.Skip("wall-clock scaling is unreliable under -race and -short; see BenchmarkStreamAppend1k and 10k")
	}
	if os.Getenv("TUI_TIMING_TESTS") == "" {
		t.Skip("wall-clock ratio test; set TUI_TIMING_TESTS=1 to run it")
	}
	best := func(n int) time.Duration {
		toks := streamTokens(n)
		d := time.Duration(1 << 62)
		for r := 0; r < 5; r++ {
			t0 := time.Now()
			runStream(toks)
			if e := time.Since(t0); e < d {
				d = e
			}
		}
		return d
	}
	small, big := best(1000), best(10000)
	if ratio := float64(big) / float64(small); ratio > 30 {
		t.Fatalf("10k/1k ratio %.1f > 30 (1k %v, 10k %v)", ratio, small, big)
	}
}

func BenchmarkStreamAppend1k(b *testing.B) {
	toks := streamTokens(1000)
	for i := 0; i < b.N; i++ {
		runStream(toks)
	}
}

func BenchmarkStreamAppend10k(b *testing.B) {
	toks := streamTokens(10000)
	for i := 0; i < b.N; i++ {
		runStream(toks)
	}
}

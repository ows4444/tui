package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestTextWrapWidth10NoWordSplit(t *testing.T) {
	src := "the quick brown fox jumps over the lazy dog and all of it"
	n := Text(src, WithWrap())
	sz := n.Measure(Loose(Size{W: 10, H: 50}))
	out := n.Render(Size{W: 10, H: sz.H})
	var words []string
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.Width(l); w > 10 {
			t.Fatalf("line %q is %d columns, want <= 10", l, w)
		}
		words = append(words, strings.Fields(l)...)
	}
	for _, w := range words {
		if len(w) > 10 {
			continue
		}
		if !strings.Contains(" "+src+" ", " "+w+" ") {
			t.Fatalf("word %q was split", w)
		}
	}
}

func TestTextNoWrapClipsAndEmpty(t *testing.T) {
	if got := Text("hello world").Render(Size{W: 5, H: 1}); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if s := Text("").Measure(Unconstrained()); s != (Size{}) {
		t.Fatalf("empty measures %v", s)
	}
}

func TestTextEllipsisMarksClippedLines(t *testing.T) {
	out := Text("hello world\nshort", WithEllipsis("…")).Render(Size{W: 8, H: 2})
	lines := strings.Split(out, "\n")
	if lines[0] != "hello w…" || lines[1] != "short   " {
		t.Fatalf("got %q", lines)
	}
	for _, w := range []int{1, 2, 11} {
		for _, l := range strings.Split(Text("hello world", WithEllipsis("…")).Render(Size{W: w, H: 1}), "\n") {
			if ansi.Width(l) != w {
				t.Fatalf("width %d: line %q is %d columns", w, l, ansi.Width(l))
			}
		}
	}
	if got := Text("hello world", WithEllipsis("…")).Render(Size{W: 11, H: 1}); got != "hello world" {
		t.Fatalf("a line that fits gained an ellipsis: %q", got)
	}
	// A styled line is cut without leaving an open escape.
	st := Text(ansi.NewStyle().Bold().Render("hello world"), WithEllipsis("…")).Render(Size{W: 6, H: 1})
	if ansi.Width(st) != 6 || !strings.Contains(ansi.StripANSI(st), "hello…") {
		t.Fatalf("styled: %q", st)
	}
}

func TestTextAlign(t *testing.T) {
	sz := Size{W: 9, H: 2}
	for _, c := range []struct {
		a    Align
		want string
	}{
		{AlignStart, "ab       \nabcd     "},
		{AlignCenter, "   ab    \n  abcd   "},
		{AlignEnd, "       ab\n     abcd"},
	} {
		if got := Text("ab\nabcd", WithAlign(c.a)).Render(sz); got != c.want {
			t.Fatalf("align %d: got %q want %q", c.a, got, c.want)
		}
	}
}

func TestTextDefaultsUnchanged(t *testing.T) {
	src := "hello world, this is text\nsecond"
	for _, sz := range []Size{{W: 8, H: 2}, {W: 40, H: 3}} {
		want := Text(src, WithWrap()).Render(sz)
		if got := Text(src, WithWrap(), WithAlign(AlignStart)).Render(sz); got != want {
			t.Fatalf("explicit AlignStart changed output: %q vs %q", got, want)
		}
		if plain := Text(src).Render(sz); plain != Text(src, WithAlign(AlignStart)).Render(sz) {
			t.Fatalf("plain text changed")
		}
	}
	if got := Text("hello world").Render(Size{W: 5, H: 1}); got != "hello" {
		t.Fatalf("clip without options changed: %q", got)
	}
}

func TestTextEllipsisGlyphFollowsTheCaller(t *testing.T) {
	if got := Text("hello world", WithEllipsis("~")).Render(Size{W: 6, H: 1}); got != "hello~" {
		t.Fatalf("ASCII glyph: %q", got)
	}
	if got := Text("hello world", WithEllipsis("")).Render(Size{W: 6, H: 1}); got != "hello~" {
		t.Fatalf("default glyph: %q", got)
	}
}

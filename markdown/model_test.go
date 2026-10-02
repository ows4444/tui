package markdown

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	src := "# Title\n\nSome **bold** and `code` with [a link](http://x.io).\n\n- one\n- two\n\n1. first\n2. second\n\n> quoted\n\n---\n\n```go\nx := 1\n```\n\n| A | B |\n|---|---|\n| 1 | 2 |\n"
	want := strings.Join([]string{
		"Heading level 1: Title",
		"Some bold and code with a link (http://x.io).",
		"- one",
		"- two",
		"1. first",
		"2. second",
		"Quote:",
		"  quoted",
		"Code (go)",
		"x := 1",
		"End code",
		"A: 1, B: 2",
	}, "\n")
	if got := New(src).Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := New("").Linearize(); got != "Empty document" {
		t.Errorf("empty = %q", got)
	}
	if got := New("\x1b[31mred\x1b[0m").Linearize(); strings.Contains(got, "\x1b") {
		t.Errorf("escape leaked: %q", got)
	}
}

func TestLayoutNodeFitsAnySize(t *testing.T) {
	m := New("# Hi\n\nA paragraph that wraps over several lines in a narrow slot.\n\n```\ncode\n```")
	n := m.LayoutNode(theme.DarkTheme())
	for _, s := range []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 1, H: 1}, {W: 40, H: 20}} {
		lines := strings.Split(n.Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for i, l := range lines {
			if ansi.Width(l) != s.W {
				t.Fatalf("%v: row %d is %d wide", s, i, ansi.Width(l))
			}
		}
	}
	if got := n.Render(layout.Size{}); got != "" {
		t.Errorf("zero size = %q", got)
	}
}

func TestLayoutNodeMeasureWrapsAtMaxWidth(t *testing.T) {
	m := New(strings.Repeat("word ", 40))
	n := m.LayoutNode(theme.DarkTheme())
	s := n.Measure(layout.Loose(layout.Size{W: 20, H: 100}))
	if s.W > 20 || s.H < 2 {
		t.Errorf("Measure = %+v, want wrapped within 20 cols", s)
	}
	if u := n.Measure(layout.Unconstrained()); u.W > defaultMeasureWidth {
		t.Errorf("unbounded Measure width %d > %d", u.W, defaultMeasureWidth)
	}
	if got := m.View(20, theme.DarkTheme()); got != Render(m.Source(), 20, theme.DarkTheme()) {
		t.Error("View differs from Render")
	}
}

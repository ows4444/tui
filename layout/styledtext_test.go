package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestTextWrapsToTheAllottedWidthAndStylesAfterWrapping(t *testing.T) {
	n := StyledText("the quick brown fox jumps", ansi.NewStyle().Bold(), true)
	if got := n.Measure(Loose(Size{W: 10, H: 100})); got.W > 10 || got.H < 3 {
		t.Errorf("Measure at width 10 = %v, want wrapped to at most 10 wide and 3+ rows", got)
	}
	out := n.Render(Size{W: 10, H: 5})
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Fatalf("%d rows", len(lines))
	}
	for _, l := range lines {
		if ansi.Width(l) != 10 {
			t.Errorf("row width %d: %q", ansi.Width(l), l)
		}
	}
	if !strings.Contains(lines[0], "\x1b[1m") || strings.Count(lines[0], "\x1b[0m") != 1 {
		t.Errorf("each line should be styled on its own: %q", lines[0])
	}
	if got := strings.Join(strings.Fields(ansi.StripANSI(out)), " "); got != "the quick brown fox jumps" {
		t.Errorf("text lost: %q", got)
	}
}

func TestTextCutsRowsAndHandlesEmptyAndDegenerate(t *testing.T) {
	long := StyledText("one two three four five six", ansi.Style{}, true)
	out := long.Render(Size{W: 5, H: 2})
	if strings.Count(out, "\n") != 1 {
		t.Errorf("rows not cut to 2: %q", out)
	}
	empty := StyledText("", ansi.Style{}, true)
	if got := empty.Measure(Unconstrained()); got != (Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
	if got := empty.Render(Size{W: 4, H: 2}); got != "    \n    " {
		t.Errorf("empty Render = %q", got)
	}
	if long.Render(Size{W: 0, H: 3}) != "" {
		t.Error("zero width should render empty")
	}
	noWrap := StyledText("abcdefghij", ansi.Style{}, false)
	if got := noWrap.Render(Size{W: 4, H: 1}); got != "abcd" {
		t.Errorf("no-wrap clip = %q", got)
	}
}

func TestViewIsReadEachTime(t *testing.T) {
	calls := 0
	n := ViewFunc(func() string { calls++; return "tick" })
	n.Measure(Unconstrained())
	n.Render(Size{W: 4, H: 1})
	if calls != 2 {
		t.Errorf("view function called %d times, want once per Measure and Render", calls)
	}
}

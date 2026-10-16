package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func richSample() ansi.Text {
	return ansi.NewText("plain ", ansi.Style{}).Append("bold words here", ansi.NewStyle().Bold())
}

func TestRichTextWrapsAndKeepsStyles(t *testing.T) {
	n := RichText(richSample(), true)
	if got := n.Measure(Constraints{MaxW: 10, MaxH: Unbounded}); got != (Size{W: 10, H: 2}) {
		t.Errorf("Measure at 10 = %+v", got)
	}
	out := n.Render(Size{W: 10, H: 3})
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("%d lines", len(lines))
	}
	if ansi.StripANSI(lines[0]) != "plain bold" || ansi.StripANSI(lines[1]) != "words here" {
		t.Errorf("wrapped lines = %q", ansi.StripANSI(out))
	}
	if !strings.Contains(lines[1], "\x1b[1m") {
		t.Errorf("the bold style did not carry onto the second line: %q", lines[1])
	}
	for i, l := range lines {
		if ansi.Width(l) != 10 {
			t.Errorf("line %d is %d wide, want exactly 10", i, ansi.Width(l))
		}
	}
}

func TestRichTextWithoutWrapCutsLines(t *testing.T) {
	n := RichText(ansi.NewText("abcdefgh\nxy", ansi.Style{}), false)
	if got := n.Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded}); got != (Size{W: 8, H: 2}) {
		t.Errorf("natural size = %+v", got)
	}
	if got := ansi.StripANSI(n.Render(Size{W: 4, H: 2})); got != "abcd\nxy  " {
		t.Errorf("cut = %q", got)
	}
}

func TestRichTextEmptyAndRowsAreCut(t *testing.T) {
	if got := RichText(nil, true).Measure(Constraints{MaxW: 10, MaxH: 10}); got != (Size{}) {
		t.Errorf("empty = %+v", got)
	}
	if RichText(richSample(), true).Render(Size{}) != "" {
		t.Error("a zero size drew something")
	}
	out := RichText(richSample(), true).Render(Size{W: 10, H: 1})
	if strings.Contains(out, "\n") {
		t.Errorf("rows past the height were not cut: %q", out)
	}
	// As a child of a flex row, it takes the width it is given.
	row := Row(1, FlexChild{Node: RichText(richSample(), true), BasisLen: Fr(1)}, FlexChild{Node: Block("|"), Basis: 1})
	if ansi.StripANSI(Draw(row, Constraints{MaxW: 12, MaxH: Unbounded})) == "" {
		t.Error("no output from a flex row of RichText")
	}
}

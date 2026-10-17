package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestPresetBordersAreWholeAndSameSize(t *testing.T) {
	for name, b := range map[string]Border{
		"block": BlockBorder(), "outer": OuterHalfBlockBorder(), "inner": InnerHalfBlockBorder(),
		"dashed": DashedBorder(), "heavy dashed": HeavyDashedBorder(), "markdown": MarkdownBorder(), "hidden": HiddenBorder(),
	} {
		for _, g := range []string{b.Top, b.Bottom, b.Left, b.Right, b.TopLeft, b.TopRight, b.BottomLeft, b.BottomRight} {
			if g == "" || ansi.Width(g) != 1 {
				t.Errorf("%s has a glyph %q that is not one cell", name, g)
			}
		}
		out := NewBox().Border(b).Padding(0, 1, 0, 1).render("ab")
		lines := strings.Split(out, "\n")
		if len(lines) != 3 || ansi.Width(lines[0]) != 6 || ansi.Width(lines[1]) != 6 || ansi.Width(lines[2]) != 6 {
			t.Errorf("%s: a box of 'ab' is %q; want 3 rows of 6", name, out)
		}
	}
}

func TestHiddenBorderKeepsTheSizeAndDrawsBlanks(t *testing.T) {
	plain := strings.Split(NewBox().Border(NormalBorder()).render("hi"), "\n")
	hidden := strings.Split(NewBox().Border(HiddenBorder()).render("hi"), "\n")
	if len(plain) != len(hidden) || ansi.Width(plain[0]) != ansi.Width(hidden[0]) {
		t.Fatalf("hidden %q is not the size of %q", hidden, plain)
	}
	if strings.TrimSpace(hidden[0]) != "" || strings.TrimSpace(hidden[2]) != "" || strings.TrimSpace(hidden[1]) != "hi" {
		t.Fatalf("hidden border drew something: %q", hidden)
	}
}

func TestBorderColorsPerSide(t *testing.T) {
	red, green := ansi.Red, ansi.Green
	b := NewBox().Border(NormalBorder()).BorderColor(ansi.Blue).BorderColors(red, nil, green, nil)
	lines := strings.Split(b.render("x"), "\n")
	redTop := ansi.NewStyle().Foreground(red).Render(NormalBorder().TopLeft + NormalBorder().Top + NormalBorder().TopRight)
	greenBottom := ansi.NewStyle().Foreground(green).Render(NormalBorder().BottomLeft + NormalBorder().Bottom + NormalBorder().BottomRight)
	blueSide := ansi.NewStyle().Foreground(ansi.Blue).Render(NormalBorder().Left)
	if lines[0] != redTop {
		t.Errorf("top %q; want %q", lines[0], redTop)
	}
	if lines[2] != greenBottom {
		t.Errorf("bottom %q; want %q", lines[2], greenBottom)
	}
	if !strings.HasPrefix(lines[1], blueSide) {
		t.Errorf("left side %q does not fall back to BorderColor", lines[1])
	}
}

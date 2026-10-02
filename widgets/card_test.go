package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// TestCardAlwaysRounded proves criterion #306: Card always renders with a
// rounded border, regardless of the theme's configured border style —
// tested against theme.Dark (a single-line border) and a custom theme
// with a double-line border, neither of which is rounded.
func TestCardAlwaysRounded(t *testing.T) {
	themes := []theme.Theme{theme.DarkTheme(), theme.LightTheme()}
	custom := theme.DarkTheme()
	custom.Border = layout.DoubleBorder()
	themes = append(themes, custom)

	for i, th := range themes {
		got := Card("Title", "content", th, 0)
		lines := strings.Split(got, "\n")
		if !strings.Contains(lines[0], layout.RoundedBorder().TopLeft) {
			t.Errorf("theme %d: Card top border %q should use RoundedBorder's %q corner regardless of theme.Border", i, lines[0], layout.RoundedBorder().TopLeft)
		}
		if strings.Contains(lines[0], layout.DoubleBorder().TopLeft) {
			t.Errorf("theme %d: Card top border %q should not use the theme's own border style", i, lines[0])
		}
	}
}

// TestCardTitleSeparated proves Card shares Panel's title-separation
// behavior (its differentiator from Box).
func TestCardTitleSeparated(t *testing.T) {
	got := Card("Info", "body", theme.DarkTheme(), 0)
	lines := strings.Split(got, "\n")
	titleIdx, bodyIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Info") {
			titleIdx = i
		}
		if strings.Contains(l, "body") {
			bodyIdx = i
		}
	}
	if titleIdx == -1 || bodyIdx == -1 || bodyIdx-titleIdx < 2 {
		t.Errorf("expected Card's title and body to be separated by a divider row, got title at %d, body at %d in:\n%s", titleIdx, bodyIdx, got)
	}
}

// TestCardFixedWidth proves criterion #307 for Card.
func TestCardFixedWidth(t *testing.T) {
	const width = 18
	got := Card("Card Title", "some content that needs to wrap across several lines here", theme.DarkTheme(), width)
	for i, l := range strings.Split(got, "\n") {
		if w := ansi.Width(l); w != width {
			t.Errorf("line %d width = %d, want %d; line: %q", i, w, width, l)
		}
	}
}

// TestCardEmptyContent proves criterion #308 for Card.
func TestCardEmptyContent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Card panicked on empty content: %v", r)
		}
	}()
	if got := Card("", "", theme.DarkTheme(), 0); got == "" {
		t.Errorf("Card(\"\", \"\", ...) should still render border rows, got empty string")
	}
	if got := Card("Title", "", theme.DarkTheme(), 10); got == "" {
		t.Errorf("Card with title and empty content should still render, got empty string")
	}
}

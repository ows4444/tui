package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestPanelTitleSeparated proves criterion #305: a Panel's title renders
// on its own row, styled distinctly from body text (reverse video, here),
// using the theme's colors — not just a bold line like Box.
func TestPanelTitleSeparated(t *testing.T) {
	dt := theme.DarkTheme()
	got := Panel("Info", "some body text", dt, 0)

	cw := ansi.Width("some body text") // wider than the title, so it sets the box's content width
	titleBar := ansi.NewStyle().Bold().Background(dt.Primary).Foreground(dt.TextInverse).Render(padRow("Info", cw))
	if !strings.Contains(got, titleBar) {
		t.Errorf("Panel output does not contain the expected reverse-styled title row %q\ngot:\n%s", titleBar, got)
	}

	// Body content must not carry the same reverse styling as the title.
	bodyStyled := ansi.NewStyle().Bold().Background(dt.Primary).Foreground(dt.TextInverse).Render("some body text")
	if strings.Contains(got, bodyStyled) {
		t.Errorf("Panel body text should not be styled the same as the title row")
	}
	if !strings.Contains(got, "some body text") {
		t.Errorf("Panel output does not contain body content")
	}

	// The title row and body must be on separate lines, with something
	// (a divider row) between them.
	lines := strings.Split(got, "\n")
	titleIdx, bodyIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Info") {
			titleIdx = i
		}
		if strings.Contains(l, "some body text") {
			bodyIdx = i
		}
	}
	if titleIdx == -1 || bodyIdx == -1 {
		t.Fatalf("could not locate title/body rows in Panel output:\n%s", got)
	}
	if bodyIdx-titleIdx < 2 {
		t.Errorf("expected at least one row (a divider) between title (line %d) and body (line %d)", titleIdx, bodyIdx)
	}
}

// TestPanelDefaultBorder proves Panel uses the theme's own border style,
// unlike Card.
func TestPanelDefaultBorder(t *testing.T) {
	dt := theme.LightTheme() // rounded border
	got := Panel("T", "x", dt, 0)
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[0], dt.Border.TopLeft) {
		t.Errorf("Panel top border %q should use theme.Light's rounded border %q", lines[0], dt.Border.TopLeft)
	}
}

// TestPanelFixedWidth proves criterion #307 for Panel: every output
// line's width equals the requested width, across title, divider and
// body rows.
// TestPanelUsesPlainThemeOmitsBorder proves acceptance
// criterion end-to-end, through a real widget: theme.Theme.Plain applied
// to Dark actually removes box-drawing decoration from Panel's output, not
// just from the Theme value in isolation. Uses the no-title path: Panel's
// title/body divider is deliberately independent of Border (see
// panelDivider's doc comment) and still renders even on a Plain theme, so
// only the no-title path — pure Box.Border, nothing else — demonstrates
// Plain's effect with zero ambiguity.
func TestPanelUsesPlainThemeOmitsBorder(t *testing.T) {
	dt := theme.DarkTheme()
	decorated := Panel("", "x", dt, 0)
	plain := Panel("", "x", dt.Plain(), 0)

	for _, r := range []string{dt.Border.TopLeft, dt.Border.Top, dt.Border.TopRight, dt.Border.Left, dt.Border.Right, dt.Border.BottomLeft, dt.Border.Bottom, dt.Border.BottomRight} {
		if r == "" {
			continue
		}
		if strings.Contains(plain, r) {
			t.Errorf("Panel with a Plain() theme still contains border rune %q: %q", r, plain)
		}
	}
	if !strings.Contains(decorated, dt.Border.TopLeft) {
		t.Fatalf("sanity check failed: Panel with the ordinary theme has no border rune at all: %q", decorated)
	}
}

func TestPanelFixedWidth(t *testing.T) {
	dt := theme.DarkTheme()
	const width = 24
	got := Panel("A Panel Title", "body text that is long enough to need wrapping across multiple lines", dt, width)
	for i, l := range strings.Split(got, "\n") {
		if w := ansi.Width(l); w != width {
			t.Errorf("line %d width = %d, want %d; line: %q", i, w, width, l)
		}
	}
}

// TestPanelEmptyContent proves criterion #308 for Panel.
func TestPanelEmptyContent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Panel panicked on empty content: %v", r)
		}
	}()
	if got := Panel("", "", theme.DarkTheme(), 0); got == "" {
		t.Errorf("Panel(\"\", \"\", ...) should still render border rows, got empty string")
	}
	if got := Panel("Title", "", theme.DarkTheme(), 12); got == "" {
		t.Errorf("Panel with title and empty content should still render, got empty string")
	}
}

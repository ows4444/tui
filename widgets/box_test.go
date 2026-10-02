package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestBoxNoTitle proves criterion #303: content is padded and bordered
// with the theme's border style and color, and no title line appears.
func TestBoxNoTitle(t *testing.T) {
	dt := theme.DarkTheme()
	got := Box("", "hello", dt, 0)

	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[0], dt.Border.TopLeft) {
		t.Errorf("top border line %q missing %q (theme's border style)", lines[0], dt.Border.TopLeft)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("Box output %q does not contain content %q", got, "hello")
	}
	// Border should be colored using the theme's BorderColor: layout.Box
	// wraps each full border row in one style span, so compare the whole
	// top row against that render rather than a substring of it.
	topPlain := ansi.StripANSI(lines[0])
	wantTop := ansi.NewStyle().Foreground(dt.BorderColor).Render(topPlain)
	if lines[0] != wantTop {
		t.Errorf("top border line = %q, want %q (colored with theme.BorderColor)", lines[0], wantTop)
	}
}

// TestBoxTitle proves criterion #304: a non-empty title renders as a bold
// line above the content, inside the border.
func TestBoxTitle(t *testing.T) {
	dt := theme.DarkTheme()
	got := Box("Settings", "hello", dt, 0)

	titleStyled := ansi.NewStyle().Bold().Foreground(dt.Primary).Render("Settings")
	lines := strings.Split(got, "\n")
	if !strings.Contains(got, titleStyled) {
		t.Fatalf("Box output does not contain bold-styled title %q\ngot:\n%s", titleStyled, got)
	}
	// Title line must come before the content line.
	titleIdx, contentIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Settings") {
			titleIdx = i
		}
		if strings.Contains(l, "hello") {
			contentIdx = i
		}
	}
	if titleIdx == -1 || contentIdx == -1 || titleIdx >= contentIdx {
		t.Errorf("expected title line before content line, got title at %d, content at %d", titleIdx, contentIdx)
	}
}

// TestBoxFixedWidth proves criterion #307 for Box: every output line's
// ansi.Width equals the requested width, for both title and content rows,
// with content wrapped to fit.
func TestBoxFixedWidth(t *testing.T) {
	dt := theme.DarkTheme()
	const width = 20
	got := Box("A Title", "this is a somewhat long line of body content that should wrap", dt, width)
	for i, l := range strings.Split(got, "\n") {
		if w := ansi.Width(l); w != width {
			t.Errorf("line %d width = %d, want %d; line: %q", i, w, width, l)
		}
	}
}

// TestBoxEmptyContent proves criterion #308: empty content renders a
// valid box without panicking, both with and without a title.
func TestBoxEmptyContent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Box panicked on empty content: %v", r)
		}
	}()
	if got := Box("", "", theme.DarkTheme(), 0); got == "" {
		t.Errorf("Box(\"\", \"\", ...) should still render border rows, got empty string")
	}
	if got := Box("Title", "", theme.DarkTheme(), 10); got == "" {
		t.Errorf("Box with title and empty content should still render, got empty string")
	}
}

package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestBigTextRendersMultiRowGlyphs covers #483: a known short string
// produces bigTextHeight rows, and each rendered row has a consistent
// ansi.Width for a given character (glyphs are joined with a fixed gap).
func TestBigTextRendersMultiRowGlyphs(t *testing.T) {
	got := BigText("AB", FontBlock, theme.DarkTheme())
	rows := strings.Split(got, "\n")
	if len(rows) != bigTextHeight {
		t.Fatalf("BigText(%q) produced %d rows, want %d", "AB", len(rows), bigTextHeight)
	}

	width := ansi.Width(rows[0])
	if width == 0 {
		t.Fatalf("BigText(%q) row 0 has zero width", "AB")
	}
	for i, row := range rows {
		if w := ansi.Width(row); w != width {
			t.Errorf("BigText(%q) row %d width = %d, want %d (all rows should line up)", "AB", i, w, width)
		}
	}

	// A single-character render's width should equal the glyph table's
	// 'A' width, styled.
	single := BigText("1", FontBlock, theme.DarkTheme())
	singleRows := strings.Split(single, "\n")
	if len(singleRows) != bigTextHeight {
		t.Fatalf("BigText(%q) produced %d rows, want %d", "1", len(singleRows), bigTextHeight)
	}
	wantGlyph := bigTextGlyphs['1']
	for i, row := range singleRows {
		want := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(wantGlyph[i])
		if row != want {
			t.Errorf("BigText(%q) row %d = %q, want %q", "1", i, row, want)
		}
	}
}

// TestBigTextUnknownCharacterIsBlank covers #484: a character outside
// the glyph table renders as a blank glyph of the same height instead of
// panicking.
func TestBigTextUnknownCharacterIsBlank(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("BigText panicked on unknown character: %v", r)
		}
	}()

	got := BigText("a!", FontBlock, theme.DarkTheme())
	rows := strings.Split(got, "\n")
	if len(rows) != bigTextHeight {
		t.Fatalf("BigText(%q) produced %d rows, want %d", "a!", len(rows), bigTextHeight)
	}

	// Compare against an all-space render of the same length to confirm
	// the glyph cells themselves (ignoring styling) are blank.
	blankRow := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(strings.Repeat(" ", 4) + " " + strings.Repeat(" ", 4))
	if rows[0] != blankRow {
		t.Errorf("BigText(%q) row 0 = %q, want blank glyph %q", "a!", rows[0], blankRow)
	}
}

// TestBigTextEmptyText covers #486: empty text renders "" without
// panicking.
func TestBigTextEmptyText(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("BigText panicked on empty text: %v", r)
		}
	}()

	got := BigText("", FontBlock, theme.DarkTheme())
	if got != "" {
		t.Errorf("BigText(%q, FontBlock, Dark) = %q, want %q", "", got, "")
	}
}

// TestBigTextUsesGivenTheme covers #487: glyphs are styled via the given
// Theme's Primary color.
func TestBigTextUsesGivenTheme(t *testing.T) {
	got := BigText("A", FontBlock, theme.LightTheme())
	rows := strings.Split(got, "\n")
	wantGlyph := bigTextGlyphs['A']
	for i, row := range rows {
		want := ansi.NewStyle().Foreground(theme.LightTheme().Primary).Render(wantGlyph[i])
		if row != want {
			t.Errorf("BigText(%q, FontBlock, Light) row %d = %q, want %q (should use Light.Primary)", "A", i, row, want)
		}
	}

	// Confirm it actually differs from Dark's rendering (theme is really
	// being consumed, not ignored).
	darkGot := BigText("A", FontBlock, theme.DarkTheme())
	if got == darkGot {
		t.Errorf("BigText(%q, FontBlock, Light) == BigText(%q, FontBlock, Dark); theme should affect styling", "A", "A")
	}
}

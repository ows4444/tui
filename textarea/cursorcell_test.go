package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// viewCursor finds the cursor cell by drawing: it is the first cell a row
// paints in CursorStyle, so its column is the display width of what precedes
// the style's opening escape.
func viewCursor(t *testing.T, view string, style ansi.Style) (x, y int) {
	t.Helper()
	open, _, _ := strings.Cut(style.Render("\x00"), "\x00")
	for row, line := range strings.Split(view, "\n") {
		if i := strings.Index(line, open); i >= 0 {
			return ansi.Width(line[:i]), row
		}
	}
	t.Fatalf("no cursor cell in %q", view)
	return 0, 0
}

// Criterion #103: CursorCell reports the cell View draws the cursor in, in
// every layout: plain, horizontally scrolled, soft wrapped and windowed, with
// wide and multi-rune clusters.
func TestCursorCellMatchesView(t *testing.T) {
	cases := []struct {
		name  string
		set   func(*Model)
		text  string
		moves []int // cursor rune offsets to try
	}{
		{"plain", func(m *Model) {}, "one\ntwo two\n" + family + "x\nend", []int{0, 2, 5, 12, 14, 16}},
		{"scrolled", func(m *Model) { m.Width = 6 }, "abcdefghijklmnop\nshort\n" + family + "ab" + family + "cdefgh", []int{0, 3, 10, 17, 22, 30}},
		{"wrapped", func(m *Model) { m.Width = 5; m.SoftWrap = true }, "abcdefghijkl\nxy\n" + family + family + "zz", []int{0, 4, 5, 9, 12, 13, 20}},
		{"windowed", func(m *Model) { m.Height = 2 }, "a\nbb\nccc\ndddd\neeeee", []int{0, 2, 5, 9, 15}},
		{"wrapped window", func(m *Model) { m.Width = 4; m.SoftWrap = true; m.Height = 3 }, "abcdefghijklmnop\nqrs", []int{0, 5, 9, 15, 17, 20}},
	}
	for _, c := range cases {
		m := New()
		m.CursorStyle = ansi.NewStyle().Underline()
		c.set(&m)
		m.Focus()
		m.cursorVisible = true
		m.SetValue(c.text)
		n := len([]rune(m.Value()))
		for _, pos := range c.moves {
			m.SetCursor(min(pos, n))
			m.scrollToCursor()
			x, y, ok := m.CursorCell()
			wx, wy := viewCursor(t, m.View(), m.CursorStyle)
			if !ok || x != wx || y != wy {
				t.Errorf("%s: cursor %d: CursorCell = (%d, %d, %v), View draws it at (%d, %d)", c.name, m.Cursor(), x, y, ok, wx, wy)
			}
		}
	}

	m := New()
	if _, _, ok := m.CursorCell(); ok {
		t.Fatal("an unfocused field reported a cursor")
	}
}

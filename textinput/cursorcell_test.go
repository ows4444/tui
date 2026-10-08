package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// Criterion #103: CursorCell reports the cell View draws the cursor in, with a
// prompt, wide clusters, a scrolled window and an empty value.
func TestCursorCellMatchesView(t *testing.T) {
	for _, width := range []int{0, 6} {
		for _, text := range []string{"", "abc", "abé\U0001F468\u200d\U0001F469cd中文efghijkl"} {
			m := New()
			m.Prompt = "> "
			m.Width = width
			m.CursorStyle = ansi.NewStyle().Underline()
			m.Focus()
			m.cursorVisible = true
			m.SetValue(text)
			for pos := 0; pos <= m.ed.Len(); pos++ {
				m.ed.SetCursor(pos)
				x, y, ok := m.CursorCell()
				open, _, _ := strings.Cut(m.CursorStyle.Render("\x00"), "\x00")
				v := m.View()
				i := strings.Index(v, open)
				if i < 0 {
					t.Fatalf("no cursor in %q", v)
				}
				if want := ansi.Width(v[:i]); !ok || y != 0 || x != want {
					t.Errorf("width %d %q cursor %d: CursorCell = (%d, %d, %v), View draws it at %d", width, text, pos, x, y, ok, want)
				}
			}
		}
	}
	if _, _, ok := New().CursorCell(); ok {
		t.Fatal("an unfocused field reported a cursor")
	}
}

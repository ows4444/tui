package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

func plainView(m Model) string { return ansi.StripANSI(m.View()) }

func TestLineNumbersOffByDefault(t *testing.T) {
	m := focusedWith("a\nb")
	m.Blur()
	if got := plainView(m); got != "a\nb" {
		t.Errorf("View = %q", got)
	}
	if m.gutter() != 0 {
		t.Error("gutter without ShowLineNumbers")
	}
}

func TestLineNumbersRightAlignedAndWidthGrows(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.SetValue("a\nb\nc")
	if got := plainView(m); got != "1 a\n2 b\n3 c" {
		t.Errorf("3 lines = %q", got)
	}
	m.SetValue(strings.Repeat("x\n", 9) + "x") // ten lines
	lines := strings.Split(plainView(m), "\n")
	if lines[0] != " 1 x" || lines[9] != "10 x" {
		t.Errorf("10 lines: first %q last %q", lines[0], lines[9])
	}
}

func TestLineNumbersEmptyValueShowsOne(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.Placeholder = "type here"
	if got := plainView(m); got != "1 type here" {
		t.Errorf("View = %q", got)
	}
}

func TestLineNumbersSkipWrappedRows(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.SoftWrap, m.Width = true, 4
	m.SetValue("abcdefgh\nij")
	want := "1 abcd\n  efgh\n2 ij"
	if got := strings.TrimRight(plainView(m), " "); got != want {
		t.Errorf("View =\n%q\nwant\n%q", got, want)
	}
}

func TestLineNumbersFollowTheHeightWindow(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.Height = 2
	m.Focus()
	m.SetValue("a\nb\nc\nd") // cursor at the end: window is lines 3 and 4
	if got := plainView(m); !strings.HasPrefix(got, "3 c\n4 d") {
		t.Errorf("View = %q", got)
	}
}

func TestLineNumbersShiftCursorCellAndMouse(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.Focus()
	m.SetValue("hello\nworld")
	m.SetCursorPosition(1, 3)
	if x, y, ok := m.CursorCell(); !ok || x != 2+3 || y != 1 {
		t.Errorf("CursorCell = %d,%d,%v, want 5,1", x, y, ok)
	}
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 20, H: 5}
	m, _ = m.Update(tui.MouseEvent{X: 2 + 1, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Line() != 0 || m.Column() != 1 {
		t.Errorf("click on the second text cell: line %d col %d, want 0,1", m.Line(), m.Column())
	}
	m, _ = m.Update(tui.MouseEvent{X: 0, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Line() != 1 || m.Column() != 0 {
		t.Errorf("click in the numbers: line %d col %d, want 1,0", m.Line(), m.Column())
	}
}

func TestLineNumbersDrawCellsMatchView(t *testing.T) {
	cases := map[string]func() Model{
		"plain": func() Model { m := New(); m.SetValue("a\nbb\nccc"); return m },
		"focused": func() Model {
			m := focusedWith("a\nbb\nccc")
			m.SetCursorPosition(1, 1)
			return m
		},
		"wrapped": func() Model {
			m := focusedWith("abcdefgh\nij")
			m.SoftWrap, m.Width = true, 4
			return m
		},
		"window": func() Model {
			m := focusedWith("a\nb\nc\nd\ne")
			m.Height = 2
			return m
		},
		"empty": func() Model { m := New(); m.Placeholder = "hint"; return m },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			m := build()
			m.ShowLineNumbers = true
			sameScreen(t, m, 14, 6)
		})
	}
}

func TestLineNumbersLayoutNode(t *testing.T) {
	m := New()
	m.ShowLineNumbers = true
	m.SetValue("hello\nworld")
	if sz := m.LayoutNode().Measure(layout.Unconstrained()); sz.W != 2+5 || sz.H != 2 {
		t.Errorf("Measure = %+v, want 7x2", sz)
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 9, H: 2}))
	if out != "1 hello  \n2 world  " {
		t.Errorf("Render = %q", out)
	}
}

func TestLineColumnAndSetCursorPosition(t *testing.T) {
	m := focusedWith("hello\nwörld\n")
	if m.LineCount() != 3 || m.Line() != 2 || m.Column() != 0 {
		t.Errorf("end: count %d line %d col %d", m.LineCount(), m.Line(), m.Column())
	}
	m.SetCursorPosition(1, 2)
	if m.Line() != 1 || m.Column() != 2 || m.Cursor() != 8 {
		t.Errorf("(1,2): line %d col %d cursor %d", m.Line(), m.Column(), m.Cursor())
	}
	m.SetCursorPosition(99, 99)
	if m.Line() != 2 {
		t.Errorf("clamped line = %d", m.Line())
	}
	m.SetCursorPosition(0, 99)
	if m.Line() != 0 || m.Column() != 5 {
		t.Errorf("clamped column: %d,%d", m.Line(), m.Column())
	}
	m.SetCursorPosition(-3, -3)
	if m.Cursor() != 0 {
		t.Errorf("negative: cursor %d", m.Cursor())
	}
	if New().LineCount() != 1 {
		t.Error("empty value has one line")
	}
}

func TestInsertString(t *testing.T) {
	m := focusedWith("held")
	m.SetCursorPosition(0, 3)
	m.InsertString("lo wor\r\nl")
	if m.Value() != "hello wor\nld" || m.Cursor() != 11 {
		t.Errorf("Value = %q cursor %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey)
	if m.Value() != "held" {
		t.Errorf("one undo step: %q", m.Value())
	}
	m.SetValue("0123")
	m.CharLimit = 6
	m.InsertString("abcdef")
	if m.Value() != "0123ab" {
		t.Errorf("CharLimit: %q", m.Value())
	}
	m.SetValue("abcdef")
	m.sel = 2 // anchor after "a"
	m.SetCursor(4)
	m.sel = 2
	m.InsertString("X")
	if m.Value() != "aXef" {
		t.Errorf("replaces the selection: %q", m.Value())
	}
}

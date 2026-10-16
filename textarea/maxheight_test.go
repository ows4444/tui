package textarea

import (
	"strings"
	"testing"
)

func TestMaxHeightGrowsThenScrolls(t *testing.T) {
	m := New()
	m.MaxHeight = 3
	m.Focus()
	for i, want := range []int{1, 2, 3, 3, 3} {
		if i > 0 {
			m.InsertString("\n")
		}
		m.InsertString("x")
		if got := strings.Count(plainView(m), "\n") + 1; got != want {
			t.Errorf("%d line(s) of text: View is %d rows, want %d", i+1, got, want)
		}
	}
	// Past the cap the window follows the cursor.
	m.SetCursorPosition(0, 0)
	m.scrollToCursor()
	if got := strings.Count(plainView(m), "\n") + 1; got != 3 {
		t.Errorf("window = %d rows after moving up, want 3", got)
	}
}

func TestMaxHeightIgnoredWhenHeightIsSet(t *testing.T) {
	m := New()
	m.Height, m.MaxHeight = 2, 5
	m.SetValue("a\nb\nc\nd")
	if got := strings.Count(plainView(m), "\n") + 1; got != 2 {
		t.Errorf("View is %d rows, want Height 2", got)
	}
	if New().limit() != 0 {
		t.Error("the zero Model has a limit")
	}
}

func TestMaxHeightCountsWrappedRows(t *testing.T) {
	m := New()
	m.Width, m.SoftWrap, m.MaxHeight = 4, true, 2
	m.SetValue("abcdefghij") // one line, three rows
	if got := strings.Count(plainView(m), "\n") + 1; got != 2 {
		t.Errorf("View is %d rows, want the cap 2", got)
	}
}

func TestLineInfo(t *testing.T) {
	m := New()
	m.SetValue("ab\ncd")
	m.Focus()
	m.SetCursorPosition(1, 1)
	if got := m.LineInfo(); got != (LineInfo{Line: 1, Column: 1, Row: 0, Rows: 1}) {
		t.Errorf("plain: %+v", got)
	}
	m.Width, m.SoftWrap = 4, true
	m.SetValue("abcdefghij\nxy")
	m.SetCursorPosition(0, 9)
	if got := m.LineInfo(); got != (LineInfo{Line: 0, Column: 9, Row: 2, Rows: 3}) {
		t.Errorf("wrapped: %+v", got)
	}
	m.SetCursorPosition(1, 1)
	if got := m.LineInfo(); got != (LineInfo{Line: 1, Column: 1, Row: 0, Rows: 1}) {
		t.Errorf("second line: %+v", got)
	}
	if got := New().LineInfo(); got != (LineInfo{Rows: 1}) {
		t.Errorf("empty: %+v", got)
	}
}

func TestLineLabelReplacesTheNumberAndSizesTheColumn(t *testing.T) {
	m := New()
	m.LineLabel = func(line int, cont bool) string {
		if line == 2 {
			return ">"
		}
		return "·"
	}
	m.SetValue("a\nb\nc")
	if got := plainView(m); got != "· a\n> b\n· c" {
		t.Errorf("View = %q", got)
	}
	if m.gutter() != 2 {
		t.Errorf("gutter = %d, want 2", m.gutter())
	}
	// A label wider than the column (taken from the first and last line) is cut.
	m.LineLabel = func(line int, cont bool) string {
		if line == 2 {
			return "middle"
		}
		return "ab"
	}
	if got := plainView(m); got != "ab a\nmi b\nab c" {
		t.Errorf("long label: View = %q", got)
	}
}

package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func aidArea(value string, w, h int) Model {
	m := New()
	m.Width = w
	m.SetValue(value)
	m.Focus()
	return m
}

// grid renders m through View and returns it as cells.
func grid(t *testing.T, m Model, w, h int) *cellbuf.Buffer {
	t.Helper()
	b := cellbuf.New(w, h)
	rows := strings.Split(m.View(), "\n")
	for y, row := range rows {
		if y >= h {
			break
		}
		if _, err := b.SetStyled(0, y, row); err != nil {
			t.Fatalf("row %d: %v", y, err)
		}
	}
	return b
}

func bg(b *cellbuf.Buffer, x, y int) cellbuf.Color { return b.Style(b.At(x, y).Style).BG }

func wantBG(c ansi.Color) cellbuf.Color { return cellStyle(ansi.NewStyle().Background(c)).BG }

// sameAsView fails t unless DrawCells draws what View shows, cell by cell.
func sameAsView(t *testing.T, m Model, w, h int) {
	t.Helper()
	want := grid(t, m, w, h)
	got := cellbuf.New(w, h)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g, e := got.At(x, y), want.At(x, y)
			if g.Cluster != e.Cluster || got.Style(g.Style) != want.Style(e.Style) {
				t.Fatalf("cell (%d,%d): DrawCells %q %+v, View %q %+v\n%s", x, y,
					g.Cluster, got.Style(g.Style), e.Cluster, want.Style(e.Style), m.View())
			}
		}
	}
}

func TestCursorLineIsHighlightedAcrossTheWidth(t *testing.T) {
	m := aidArea("one\ntwo\nthree", 10, 3)
	m.HighlightCursorLine = true
	m.SetCursorPosition(1, 1) // on "two"
	b := grid(t, m, 10, 3)
	on := wantBG(defaultCursorLine)
	for x := 0; x < 10; x++ {
		if x != 1 && bg(b, x, 1) != on { // x=1 is the cursor cell
			t.Errorf("cell (%d,1) is not on the cursor line background", x)
		}
	}
	if bg(b, 0, 0) == on || bg(b, 2, 2) == on {
		t.Error("another line was highlighted")
	}
	m.CursorLineColor = ansi.RGB{R: 10, G: 20, B: 30}
	if got := grid(t, m, 10, 3); bg(got, 0, 1) != wantBG(ansi.RGB{R: 10, G: 20, B: 30}) {
		t.Error("CursorLineColor was ignored")
	}
}

func TestBracketsMatchBothWays(t *testing.T) {
	plain := aidArea("f(a[1], {b})", 0, 1)
	reversed := func(m Model) map[int]bool {
		b := grid(t, m, 20, 1)
		out := map[int]bool{}
		base := b.Style(b.At(0, 0).Style)
		for x := 0; x < 12; x++ {
			if b.Style(b.At(x, 0).Style) != base {
				out[x] = true
			}
		}
		return out
	}
	for name, tc := range map[string]struct {
		cursor int
		want   []int
	}{
		"on an opening bracket":    {1, []int{1, 11}},
		"on a closing bracket":     {11, []int{1, 11}},
		"just after a closing one": {12, []int{1, 11}},
		"on a square bracket":      {3, []int{3, 5}},
		"on a brace":               {8, []int{8, 10}},
		"in plain text":            {0, nil},
	} {
		m := plain
		m.MatchBrackets = true
		m.SetCursorPosition(0, tc.cursor)
		m.focused = false // keep the cursor cell out of the comparison
		got := reversed(m)
		for _, x := range tc.want {
			if !got[x] {
				t.Errorf("%s: column %d is not marked (marked: %v)", name, x, got)
			}
		}
		if len(got) != len(tc.want) {
			t.Errorf("%s: marked %v, want only %v", name, got, tc.want)
		}
	}
	nested := aidArea("(a(b)c)", 0, 1)
	nested.MatchBrackets, nested.focused = true, false
	nested.SetCursorPosition(0, 2)
	if got := reversed(nested); !got[2] || !got[4] || len(got) != 2 {
		t.Errorf("nested pair marked %v, want 2 and 4", got)
	}
	unmatched := aidArea("(a", 0, 1)
	unmatched.MatchBrackets, unmatched.focused = true, false
	unmatched.SetCursorPosition(0, 0)
	if got := reversed(unmatched); len(got) != 0 {
		t.Errorf("an unmatched bracket marked %v", got)
	}
}

func TestBracketSearchGivesUp(t *testing.T) {
	m := aidArea("("+strings.Repeat("x", bracketScanLimit+10)+")", 0, 1)
	m.MatchBrackets = true
	m.SetCursorPosition(0, 0)
	if _, _, ok := m.bracketPair(); ok {
		t.Error("the search went past its limit")
	}
}

func TestShowWhitespaceDrawsSpacesAsMarks(t *testing.T) {
	m := aidArea("a b  c", 0, 1)
	m.ShowWhitespace = true
	m.focused = false
	if got := ansi.StripANSI(m.View()); got != "a·b··c" {
		t.Errorf("View = %q", got)
	}
	m.WhitespaceMark = "."
	if got := ansi.StripANSI(m.View()); got != "a.b..c" {
		t.Errorf("custom mark: %q", got)
	}
	m.focused = true
	m.SetCursorPosition(0, 1) // on the first space: the cursor cell wins
	if got := ansi.StripANSI(m.View()); !strings.HasPrefix(got, "a ") {
		t.Errorf("under the cursor: %q", got)
	}
	m.ShowWhitespace = false
	if got := ansi.StripANSI(m.View()); got != "a b  c" {
		t.Errorf("off: %q", got)
	}
}

func TestRulerMarksAColumnOnEveryRow(t *testing.T) {
	m := aidArea("ab\nabcdefgh", 0, 2)
	m.Ruler = []int{5}
	m.focused = false
	b := grid(t, m, 10, 2)
	on := wantBG(defaultRuler)
	for y := 0; y < 2; y++ {
		if bg(b, 4, y) != on {
			t.Errorf("row %d: column 5 is not the ruler", y)
		}
		if bg(b, 3, y) == on || bg(b, 5, y) == on {
			t.Errorf("row %d: the ruler is wider than one column", y)
		}
	}
	if got := ansi.StripANSI(m.View()); strings.Split(got, "\n")[0] != "ab   " {
		t.Errorf("a short row is padded to the ruler: %q", got)
	}
	m.Ruler = []int{0, -3}
	if got := m.View(); strings.Contains(got, "48;") {
		t.Errorf("an invalid ruler column drew something: %q", got)
	}
}

func TestAidsWorkWithASelectionAndAHighlighter(t *testing.T) {
	m := aidArea("let x = (1 + 2)", 0, 1)
	m.Highlighter = func(line string) ansi.Text {
		return ansi.NewText(line, ansi.NewStyle().Foreground(ansi.Green))
	}
	m.MatchBrackets, m.ShowWhitespace, m.HighlightCursorLine = true, true, true
	m.SetCursorPosition(0, 8)
	m.sel = 1 // select [0, cursor)
	sameAsView(t, m, 30, 1)
	if got := ansi.StripANSI(m.View()); !strings.Contains(got, "(1·+·2)") || !strings.HasPrefix(got, "let x =") {
		t.Errorf("View = %q", got)
	}
}

func TestDrawCellsMatchesViewWithAids(t *testing.T) {
	text := "func main() {\n\tfmt.Println(\"hi\")\n}\n\nshort"
	for name, set := range map[string]func(*Model){
		"cursor line": func(m *Model) { m.HighlightCursorLine = true },
		"brackets":    func(m *Model) { m.MatchBrackets = true },
		"whitespace":  func(m *Model) { m.ShowWhitespace = true },
		"ruler":       func(m *Model) { m.Ruler = []int{4, 10} },
		"all": func(m *Model) {
			m.HighlightCursorLine, m.MatchBrackets, m.ShowWhitespace, m.Ruler = true, true, true, []int{6}
		},
		"soft wrap":     func(m *Model) { m.SoftWrap = true; m.HighlightCursorLine = true; m.Ruler = []int{5} },
		"height window": func(m *Model) { m.Height = 2; m.HighlightCursorLine = true; m.MatchBrackets = true },
		"unfocused":     func(m *Model) { m.Blur(); m.HighlightCursorLine = true; m.ShowWhitespace = true },
		"wide text":     func(m *Model) { m.SetValue("日本 語 x"); m.ShowWhitespace = true; m.Ruler = []int{4} },
	} {
		t.Run(name, func(t *testing.T) {
			m := aidArea(strings.ReplaceAll(text, "\t", "  "), 12, 0)
			m.SetCursorPosition(0, 12)
			set(&m)
			sameAsView(t, m, 12, 8)
		})
	}
}

func TestNoAidsLeavesViewAlone(t *testing.T) {
	m := aidArea("a b\n(c)", 0, 2)
	before := m.View()
	m.Ruler = nil
	if m.View() != before || m.aidsOn() {
		t.Error("an empty configuration changed the view")
	}
}

func TestCursorLineCoversEveryRowOfAWrappedLine(t *testing.T) {
	m := aidArea("abcdefghij\nxyz", 5, 4)
	m.SoftWrap, m.HighlightCursorLine = true, true
	m.SetCursorPosition(0, 1) // on the first visual row of the first logical line
	b := grid(t, m, 5, 3)
	on := wantBG(defaultCursorLine)
	if bg(b, 3, 0) != on || bg(b, 2, 1) != on {
		t.Error("both visual rows of the cursor's line should be highlighted")
	}
	if bg(b, 0, 2) == on {
		t.Error("the next line was highlighted")
	}
}

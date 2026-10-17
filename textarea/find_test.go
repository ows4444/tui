package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func withText(s string) Model {
	m := New()
	m.SetValue(s)
	m.SetCursor(0)
	m.Focus()
	return m
}

func TestFindSelectsAndWraps(t *testing.T) {
	m := withText("Foo bar foo\nFOO")
	if n := m.Find("foo"); n != 3 {
		t.Fatalf("Find counted %d; want 3", n)
	}
	if lo, hi, ok := m.Selection(); !ok || lo != 0 || hi != 3 {
		t.Fatalf("first match: %d..%d %v", lo, hi, ok)
	}
	for _, want := range []int{8, 12, 0} { // the wrap goes back to the first
		if !m.FindNext("foo") {
			t.Fatal("FindNext found nothing")
		}
		if lo, _, _ := m.Selection(); lo != want {
			t.Fatalf("FindNext went to %d; want %d", lo, want)
		}
	}
	if !m.FindPrev("foo") {
		t.Fatal("FindPrev found nothing")
	}
	if lo, _, _ := m.Selection(); lo != 12 {
		t.Fatalf("FindPrev wrapped to %d; want 12", lo)
	}
	if m.SelectedText() != "FOO" {
		t.Fatalf("selected %q", m.SelectedText())
	}
	if m.Cursor() != 15 {
		t.Fatalf("cursor %d; want it at the end of the match", m.Cursor())
	}
}

func TestFindStartsAtTheCursor(t *testing.T) {
	m := withText("ab ab ab")
	m.SetCursor(4)
	m.Find("ab")
	if lo, _, _ := m.Selection(); lo != 6 {
		t.Fatalf("selected from %d; want the first match at or after the cursor, 6", lo)
	}
}

func TestFindNothing(t *testing.T) {
	m := withText("abc")
	m.SetCursor(2)
	if m.Find("") != 0 || m.Find("zz") != 0 || m.FindNext("zz") || m.FindPrev("") {
		t.Fatal("found text that is not there")
	}
	if _, _, ok := m.Selection(); ok || m.Cursor() != 2 {
		t.Fatal("a failed Find moved things")
	}
}

func TestFindOnWideText(t *testing.T) {
	m := withText("世界 世界")
	if m.Find("界") != 2 {
		t.Fatal("wrong count over wide runes")
	}
	m.FindNext("界")
	if lo, hi, _ := m.Selection(); lo != 4 || hi != 5 {
		t.Fatalf("selection %d..%d; rune offsets expected", lo, hi)
	}
}

func TestReplaceAllIsOneUndo(t *testing.T) {
	m := withText("cat Cat\ndog CAT")
	m.SetCursor(len("cat Cat\ndog CAT"))
	n := m.ReplaceAll("cat", "tiger")
	if n != 3 || m.Value() != "tiger tiger\ndog tiger" {
		t.Fatalf("replaced %d: %q", n, m.Value())
	}
	// The cursor was at the end of the text, after all three matches.
	if m.Cursor() != len(m.Value()) {
		t.Fatalf("cursor %d; want the end, %d", m.Cursor(), len(m.Value()))
	}
	press(&m, tui.Key{Type: tui.KeyCtrl, Code: 'z'})
	if m.Value() != "cat Cat\ndog CAT" {
		t.Fatalf("one undo should restore everything, got %q", m.Value())
	}
	if m.ReplaceAll("zzz", "x") != 0 || m.ReplaceAll("", "x") != 0 {
		t.Fatal("replaced something that is not there")
	}
}

func TestReplaceAllWithNewlinesAndEmpty(t *testing.T) {
	m := withText("a,b,c")
	m.ReplaceAll(",", "\n")
	if m.Value() != "a\nb\nc" || m.LineCount() != 3 {
		t.Fatalf("%q", m.Value())
	}
	m.ReplaceAll("\n", "")
	if m.Value() != "abc" {
		t.Fatalf("%q", m.Value())
	}
}

func TestAutoIndent(t *testing.T) {
	m := withText("    one\n\ttwo")
	m.AutoIndent = true
	m.SetCursor(7) // end of line 1
	press(&m, key(tui.KeyEnter))
	if m.Value() != "    one\n    \n\ttwo" || m.Cursor() != 12 {
		t.Fatalf("%q cursor %d", m.Value(), m.Cursor())
	}
	m.SetCursor(len("    one\n    \n\ttwo"))
	press(&m, key(tui.KeyEnter))
	if !strings.HasSuffix(m.Value(), "\ttwo\n\t") {
		t.Fatalf("tab indent not kept: %q", m.Value())
	}
	// Enter inside the indent copies only what is left of the cursor.
	m2 := withText("      x")
	m2.AutoIndent = true
	m2.SetCursor(2)
	press(&m2, key(tui.KeyEnter))
	// the new line starts with the two spaces before the cursor, then the rest
	if m2.Value() != "  \n      x" {
		t.Fatalf("%q", m2.Value())
	}
	off := withText("  a")
	off.SetCursor(3)
	press(&off, key(tui.KeyEnter))
	if off.Value() != "  a\n" {
		t.Fatalf("AutoIndent off must not indent: %q", off.Value())
	}
}

// highlight colours the word "key" red and leaves the rest unstyled.
func highlight(line string) ansi.Text {
	var t ansi.Text
	for len(line) > 0 {
		i := strings.Index(line, "key")
		if i < 0 {
			return t.Append(line, ansi.Style{})
		}
		t = t.Append(line[:i], ansi.Style{}).Append("key", ansi.NewStyle().Foreground(ansi.Red))
		line = line[i+3:]
	}
	return t
}

func TestHighlighterColoursView(t *testing.T) {
	m := withText("a key\nplain")
	m.Highlighter = highlight
	view := m.View()
	red := ansi.NewStyle().Foreground(ansi.Red).Render("k")
	if !strings.Contains(view, red) {
		t.Fatalf("no red in %q", view)
	}
	if strings.Contains(strings.Split(view, "\n")[1], "\x1b[31m") {
		t.Fatal("the plain line was coloured")
	}
	if ansi.StripANSI(view) != "a key\nplain" {
		t.Fatalf("text changed: %q", ansi.StripANSI(view))
	}
}

func TestHighlighterDrawCellsMatchesView(t *testing.T) {
	cases := map[string]func(*Model){
		"plain":    func(m *Model) {},
		"wrapped":  func(m *Model) { m.Width = 6; m.SoftWrap = true },
		"scrolled": func(m *Model) { m.Width = 6 },
		"height":   func(m *Model) { m.Height = 2 },
		"selected": func(m *Model) { m.SelectAll() },
		"numbered": func(m *Model) { m.ShowLineNumbers = true },
	}
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			m := withText("a key and a key\nkeykey\n世 key")
			m.Highlighter = highlight
			m.Focus()
			set(&m)
			sameScreen(t, m, 20, 10)
		})
	}
}

func TestHighlighterThatLiesIsIgnored(t *testing.T) {
	m := withText("abc")
	m.Highlighter = func(string) ansi.Text {
		return ansi.NewText("longer than the line", ansi.NewStyle().Foreground(ansi.Red))
	}
	if strings.Contains(m.View(), "\x1b[31m") || ansi.StripANSI(m.View()) != "abc" {
		t.Fatalf("a bad Highlighter changed the view: %q", m.View())
	}
}

func TestNoHighlighterIsUnchanged(t *testing.T) {
	a, b := withText("a key\nplain"), withText("a key\nplain")
	b.Highlighter = func(line string) ansi.Text { return ansi.NewText(line, ansi.Style{}) }
	if ansi.StripANSI(a.View()) != ansi.StripANSI(b.View()) {
		t.Fatal("an identity Highlighter changed the text")
	}
}

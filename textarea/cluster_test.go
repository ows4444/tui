package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

const family = "\U0001F468‍\U0001F469‍\U0001F467" // ZWJ family, one cluster

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// Criterion #39: wide characters render within Width columns.
func TestWideCharsStayWithinWidth(t *testing.T) {
	m := focused()
	m.Width = 10
	m.SetValue(strings.Repeat("你", 12) + "\nab")
	for _, pos := range []int{0, 6, 12} {
		m.SetCursor(pos)
		for i, l := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(l); w > 10 {
				t.Errorf("cursor %d row %d is %d columns, want <= 10: %q", pos, i, w, l)
			}
		}
	}
}

// Criterion #39: Backspace, Delete, Left and Right act on whole clusters.
func TestClusterEditing(t *testing.T) {
	m := New()
	m.Focus()
	m.SetValue("a" + family + "é")
	press(&m, key(tui.KeyBackspace))
	if m.Value() != "a"+family {
		t.Fatalf("Backspace over e+combining: %q", m.Value())
	}
	press(&m, key(tui.KeyBackspace))
	if m.Value() != "a" {
		t.Fatalf("Backspace over ZWJ family left %q", m.Value())
	}
	m.SetValue("a" + family + "b")
	m.SetCursor(1)
	press(&m, key(tui.KeyRight))
	if m.Cursor() != 1+len([]rune(family)) {
		t.Fatalf("Right over family: cursor %d", m.Cursor())
	}
	press(&m, key(tui.KeyLeft))
	press(&m, key(tui.KeyDelete))
	if m.Value() != "ab" {
		t.Fatalf("Delete over family left %q", m.Value())
	}
	// A cursor set inside a cluster snaps to its start.
	m.SetValue("a" + family)
	m.SetCursor(3)
	if m.Cursor() != 1 {
		t.Fatalf("SetCursor inside a cluster = %d, want 1", m.Cursor())
	}
}

// Criterion #39: the cursor cluster is drawn as one unit and a wide-character
// column drives vertical movement.
func TestVerticalMoveKeepsDisplayColumn(t *testing.T) {
	m := focused()
	m.SetValue("你好ab\nabcdef")
	m.SetCursor(2) // after 你好: column 4
	press(&m, key(tui.KeyDown))
	if m.Cursor() != 5+4 { // "你好ab\n" is 5 runes; 4 columns in
		t.Fatalf("Down: cursor %d, want 9", m.Cursor())
	}
	press(&m, key(tui.KeyUp))
	if m.Cursor() != 2 {
		t.Fatalf("Up: cursor %d, want 2", m.Cursor())
	}
	m.SetValue("你好\nabc")
	m.SetCursor(4) // rune 4 is 'b' (column 1)
	press(&m, key(tui.KeyUp))
	if m.Cursor() != 0 { // column 1 falls inside 你: stay before it
		t.Fatalf("Up into a wide cluster: cursor %d, want 0", m.Cursor())
	}
}

func TestWordJump(t *testing.T) {
	m := focused()
	m.SetValue("foo bar baz")
	press(&m, tui.Key{Type: tui.KeyLeft, Mod: input.ModAlt})
	if m.Cursor() != 8 {
		t.Fatalf("Alt+Left: %d, want 8", m.Cursor())
	}
	m.SetCursor(0)
	press(&m, tui.Key{Type: tui.KeyRight, Mod: input.ModCtrl})
	if m.Cursor() != 3 {
		t.Fatalf("Ctrl+Right: %d, want 3", m.Cursor())
	}
}

// Criterion #40: a vertical move does not touch the whole buffer, so its
// cost is the same for a huge buffer as for a tiny one.
func TestVerticalMoveDoesNotScaleWithBuffer(t *testing.T) {
	cost := func(lines int) float64 {
		m := New()
		m.SetValue(strings.Repeat("the quick brown fox\n", lines))
		m.SetCursor(m.value.len() / 2)
		return testing.AllocsPerRun(50, func() { m.moveVertical(1); m.moveVertical(-1) })
	}
	small, big := cost(10), cost(100000)
	if big > small+2 {
		t.Fatalf("moveVertical allocs: %v at 10 lines, %v at 100,000", small, big)
	}
	m := New()
	m.SetValue("abc\nde\nfghij")
	m.SetCursor(3)
	m.moveVertical(1)
	m.moveVertical(1)
	if m.Cursor() != 9 { // "de" clamps to its end (6); fghij then column 2 -> 6+... offset 9
		t.Fatalf("cursor %d, want 9", m.Cursor())
	}
}

func rows(m Model) []string {
	m.focused = false
	m.SetCursor(0)
	return strings.Split(ansi.StripANSI(m.View()), "\n")
}

// Criterion #41: with SoftWrap, long lines wrap at column width.
func TestSoftWrapWrapsAtColumnWidth(t *testing.T) {
	m := New()
	m.SoftWrap = true
	m.Width = 5
	m.SetValue("abcdefghijkl\n你好世界x")
	got := rows(m)
	want := []string{"abcde", "fghij", "kl", "你好", "世界x"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("rows %q, want %q", got, want)
	}
	m.SoftWrap = false
	for _, l := range rows(m) {
		if ansi.Width(l) > 5 {
			t.Fatalf("scrolling row too wide: %q", l)
		}
	}
	if len(rows(m)) != 2 {
		t.Fatalf("without SoftWrap the lines must not wrap: %q", rows(m))
	}
}

// Criterion #41: Up and Down follow visual rows.
func TestSoftWrapCursorFollowsVisualRows(t *testing.T) {
	m := focused()
	m.SoftWrap = true
	m.Width = 5
	m.SetValue("abcdefghijkl\nxy")
	m.SetCursor(7) // 'h', row 1, col 2
	press(&m, key(tui.KeyUp))
	if m.Cursor() != 2 {
		t.Fatalf("Up: %d, want 2", m.Cursor())
	}
	press(&m, key(tui.KeyDown))
	press(&m, key(tui.KeyDown))
	if m.Cursor() != 12 { // row 2 is "kl": col 2 -> end (12)
		t.Fatalf("Down to short row: %d", m.Cursor())
	}
	press(&m, key(tui.KeyDown))
	if m.Cursor() != 13+2 {
		t.Fatalf("Down to next line: %d, want 15", m.Cursor())
	}
	press(&m, key(tui.KeyUp))
	if m.Cursor() != 12 { // back onto last row of line 0, col 2 -> 'kl' end
		t.Fatalf("Up to previous line's last row: %d, want 12", m.Cursor())
	}
}

// Criterion #41: an end-of-line cursor on a full row gets its own row, and
// Height windows visual rows.
func TestSoftWrapEndCursorAndHeight(t *testing.T) {
	m := focused()
	m.SoftWrap = true
	m.Width = 4
	m.SetValue("abcdefgh") // cursor at end, last row full
	if got := len(strings.Split(m.View(), "\n")); got != 3 {
		t.Fatalf("%d rows, want 3 (cursor row)", got)
	}
	m.Height = 2
	m.SetValue("abcdefghijklmnop")
	out := strings.Split(ansi.StripANSI(m.View()), "\n")
	if len(out) != 2 || out[0] != "mnop" || out[1] != " " {
		t.Fatalf("windowed rows %q", out)
	}
	for _, l := range strings.Split(m.View(), "\n") {
		if ansi.Width(l) > 4 {
			t.Fatalf("row wider than Width: %q", l)
		}
	}
}

func pressRunes(m *Model, s string) {
	press(m, tui.Key{Type: tui.KeyRunes, Text: s})
}

// Criterion #39: CharLimit counts grapheme clusters, as textinput's does, so a
// wide or multi-rune cluster is one unit and is never cut in half.
func TestCharLimitCountsClusters(t *testing.T) {
	m := New()
	m.CharLimit = 3
	m.SetValue("a" + family + "b" + "c")
	if m.Value() != "a"+family+"b" {
		t.Fatalf("SetValue kept %q, want the first 3 clusters", m.Value())
	}

	m = New()
	m.CharLimit = 2
	m.Focus()
	pressRunes(&m, "é"+family+"x") // e+combining, family, then one more
	if m.Value() != "é"+family {
		t.Fatalf("typed value %q, want 2 clusters", m.Value())
	}

	// A combining mark joins the cluster before the cursor, so it fits at the limit.
	m.SetValue("ab")
	pressRunes(&m, "́")
	if m.Value() != "ab́" {
		t.Fatalf("combining mark at the limit: %q", m.Value())
	}
	pressRunes(&m, "c")
	if m.Value() != "ab́" {
		t.Fatalf("a new cluster passed the limit: %q", m.Value())
	}

	// Paste counts clusters too, and newline is one.
	m.SetValue("")
	m, _ = m.Update(tui.PasteEvent{Text: family + "\n" + family})
	if m.Value() != family+"\n" {
		t.Fatalf("paste kept %q, want family and newline", m.Value())
	}
}

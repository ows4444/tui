package viewport

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func numberedLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = strconv.Itoa(i)
	}
	return strings.Join(lines, "\n")
}

func TestViewShowsFirstHeightLinesInitially(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10)) // "0".."9"
	got := m.View()
	want := "0\n1\n2"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewPadsShortContentToHeight(t *testing.T) {
	m := New(10, 5)
	m.SetContent("a\nb")
	got := m.View()
	want := "a\nb\n\n\n"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestLineDownAndLineUp(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))

	m.LineDown(2)
	if got := m.View(); got != "2\n3\n4" {
		t.Errorf("after LineDown(2): View() = %q, want %q", got, "2\n3\n4")
	}

	m.LineUp(1)
	if got := m.View(); got != "1\n2\n3" {
		t.Errorf("after LineUp(1): View() = %q, want %q", got, "1\n2\n3")
	}
}

func TestScrollClampsAtTopAndBottom(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10)) // maxYOffset = 10-3 = 7

	m.LineUp(100)
	if !m.AtTop() {
		t.Error("LineUp(100) from the top should still be AtTop")
	}
	if got := m.View(); got != "0\n1\n2" {
		t.Errorf("View() = %q, want %q", got, "0\n1\n2")
	}

	m.LineDown(1000)
	if !m.AtBottom() {
		t.Error("LineDown(1000) should land AtBottom")
	}
	if got := m.View(); got != "7\n8\n9" {
		t.Errorf("View() = %q, want %q", got, "7\n8\n9")
	}
}

func TestPageUpDownAndHalfPage(t *testing.T) {
	m := New(10, 4)
	m.SetContent(numberedLines(20)) // maxYOffset = 16

	m.PageDown() // +4
	if got := m.View(); got != "4\n5\n6\n7" {
		t.Errorf("after PageDown: View() = %q, want %q", got, "4\n5\n6\n7")
	}
	m.HalfPageDown() // +2
	if got := m.View(); got != "6\n7\n8\n9" {
		t.Errorf("after HalfPageDown: View() = %q, want %q", got, "6\n7\n8\n9")
	}
	m.HalfPageUp() // -2
	m.PageUp()     // -4
	if got := m.View(); got != "0\n1\n2\n3" {
		t.Errorf("after PageUp+HalfPageUp back to top: View() = %q, want %q", got, "0\n1\n2\n3")
	}
}

func TestGotoTopAndBottom(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))

	m.GotoBottom()
	if !m.AtBottom() {
		t.Error("GotoBottom should be AtBottom")
	}
	m.GotoTop()
	if !m.AtTop() {
		t.Error("GotoTop should be AtTop")
	}
}

func TestScrollPercent(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10)) // maxYOffset = 7

	if p := m.ScrollPercent(); p != 0 {
		t.Errorf("ScrollPercent() at top = %v, want 0", p)
	}
	m.GotoBottom()
	if p := m.ScrollPercent(); p != 1 {
		t.Errorf("ScrollPercent() at bottom = %v, want 1", p)
	}
	m.GotoTop()
	m.LineDown(3) // halfway-ish through 0..7
	if got, want := m.ScrollPercent(), 3.0/7.0; got != want {
		t.Errorf("ScrollPercent() = %v, want %v", got, want)
	}
}

func TestScrollPercentWhenContentFitsEntirely(t *testing.T) {
	m := New(10, 20)
	m.SetContent(numberedLines(5))
	if p := m.ScrollPercent(); p != 1 {
		t.Errorf("ScrollPercent() when content fits = %v, want 1 (nothing to scroll)", p)
	}
}

func TestSetContentReclampsOffsetForShorterContent(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))
	m.GotoBottom() // yOffset = 7

	m.SetContent(numberedLines(4)) // maxYOffset now 1
	if got := m.View(); got != "1\n2\n3" {
		t.Errorf("View() after shrinking content = %q, want %q (offset should re-clamp)", got, "1\n2\n3")
	}
}

func TestUpdateKeys(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))

	next, cmd := m.Update(tui.Key{Type: tui.KeyDown})
	m = next
	if cmd != nil {
		t.Error("scrolling should not return a Cmd")
	}
	if got := m.View(); got != "1\n2\n3" {
		t.Errorf("after KeyDown: View() = %q, want %q", got, "1\n2\n3")
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyEnd})
	m = next
	if !m.AtBottom() {
		t.Error("KeyEnd should scroll to bottom")
	}
}

func TestUpdateMouseWheel(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))

	next, _ := m.Update(tui.MouseEvent{Action: tui.MouseActionPress, Button: tui.MouseButtonWheelDown})
	m = next
	if got := m.View(); got != "3\n4\n5" {
		t.Errorf("after wheel down: View() = %q, want %q", got, "3\n4\n5")
	}

	next, _ = m.Update(tui.MouseEvent{Action: tui.MouseActionPress, Button: tui.MouseButtonWheelUp})
	m = next
	if got := m.View(); got != "0\n1\n2" {
		t.Errorf("after wheel up: View() = %q, want %q", got, "0\n1\n2")
	}
}

func TestUpdateMouseMotionIgnored(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(10))

	next, _ := m.Update(tui.MouseEvent{Action: tui.MouseActionMotion, Button: tui.MouseButtonWheelDown})
	m = next
	if got := m.View(); got != "0\n1\n2" {
		t.Errorf("motion events should not scroll: View() = %q, want %q", got, "0\n1\n2")
	}
}

func TestViewTruncatesToWidth(t *testing.T) {
	m := New(3, 2)
	m.SetContent("hello\nworld")
	got := m.View()
	want := "hel\nwor"
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestLineRightAndLineLeft(t *testing.T) {
	m := New(3, 2)
	m.SetContent("abcdefghij\nABCDEFGHIJ") // width 10 lines, viewport width 3

	m.LineRight(4)
	if got := m.View(); got != "efg\nEFG" {
		t.Errorf("after LineRight(4): View() = %q, want %q", got, "efg\nEFG")
	}

	m.LineLeft(2)
	if got := m.View(); got != "cde\nCDE" {
		t.Errorf("after LineLeft(2): View() = %q, want %q", got, "cde\nCDE")
	}
}

func TestHorizontalScrollClampsAtEdges(t *testing.T) {
	m := New(3, 1)
	m.SetContent("abcdefghij") // width 10, viewport width 3 -> maxXOffset = 7

	m.LineLeft(5) // already at 0, should clamp, not go negative
	if got := m.View(); got != "abc" {
		t.Errorf("LineLeft below 0: View() = %q, want %q", got, "abc")
	}

	m.LineRight(100) // far past the right edge, should clamp to maxXOffset
	if got := m.View(); got != "hij" {
		t.Errorf("LineRight past the edge: View() = %q, want %q", got, "hij")
	}

	m.LineRight(1) // already at the max, should stay there
	if got := m.View(); got != "hij" {
		t.Errorf("LineRight already at max: View() = %q, want %q", got, "hij")
	}
}

func TestMaxXOffsetZeroWhenContentFits(t *testing.T) {
	m := New(20, 1)
	m.SetContent("short")

	if m.maxXOffset() != 0 {
		t.Errorf("maxXOffset() = %d, want 0 (content narrower than Width)", m.maxXOffset())
	}

	m.LineRight(10) // should be a no-op: nothing to scroll to
	if got := m.View(); got != "short" {
		t.Errorf("LineRight with nothing to scroll to: View() = %q, want %q", got, "short")
	}
}

func TestSetContentReclampsXOffsetForNarrowerContent(t *testing.T) {
	m := New(3, 1)
	m.SetContent("abcdefghij") // width 10
	m.LineRight(7)             // scroll all the way right (maxXOffset = 7)
	if got := m.View(); got != "hij" {
		t.Fatalf("setup: View() = %q, want %q", got, "hij")
	}

	m.SetContent("ab") // now narrower than Width itself
	if got := m.View(); got != "ab" {
		t.Errorf("after SetContent with narrower content: View() = %q, want %q (xOffset should have been re-clamped to 0)", got, "ab")
	}
}

func TestHorizontalScrollBoundedByWidth(t *testing.T) {
	m := New(3, 1)
	m.SetContent("abcdefghij")

	for offset := 0; offset <= 7; offset++ {
		m.LineLeft(100)
		m.LineRight(offset)
		for _, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > 3 {
				t.Errorf("offset=%d: line %q has ansi.Width %d, want <= 3", offset, line, w)
			}
		}
	}
}

// Criterion: default SetContent/AppendLine drop non-SGR sequences; Raw keeps text.
func TestSanitisesByDefaultAndRaw(t *testing.T) {
	const evil = "a\x1b]52;c;ZXZpbA==\x07b\x1b[2Jc\x1b[31md\x1b[0m\te"
	m := New(80, 5)
	m.SetContent(evil)
	m.AppendLine(evil)
	want := "abc\x1b[31md\x1b[0m" + strings.Repeat(" ", 4) + "e" // abcd is 4 columns, tab stops at 8
	for _, l := range []string{m.lines[0], m.lines[1]} {
		if l != want {
			t.Errorf("line %q, want %q", l, want)
		}
		if strings.ContainsAny(l, "\x07\t") || strings.Contains(l, "\x1b]") || strings.Contains(l, "\x1b[2J") {
			t.Errorf("line %q still has sequences", l)
		}
	}
	r := New(80, 5)
	r.Raw = true
	r.SetContent(evil)
	r.AppendLine(evil)
	if r.lines[0] != evil || r.lines[1] != evil {
		t.Errorf("Raw changed content: %q", r.lines)
	}
}

// TestSetContentFastPathMatchesTheFullClean: printable-ASCII text takes the
// fast path, everything else the full clean, and both give what clean would.
func TestSetContentFastPathMatchesTheFullClean(t *testing.T) {
	for _, in := range []string{
		"", "one", "one\ntwo words\nthree", "trailing\n", "\n\nblank\n",
		"tab\tin", "esc \x1b[31mred\x1b[0m ok", "ctl \x01 char", "del\x7f", "cr\r\nx", "café\n中文",
	} {
		fast, ref := New(20, 3), New(20, 3)
		ref.Raw = true
		fast.SetContent(in)
		ref.SetContent(fast.clean(in)) // the same text, cleaned, through the Raw path
		if fast.LineCount() != ref.LineCount() || fast.maxLineWidth != ref.maxLineWidth {
			t.Fatalf("%q: lines %d/%d width %d/%d", in, fast.LineCount(), ref.LineCount(), fast.maxLineWidth, ref.maxLineWidth)
		}
		for i := range ref.lines {
			if fast.lines[i] != ref.lines[i] {
				t.Fatalf("%q: line %d = %q, want %q", in, i, fast.lines[i], ref.lines[i])
			}
		}
	}
}

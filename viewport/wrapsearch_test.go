package viewport

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func TestSoftWrapWrapsByDisplayWidth(t *testing.T) {
	m := New(5, 10)
	m.SoftWrap = true
	m.SetContent("abcdefghijkl\nxy")
	if got, want := m.View(), "abcde\nfghij\nkl\nxy\n\n\n\n\n\n"; got != want {
		t.Errorf("View = %q, want %q", got, want)
	}
	// wide runes: width 5 holds two 2-column runes, never a split one
	m.SetContent("世界你好吗")
	if got := strings.Split(m.View(), "\n"); got[0] != "世界" || got[1] != "你好" || got[2] != "吗" {
		t.Errorf("wide wrap = %q", got[:3])
	}
}

func TestSoftWrapOffMatchesClip(t *testing.T) {
	m := New(5, 2)
	m.SetContent("abcdefghijkl")
	if got := m.View(); got != "abcde\n" {
		t.Errorf("View = %q", got)
	}
}

func TestSoftWrapScrollsByVisualRows(t *testing.T) {
	m := New(4, 2)
	m.SoftWrap = true
	m.SetContent("aaaabbbbcccc\ndd") // rows: aaaa bbbb cccc dd
	if m.LineCount() != 2 {
		t.Fatalf("LineCount = %d", m.LineCount())
	}
	m.LineDown(1)
	if got := m.View(); got != "bbbb\ncccc" {
		t.Errorf("after LineDown(1): %q", got)
	}
	m.GotoBottom()
	if got := m.View(); got != "cccc\ndd" || !m.AtBottom() {
		t.Errorf("bottom: %q atBottom=%v", got, m.AtBottom())
	}
	m.LineDown(10)
	m.LineUp(1)
	if got := m.View(); got != "bbbb\ncccc" {
		t.Errorf("LineUp: %q", got)
	}
}

func TestSoftWrapKeepsStylePerRow(t *testing.T) {
	m := New(4, 3)
	m.SoftWrap = true
	red := ansi.NewStyle().Foreground(ansi.Red).Render("aaaabbbb")
	m.SetContent(red)
	rows := strings.Split(m.View(), "\n")
	for i := 0; i < 2; i++ {
		if ansi.StripANSI(rows[i]) != strings.Repeat(string(rune('a'+i)), 4) || !strings.Contains(rows[i], "\x1b[") {
			t.Errorf("row %d = %q lost its style", i, rows[i])
		}
	}
}

func TestSoftWrapAppendAndTrimFront(t *testing.T) {
	m := New(4, 3)
	m.SoftWrap = true
	for i := 0; i < 6; i++ {
		m.AppendLine("aaaabbbb") // 2 rows each
	}
	m.GotoBottom()
	if m.yOffset != 12-3 {
		t.Fatalf("yOffset = %d, want 9", m.yOffset)
	}
	m.TrimFront(2) // drops 4 lines = 8 rows; 4 rows left
	if m.LineCount() != 2 || m.yOffset != 1 || !m.AtBottom() {
		t.Errorf("after trim: lines=%d y=%d bottom=%v", m.LineCount(), m.yOffset, m.AtBottom())
	}
	m.AppendLine("c")
	m.GotoBottom()
	if got := m.View(); got != "aaaa\nbbbb\nc" {
		t.Errorf("View = %q", got)
	}
	if m.total() != 5 {
		t.Errorf("total = %d, want 5", m.total())
	}
}

func TestSoftWrapWidthChangeRewraps(t *testing.T) {
	m := New(4, 5)
	m.SoftWrap = true
	m.SetContent("aaaabbbb")
	m.Width = 8
	if got := strings.Split(m.View(), "\n")[0]; got != "aaaabbbb" {
		t.Errorf("row 0 = %q", got)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.total() != 1 {
		t.Errorf("total = %d", m.total())
	}
}

func searchable() Model {
	m := New(20, 4)
	m.SetContent("foo bar\nBAR baz\nnothing\nbar bar")
	return m
}

func TestSearchHighlightsAllMatches(t *testing.T) {
	m := searchable()
	if n := m.Search("bar"); n != 4 {
		t.Fatalf("Search = %d matches, want 4", n)
	}
	rows := strings.Split(m.View(), "\n")
	for i, want := range []int{1, 1, 0, 2} {
		if got := strings.Count(rows[i], "\x1b[7m") + strings.Count(rows[i], "\x1b[1;7m"); got != want {
			t.Errorf("row %d has %d highlights, want %d: %q", i, got, want, rows[i])
		}
	}
	if got := ansi.StripANSI(rows[1]); got != "BAR baz" {
		t.Errorf("highlighting changed text: %q", got)
	}
	m.Search("")
	if m.View() != "foo bar\nBAR baz\nnothing\nbar bar" {
		t.Errorf("clearing the query left highlights: %q", m.View())
	}
}

func TestSearchNavigation(t *testing.T) {
	m := searchable()
	m.Search("bar")
	if m.CurrentMatch() != 1 || m.MatchCount() != 4 {
		t.Fatalf("current=%d count=%d", m.CurrentMatch(), m.MatchCount())
	}
	key := func(r rune) { m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})}) }
	key('n')
	if m.CurrentMatch() != 2 {
		t.Errorf("n: %d", m.CurrentMatch())
	}
	key('n')
	key('n')
	key('n') // wraps
	if m.CurrentMatch() != 1 {
		t.Errorf("wrap forward: %d", m.CurrentMatch())
	}
	key('N') // wraps back
	if m.CurrentMatch() != 4 {
		t.Errorf("N wrap: %d", m.CurrentMatch())
	}
	if !strings.Contains(m.View(), "\x1b[1;7m") {
		t.Error("current match not marked")
	}
}

func TestSearchKeysInertWithoutQuery(t *testing.T) {
	m := searchable()
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"})
	if m.CurrentMatch() != 0 {
		t.Error("n with no query moved")
	}
}

func TestSearchScrollsMatchIntoView(t *testing.T) {
	m := New(10, 3)
	m.SetContent(numberedLines(30))
	if n := m.Search("22"); n != 1 {
		t.Fatalf("n = %d", n)
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "22") {
		t.Errorf("match not visible: %q", m.View())
	}
	if n := m.Search("zzz"); n != 0 || m.NextMatch() {
		t.Error("no matches should report 0/false")
	}
}

func TestSearchMatchSpanningWrapBoundary(t *testing.T) {
	m := New(4, 4)
	m.SoftWrap = true
	m.SetContent("abcdefgh")
	m.Search("cdef")
	rows := strings.Split(m.View(), "\n")
	if ansi.StripANSI(rows[0]) != "abcd" || ansi.StripANSI(rows[1]) != "efgh" {
		t.Fatalf("rows = %q", rows)
	}
	if !strings.Contains(rows[0], "cd") || !strings.Contains(rows[1], hlCur+"ef") {
		t.Errorf("highlight not split across rows: %q", rows)
	}
}

func TestSearchPreservesStyledText(t *testing.T) {
	m := New(20, 2)
	m.SetContent(ansi.NewStyle().Foreground(ansi.Red).Render("hello world"))
	m.Search("WORLD")
	if got := ansi.StripANSI(m.View()); got != "hello world\n" {
		t.Errorf("text = %q", got)
	}
	if !strings.Contains(m.View(), "\x1b[1;7mworld") {
		t.Errorf("match not highlighted: %q", m.View())
	}
}

func TestSearchNoLongerDriftsAfterTrimFront(t *testing.T) {
	m := New(10, 2)
	m.SetContent("x\nbar\ny\nbar")
	m.Search("bar")
	m.NextMatch() // second bar, absolute line 3
	m.TrimFront(2)
	if m.CurrentMatch() != 1 || m.MatchCount() != 1 {
		t.Errorf("current=%d count=%d", m.CurrentMatch(), m.MatchCount())
	}
}

func TestDrawCellsMatchesViewWrapAndSearch(t *testing.T) {
	for _, q := range []string{"", "ij"} {
		m := New(6, 5)
		m.SoftWrap = true
		m.SetContent("abcdefghijklmnop\nqrs\n" + ansi.NewStyle().Bold().Render("stuvwxyzabcdefij"))
		m.Search(q)
		buf := cellbuf.New(6, 5)
		m.DrawCells(buf, cellbuf.Rect{W: 6, H: 5})
		sameScreen(t, m.View(), buf, 6, 5)
	}
}

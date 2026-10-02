package logview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func TestFollowPinsEvenWhenMovedOff(t *testing.T) {
	m := New(10, 2)
	m.Follow = true
	for i := 0; i < 5; i++ {
		m.Append(fmt.Sprint("l", i))
	}
	m.Viewport.GotoTop() // moved by code, not by the reader
	m.Append("l5")
	if !m.Viewport.AtBottom() || !m.Follow || m.NewLines() != 0 {
		t.Errorf("not pinned: bottom=%v follow=%v new=%d", m.Viewport.AtBottom(), m.Follow, m.NewLines())
	}
}

func TestFollowOffWhenReaderScrollsAway(t *testing.T) {
	m := New(10, 2)
	for i := 0; i < 5; i++ {
		m.Append(fmt.Sprint("l", i))
	}
	m.SetFollow(true)
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	if m.Follow {
		t.Fatal("Follow still on after scrolling up")
	}
	m.Append("x")
	if m.Viewport.AtBottom() || m.NewLines() != 1 {
		t.Errorf("should stay scrolled up: bottom=%v new=%d", m.Viewport.AtBottom(), m.NewLines())
	}
}

func TestSoftWrapLogFollowsVisualBottom(t *testing.T) {
	m := New(4, 2)
	m.Viewport.SoftWrap = true
	m.Follow = true
	m.Append("aaaabbbb")
	m.Append("cccc")
	if got := m.View(); got != "bbbb\ncccc" {
		t.Errorf("View = %q", got)
	}
}

func TestLogSearchJumpsAndLeavesFollow(t *testing.T) {
	m := New(10, 2)
	for i := 0; i < 20; i++ {
		m.Append(fmt.Sprint("line ", i))
	}
	m.SetFollow(true)
	if n := m.Search("line 3"); n != 1 {
		t.Fatalf("matches = %d", n)
	}
	if m.Follow {
		t.Error("jumping up to a match should leave follow mode")
	}
	if !strings.Contains(ansi.StripANSI(m.View()), "line 3") {
		t.Errorf("match not visible: %q", m.View())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"})
	if m.Viewport.CurrentMatch() != 1 {
		t.Error("n with one match should stay on it")
	}
}

func TestLogDrawCellsWithSearchAndIndicator(t *testing.T) {
	m := New(8, 3)
	m.NewLinesIndicator = true
	m.Viewport.SoftWrap = true
	for i := 0; i < 6; i++ {
		m.Append("abcdefghij")
	}
	m.Viewport.GotoTop()
	m.Append("zz")
	m.Search("cd")
	buf := cellbuf.New(8, 3)
	m.DrawCells(buf, cellbuf.Rect{W: 8, H: 3})
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 8; x++ {
			if want.At(x, y).Cluster != buf.At(x, y).Cluster {
				t.Fatalf("cell %d,%d: view %q draw %q", x, y, want.At(x, y).Cluster, buf.At(x, y).Cluster)
			}
		}
	}
}

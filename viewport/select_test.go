package viewport

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/hittest"
)

func mouse(m Model, action tui.MouseAction, x, y int) Model {
	m, _ = m.Update(tui.MouseEvent{Button: tui.MouseButtonLeft, Action: action, X: x, Y: y})
	return m
}

func drag(m Model, x0, y0, x1, y1 int) Model {
	m = mouse(m, tui.MouseActionPress, x0, y0)
	m = mouse(m, tui.MouseActionMotion, x1, y1)
	return mouse(m, tui.MouseActionRelease, x1, y1)
}

func sample() Model {
	m := New(20, 4)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 20, H: 4}
	m.SetContent("alpha beta\ngamma delta\nepsilon zeta\neta theta")
	return m
}

func TestDragSelectsAcrossLines(t *testing.T) {
	m := drag(sample(), 6, 0, 4, 2) // from "beta" to "epsil|o"
	if got, want := m.SelectedText(), "beta\ngamma delta\nepsil"; got != want {
		t.Errorf("SelectedText = %q, want %q", got, want)
	}
	// Dragging backwards selects the same text.
	back := drag(sample(), 4, 2, 6, 0)
	if back.SelectedText() != m.SelectedText() {
		t.Errorf("backwards = %q", back.SelectedText())
	}
}

func TestSelectionIsDrawnAndViewTextIsUnchanged(t *testing.T) {
	plain := sample()
	m := drag(sample(), 6, 0, 9, 0)
	if ansi.StripANSI(m.View()) != ansi.StripANSI(plain.View()) {
		t.Error("selecting changed the text of View")
	}
	if !strings.Contains(strings.Split(m.View(), "\n")[0], "\x1b[7m") {
		t.Errorf("the selected cells are not reversed: %q", m.View())
	}
	if strings.Contains(strings.Split(m.View(), "\n")[1], "\x1b[7m") {
		t.Error("a row outside the selection is highlighted")
	}
}

func TestDrawCellsMatchesViewWithASelection(t *testing.T) {
	m := drag(sample(), 3, 0, 5, 1)
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatal(err)
	}
	got := cellbuf.New(20, 4)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < 4; y++ {
		for x := 0; x < 20; x++ {
			a, b := want.At(x, y), got.At(x, y)
			if a.Cluster != b.Cluster || want.Style(a.Style) != got.Style(b.Style) {
				t.Fatalf("cell (%d,%d): View %q %+v, DrawCells %q %+v", x, y, a.Cluster, want.Style(a.Style), b.Cluster, got.Style(b.Style))
			}
		}
	}
}

func TestClickWithoutDragAndOutsideBoundsSelectNothing(t *testing.T) {
	m := mouse(sample(), tui.MouseActionPress, 3, 1)
	m = mouse(m, tui.MouseActionRelease, 3, 1)
	if m.SelectedText() != "" {
		t.Errorf("a click selected %q", m.SelectedText())
	}
	s := sample()
	s.Bounds = hittest.Rect{X: 5, Y: 5, W: 10, H: 2}
	if got := drag(s, 0, 0, 4, 0).SelectedText(); got != "" {
		t.Errorf("a press outside Bounds selected %q", got)
	}
	off := sample()
	off.Mouse = false
	if got := drag(off, 0, 0, 8, 1).SelectedText(); got != "" {
		t.Errorf("with Mouse off, selected %q", got)
	}
}

func TestSelectedTextIsPlainAndCopyReturnsACmd(t *testing.T) {
	m := New(20, 2)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 2}
	m.SetContent(ansi.NewStyle().Bold().Foreground(ansi.Red).Render("hot") + " cold\nnext")
	m = drag(m, 0, 0, 7, 0)
	if got := m.SelectedText(); got != "hot cold" {
		t.Errorf("SelectedText = %q", got)
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "y", Code: 'y'})
	if cmd == nil {
		t.Error("Copy with a selection returned no Cmd")
	}
	empty := sample()
	if _, cmd := empty.Update(tui.Key{Type: tui.KeyRunes, Text: "y", Code: 'y'}); cmd != nil {
		t.Error("Copy with no selection returned a Cmd")
	}
}

func TestSelectionFollowsSoftWrap(t *testing.T) {
	m := New(5, 4)
	m.SoftWrap, m.Mouse = true, true
	m.Bounds = hittest.Rect{W: 5, H: 4}
	m.SetContent("abcdefghij\nxy") // rows: abcde | fghij | xy
	m = drag(m, 3, 0, 1, 1)        // "de" and "fg" across the wrap
	if got := m.SelectedText(); got != "defg" {
		t.Errorf("SelectedText = %q, want defg", got)
	}
}

func TestDraggingPastTheBottomScrollsAndSelectAllClear(t *testing.T) {
	m := New(10, 2)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 10, H: 2}
	m.SetContent("1\n2\n3\n4\n5")
	m = mouse(m, tui.MouseActionPress, 0, 0)
	m = mouse(m, tui.MouseActionMotion, 0, 3) // below the last row
	if m.yOffset == 0 {
		t.Error("dragging below the viewport did not scroll")
	}
	m = mouse(m, tui.MouseActionRelease, 0, 3)
	if !m.sel.active {
		t.Error("no selection after a drag")
	}
	m.SelectAll()
	if got := m.SelectedText(); got != "1\n2\n3\n4\n5" {
		t.Errorf("SelectAll = %q", got)
	}
	m.ClearSelection()
	if m.SelectedText() != "" {
		t.Error("ClearSelection left text")
	}
}

func TestTrimFrontEndsASelectionThatLostALine(t *testing.T) {
	m := New(10, 2)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 10, H: 2}
	m.SetContent("a\nb\nc\nd\ne")
	m.SelectAll()
	m.TrimFront(2)
	if m.SelectedText() != "" {
		t.Errorf("selection kept after its first line was dropped: %q", m.SelectedText())
	}
	m.SelectAll()
	if got := m.SelectedText(); got != "d\ne" {
		t.Errorf("SelectAll after trim = %q", got)
	}
}

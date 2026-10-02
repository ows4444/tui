package menu

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func press(m Model, k tui.KeyType) Model {
	next, _ := m.Update(tui.Key{Type: k})
	return next
}

func TestLinearizeRootAndDrilledIn(t *testing.T) {
	m := New([]Item{
		{Label: "File", Children: []Item{{Label: "Open"}, {Label: "Recent", Children: []Item{{Label: "a.txt"}}}}},
		{Label: "Quit"},
	})
	want := "File, item 1 of 2, submenu, selected\nQuit, item 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("root =\n%s\nwant\n%s", got, want)
	}

	m = press(m, tui.KeyEnter) // into File
	want = "Submenu: File\nOpen, item 1 of 2, selected\nRecent, item 2 of 2, submenu"
	if got := m.Linearize(); got != want {
		t.Errorf("File =\n%s\nwant\n%s", got, want)
	}

	m = press(press(m, tui.KeyDown), tui.KeyEnter) // into Recent
	want = "Submenu: File, Recent\na.txt, item 1 of 1, selected"
	if got := m.Linearize(); got != want {
		t.Errorf("Recent =\n%s\nwant\n%s", got, want)
	}

	m = press(press(m, tui.KeyEsc), tui.KeyEsc) // back to root; parent cursor kept
	if got := m.Linearize(); got != "File, item 1 of 2, submenu, selected\nQuit, item 2 of 2" {
		t.Errorf("back at root = %q", got)
	}
}

func TestLinearizeEmpty(t *testing.T) {
	if got := New(nil).Linearize(); got != "No items" {
		t.Errorf("empty = %q", got)
	}
}

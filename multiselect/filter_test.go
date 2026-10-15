package multiselect

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

func typeText(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	return m
}

func filterModel() Model {
	m := NewStrings("Red apple", "Green apple", "Banana", "Apricot", "grape")
	m.Filter = true
	return m
}

func TestFilterOffIgnoresTyping(t *testing.T) {
	m := NewStrings("a", "b")
	m = typeText(m, "b")
	if m.Query() != "" || m.Cursor() != 0 {
		t.Errorf("Filter off: query %q cursor %d", m.Query(), m.Cursor())
	}
}

func TestFilterNarrowsAndRanks(t *testing.T) {
	m := typeText(filterModel(), "ap")
	got := ansi.StripANSI(m.View())
	if !strings.HasPrefix(got, "/ ap") || strings.Contains(got, "Banana") {
		t.Errorf("View:\n%s", got)
	}
	if m.Highlighted() != "Apricot" {
		t.Errorf("cursor on %q, want the best match Apricot", m.Highlighted())
	}
}

// Highlighted is a test helper: the label under the cursor.
func (m Model) Highlighted() string { return m.Items[m.cursor].Label }

func TestFilterToggleKeepsOriginalIndexAndSurvivesQueryChange(t *testing.T) {
	m := typeText(filterModel(), "ban")
	m, _ = m.Update(tui.Key{Type: tui.KeySpace}) // toggles Banana
	if got := m.SelectedIndexes(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("SelectedIndexes = %v, want [2]", got)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	m = typeText(m, "gr")
	if !m.IsSelected(2) {
		t.Error("a hidden checked item lost its check")
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	conf := cmd().(ConfirmedMsg)
	if len(conf.Items) != 1 || conf.Items[0].Label != "Banana" {
		t.Errorf("ConfirmedMsg = %+v, want Banana though hidden", conf)
	}
}

func TestFilterNoMatchesToggleDoesNothing(t *testing.T) {
	m := typeText(filterModel(), "zzz")
	if !strings.Contains(ansi.StripANSI(m.View()), "no matches") {
		t.Errorf("view:\n%s", ansi.StripANSI(m.View()))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeySpace})
	if len(m.SelectedIndexes()) != 0 {
		t.Error("Space with no matches toggled an item")
	}
}

func TestFilterBackspaceAndNavigation(t *testing.T) {
	m := typeText(filterModel(), "apple")
	first := m.Highlighted()
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	second := m.Highlighted()
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if first == second || m.Highlighted() != second {
		t.Errorf("first %q second %q then %q", first, second, m.Highlighted())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if m.Query() != "appl" {
		t.Errorf("query %q", m.Query())
	}
}

func TestFilterMouseClick(t *testing.T) {
	m := typeText(filterModel(), "ban")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 20, H: 6}
	m, _ = m.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Highlighted() != "Banana" {
		t.Errorf("click on row 1 -> %q", m.Highlighted())
	}
}

func TestFilterLinearize(t *testing.T) {
	m := typeText(filterModel(), "ban")
	want := "Filter \"ban\", 1 of 5 items\nBanana, not checked, item 1 of 1, current"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := typeText(m, "zz").Linearize(); got != "Filter \"banzz\", no matches" {
		t.Errorf("no match Linearize = %q", got)
	}
}

func TestFilterLayoutNode(t *testing.T) {
	m := typeText(filterModel(), "ap")
	if sz := m.LayoutNode().Measure(layout.Constraints{MaxW: 40, MaxH: 40}); sz.H != 5 {
		t.Errorf("Measure H = %d, want 5 (query line + four matches)", sz.H)
	}
	lines := strings.Split(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 3})), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "/ ap") {
		t.Errorf("cramped render keeps the query line: %q", lines)
	}
}

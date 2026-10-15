package picker

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
)

func typeText(m Model, s string) Model {
	for _, r := range s {
		if r == ' ' {
			m, _ = m.Update(tui.Key{Type: tui.KeySpace})
			continue
		}
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
	m.SetQuery("b")
	if m.Query() != "" {
		t.Errorf("SetQuery with Filter off kept %q", m.Query())
	}
}

func TestFilterNarrowsAndRanks(t *testing.T) {
	m := typeText(filterModel(), "ap")
	got := ansi.StripANSI(m.View())
	for _, want := range []string{"/ ap", "Red apple", "Green apple", "Apricot", "grape"} {
		if !strings.Contains(got, want) {
			t.Errorf("View missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Banana") {
		t.Errorf("Banana does not match 'ap':\n%s", got)
	}
	// Best match first: "Apricot" begins with it.
	if m.Highlighted().Label != "Apricot" {
		t.Errorf("cursor on %q, want the best match Apricot", m.Highlighted().Label)
	}
}

func TestFilterSelectKeepsOriginalIndex(t *testing.T) {
	m := typeText(filterModel(), "ban")
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on a match returned no Cmd")
	}
	msg := cmd().(SelectedMsg)
	if msg.Index != 2 || msg.Item.Label != "Banana" {
		t.Errorf("SelectedMsg = %+v, want index 2 Banana", msg)
	}
}

func TestFilterNavigationStaysInMatches(t *testing.T) {
	m := typeText(filterModel(), "apple")
	first := m.Highlighted().Label
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	second := m.Highlighted().Label
	m, _ = m.Update(tui.Key{Type: tui.KeyDown}) // already last
	if m.Highlighted().Label != second || first == second {
		t.Errorf("first %q second %q then %q", first, second, m.Highlighted().Label)
	}
	for _, l := range []string{first, second} {
		if !strings.Contains(l, "apple") {
			t.Errorf("cursor left the matches: %q", l)
		}
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyHome})
	if m.Highlighted().Label != first {
		t.Errorf("Home -> %q, want %q", m.Highlighted().Label, first)
	}
}

func TestFilterBackspaceAndEsc(t *testing.T) {
	m := typeText(filterModel(), "grz")
	if !strings.Contains(ansi.StripANSI(m.View()), "no matches") {
		t.Errorf("no match view:\n%s", ansi.StripANSI(m.View()))
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Error("Enter with no matches returned a Cmd")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if m.Query() != "gr" || m.Highlighted().Label != "grape" {
		t.Errorf("after Backspace: %q on %q", m.Query(), m.Highlighted().Label)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	if m.Query() != "" || strings.Count(ansi.StripANSI(m.View()), "\n") != 5 {
		t.Errorf("after Esc: query %q\n%s", m.Query(), ansi.StripANSI(m.View()))
	}
}

func TestFilterBindingsWinOverTyping(t *testing.T) {
	m := filterModel()
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Down = keymap.NewBinding("down", "j")
	m = typeText(m, "j")
	if m.Query() != "" || m.Cursor() != 1 {
		t.Errorf("bound 'j' should move, not type: query %q cursor %d", m.Query(), m.Cursor())
	}
}

func TestFilterHighlightsMatchedRunes(t *testing.T) {
	m := typeText(filterModel(), "gp")
	if !strings.Contains(m.View(), "\x1b[") {
		t.Error("matched runes carry no styling")
	}
	if got := ansi.StripANSI(m.View()); !strings.Contains(got, "grape") || !strings.Contains(got, "Green apple") {
		t.Errorf("view:\n%s", got)
	}
}

func TestFilterMouseClickUsesVisibleRows(t *testing.T) {
	m := typeText(filterModel(), "ban")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 20, H: 6}
	// Row 0 is the query line, row 1 the only match.
	m, _ = m.Update(tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Highlighted().Label != "Banana" {
		t.Errorf("click on row 1 -> %q", m.Highlighted().Label)
	}
}

func TestFilterLinearize(t *testing.T) {
	m := typeText(filterModel(), "ban")
	want := "Filter \"ban\", 1 of 5 items\nBanana, item 1 of 1, selected"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	m = typeText(m, "zz")
	if got := m.Linearize(); got != "Filter \"banzz\", no matches" {
		t.Errorf("no match Linearize = %q", got)
	}
	// Without a query the output is the plain list.
	if got := filterModel().Linearize(); !strings.HasPrefix(got, "Red apple, item 1 of 5") {
		t.Errorf("empty query Linearize = %q", got)
	}
}

func TestFilterLayoutNode(t *testing.T) {
	m := typeText(filterModel(), "ap")
	sz := m.LayoutNode().Measure(layout.Constraints{MaxW: 40, MaxH: 40})
	if sz.H != 5 { // query line + four matches
		t.Errorf("Measure H = %d, want 5", sz.H)
	}
	out := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 20, H: 2}))
	lines := strings.Split(out, "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "/ ap") {
		t.Errorf("cramped render keeps the query line:\n%s", out)
	}
}

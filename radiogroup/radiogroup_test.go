package radiogroup

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func sizes() Model {
	m := New("Small", "Medium", "Large")
	m.ID = "size"
	m.Focus()
	return m
}

func TestNothingIsChosenAtFirst(t *testing.T) {
	m := New("Small", "Medium")
	if _, ok := m.Index(); ok || m.Value() != "" {
		t.Fatalf("a new group has %q chosen", m.Value())
	}
	if got := plain(m); got != "( ) Small\n( ) Medium" {
		t.Errorf("View =\n%s", got)
	}
}

func TestSpaceChoosesTheOptionUnderTheCursor(t *testing.T) {
	m := sizes()
	m, cmd := m.Update(key(tui.KeySpace))
	if c, ok := changed(cmd); !ok || c != (ChangedMsg{ID: "size", Index: 0, Value: "Small"}) {
		t.Fatalf("ChangedMsg = %+v, delivered %v", c, ok)
	}
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("choosing the chosen option again delivered a message")
	}
}

func TestArrowsMoveAndChoose(t *testing.T) {
	m := sizes()
	steps := []struct {
		k    tui.KeyType
		want string
	}{
		{tui.KeyDown, "Medium"}, {tui.KeyRight, "Large"}, {tui.KeyDown, "Small"}, // wraps
		{tui.KeyUp, "Large"}, {tui.KeyLeft, "Medium"}, {tui.KeyHome, "Small"}, {tui.KeyEnd, "Large"},
	}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		c, ok := changed(cmd)
		if !ok || c.Value != s.want || m.Value() != s.want {
			t.Fatalf("step %d: chose %q (message %+v), want %q", i, m.Value(), c, s.want)
		}
		if i, _ := m.Index(); i != m.Cursor() {
			t.Fatalf("step %d: cursor %d is not on the chosen option %d", i, m.Cursor(), i)
		}
	}
}

func TestDisabledOptionsAreSkipped(t *testing.T) {
	m := New("A", "B", "C")
	m.Options[0].Disabled = true
	m.Options[1].Disabled = true
	m.Focus()
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d, want 2: the first two options are disabled", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Value() != "C" {
		t.Fatalf("Value = %q, want C: it is the only option that can be chosen", m.Value())
	}
	m.Select(0)
	if m.Value() != "C" {
		t.Fatal("Select chose a disabled option")
	}
	if m.SelectValue("A") {
		t.Fatal("SelectValue chose a disabled option")
	}
	all := New("A")
	all.Options[0].Disabled = true
	all.Focus()
	if next, cmd := all.Update(key(tui.KeySpace)); cmd != nil || next.Value() != "" {
		t.Fatal("a group with every option disabled chose one")
	}
}

func TestAGroupWithoutFocusIgnoresKeys(t *testing.T) {
	m := New("A", "B")
	if next, cmd := m.Update(key(tui.KeyDown)); cmd != nil || next.Value() != "" {
		t.Fatal("a key chose an option of a group that does not have focus")
	}
	var empty Model
	empty.Focus()
	if _, cmd := empty.Update(key(tui.KeySpace)); cmd != nil {
		t.Fatal("an empty group returned a Cmd")
	}
	m.Focus()
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled; it should leave the group")
	}
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
}

func TestSelectAndValue(t *testing.T) {
	m := New("One", "Two")
	m.Options[1].Value = "2"
	m.Select(1)
	if i, ok := m.Index(); !ok || i != 1 || m.Value() != "2" {
		t.Fatalf("Index = %d, %v; Value = %q", i, ok, m.Value())
	}
	m.Select(7)
	if m.Value() != "2" {
		t.Fatal("Select took an index out of range")
	}
	if !m.SelectValue("One") || m.Value() != "One" {
		t.Fatal("SelectValue did not find an option by its label")
	}
	if m.SelectValue("nope") {
		t.Fatal("SelectValue reported a value no option has")
	}
	m.Options = m.Options[:0]
	if m.Value() != "" {
		t.Fatal("Value() of a group whose options were removed is not empty")
	}
}

// Focus puts the cursor on the chosen option, so the arrow keys continue
// from the choice.
func TestFocusStartsAtTheChosenOption(t *testing.T) {
	m := New("A", "B", "C")
	m.Select(2)
	m.Blur()
	m.Focus()
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d, want 2", m.Cursor())
	}
}

func TestViewMarksChoiceAndCursorWithoutColour(t *testing.T) {
	m := sizes()
	m.Options[2].Disabled = true
	m, _ = m.Update(key(tui.KeyDown))
	if got := plain(m); got != "( ) Small\n<●> Medium\n(—) Large" {
		t.Errorf("View =\n%s", got)
	}
	m.Blur()
	if got := plain(m); got != "( ) Small\n(●) Medium\n(—) Large" {
		t.Errorf("View without focus =\n%s", got)
	}
	m.Horizontal = true
	if got := plain(m); got != "( ) Small  (●) Medium  (—) Large" {
		t.Errorf("horizontal = %q", got)
	}
	m.Gap = -1
	if got := plain(m); got != "( ) Small(●) Medium(—) Large" {
		t.Errorf("negative gap = %q", got)
	}
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := plain(m); got != "( ) Small(*) Medium(-) Large" {
		t.Errorf("ASCII = %q", got)
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := sizes()
	m.KeyMap = KeyMap{Next: keymap.NewBinding("next", "j")}
	if next, _ := m.Update(key(tui.KeyDown)); next.Value() != "" {
		t.Fatal("Down still chose with a custom KeyMap")
	}
	if next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "j"}); next.Value() != "Medium" {
		t.Fatal("the custom key did not choose")
	}
	z := Model{Options: []Option{{Label: "A"}, {Label: "B"}}, Theme: theme.DarkTheme()}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyDown)); next.Value() != "B" {
		t.Fatal("a struct-literal Model with no KeyMap ignored Down")
	}
	if len(m.Bindings()) != 5 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

func TestMouseChoosesTheOptionPressed(t *testing.T) {
	m := New("Small", "Medium", "Large")
	m.Mouse, m.Bounds = true, hittest.Rect{X: 4, Y: 10, W: 20, H: 3}
	press := func(x, y int) (Model, tui.Cmd) {
		return m.Update(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	}
	m, cmd := press(6, 11)
	if c, ok := changed(cmd); !ok || c.Value != "Medium" {
		t.Fatalf("press on row 2: %+v", c)
	}
	if next, cmd := press(20, 10); cmd != nil || next.Value() != "Medium" {
		t.Fatal("a press to the right of a label chose it")
	}
	if _, cmd := press(40, 30); cmd != nil {
		t.Fatal("a press outside Bounds chose an option")
	}
	if _, cmd := m.Update(tui.MouseEvent{X: 6, Y: 10, Button: tui.MouseButtonRight, Action: tui.MouseActionPress}); cmd != nil {
		t.Fatal("the right button chose an option")
	}
	m.Horizontal = true // "( ) Small  (●) Medium  ( ) Large": Large starts at local 23
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 1}
	m, cmd = press(24, 0)
	if c, ok := changed(cmd); !ok || c.Value != "Large" {
		t.Fatalf("press on Large in a horizontal group: %+v", c)
	}
	if _, cmd := press(9, 0); cmd != nil {
		t.Fatal("a press in the gap chose an option")
	}
	m.Bounds.H = 2
	if _, cmd := press(1, 1); cmd != nil {
		t.Fatal("a press under a horizontal group chose an option")
	}
	m.Mouse = false
	if _, cmd := press(1, 0); cmd != nil {
		t.Fatal("the pointer acted on a group with Mouse off")
	}
}

func TestLinearize(t *testing.T) {
	m := New("Small", "Medium")
	m.Options[1].Disabled = true
	m.Select(0)
	want := "Small, radio button 1 of 2, chosen\nMedium, radio button 2 of 2, unavailable"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := (Model{}).Linearize(); got != "No options" {
		t.Errorf("empty = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New("A", "B").SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Text: red})
	if m.Tokens().Text != red {
		t.Error("WithTokens did not override the text colour")
	}
	n := m.LayoutNode()
	s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded})
	if s != (layout.Size{W: 5, H: 2}) {
		t.Fatalf("Measure = %+v, want 5x2", s)
	}
}

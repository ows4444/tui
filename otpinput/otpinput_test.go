package otpinput

import (
	"testing"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }
func typed(s string) tui.Key    { return tui.Key{Type: tui.KeyRunes, Text: s} }

func changed(cmd tui.Cmd) (ChangedMsg, bool) {
	if cmd == nil {
		return ChangedMsg{}, false
	}
	c, ok := cmd().(ChangedMsg)
	return c, ok
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func code() Model {
	m := New(4)
	m.ID = "code"
	m.Focus()
	return m
}

func TestTypingFillsCellsAndMovesOn(t *testing.T) {
	m := code()
	for i, want := range []string{"1", "12", "123"} {
		var cmd tui.Cmd
		m, cmd = m.Update(typed(string(want[i])))
		if c, ok := changed(cmd); !ok || c.Value != want || c.Complete || c.ID != "code" {
			t.Fatalf("after %q: %+v, delivered %v", want, c, ok)
		}
		if m.Cursor() != i+1 {
			t.Fatalf("after %q: cursor %d, want %d", want, m.Cursor(), i+1)
		}
	}
	m, cmd := m.Update(typed("4"))
	if c, ok := changed(cmd); !ok || c.Value != "1234" || !c.Complete || !m.Complete() {
		t.Fatalf("the last cell: %+v", c)
	}
	if m.Cursor() != 3 {
		t.Fatalf("cursor = %d after the last cell, want it to stay on 3", m.Cursor())
	}
}

func TestCharactersItDoesNotAcceptAreDropped(t *testing.T) {
	m := code()
	for _, s := range []string{"a", " ", "-", "é"} {
		if next, cmd := m.Update(typed(s)); cmd != nil || next.Value() != "" || next.Cursor() != 0 {
			t.Errorf("%q was taken by a digits field", s)
		}
	}
	if _, cmd := m.Update(key(tui.KeySpace)); cmd != nil {
		t.Error("the space bar was taken")
	}
	m.Accept, m.Upper = unicode.IsLetter, true
	m, _ = m.Update(typed("a"))
	m, _ = m.Update(typed("7"))
	if m.Value() != "A" {
		t.Fatalf("Value = %q, want A: letters only, made capital", m.Value())
	}
}

func TestBackspaceEmptiesThisCellOrTheOneBefore(t *testing.T) {
	m := code()
	m.SetValue("12")
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d after SetValue, want 2", m.Cursor())
	}
	m, cmd := m.Update(key(tui.KeyBackspace)) // cell 2 is empty: step back and empty cell 1
	if c, ok := changed(cmd); !ok || c.Value != "1" || m.Cursor() != 1 {
		t.Fatalf("first backspace: %+v, cursor %d", c, m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyBackspace))
	if m.Value() != "" || m.Cursor() != 0 {
		t.Fatalf("second backspace: %q, cursor %d", m.Value(), m.Cursor())
	}
	if _, cmd := m.Update(key(tui.KeyBackspace)); cmd != nil {
		t.Fatal("backspace on an empty field delivered a change")
	}
	m.SetValue("1234")
	m, _ = m.Update(key(tui.KeyBackspace)) // the cursor is on the filled last cell: empty it
	if m.Value() != "123" || m.Cursor() != 3 {
		t.Fatalf("backspace on a filled cell: %q, cursor %d", m.Value(), m.Cursor())
	}
}

func TestMovingAndOverwriting(t *testing.T) {
	m := code()
	m.SetValue("1234")
	m, _ = m.Update(key(tui.KeyHome))
	m, _ = m.Update(key(tui.KeyRight))
	m, cmd := m.Update(typed("9"))
	if c, ok := changed(cmd); !ok || c.Value != "1934" || !c.Complete {
		t.Fatalf("overwriting cell 1: %+v", c)
	}
	if _, cmd := m.Update(key(tui.KeyLeft)); cmd != nil {
		t.Fatal("moving the cursor delivered a change")
	}
	m, _ = m.Update(key(tui.KeyEnd))
	if m.Cursor() != 3 {
		t.Fatalf("End: cursor %d", m.Cursor())
	}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Cursor() != 3 {
		t.Fatal("Right moved past the last cell")
	}
	m, _ = m.Update(key(tui.KeyHome))
	m, _ = m.Update(key(tui.KeyLeft))
	if m.Cursor() != 0 {
		t.Fatal("Left moved before the first cell")
	}
	// Typing the character a cell already holds changes nothing.
	if _, cmd := m.Update(typed("1")); cmd != nil {
		t.Fatal("typing the same character delivered a change")
	}
}

func TestAPasteFillsFromTheCursor(t *testing.T) {
	m := code()
	m, cmd := m.Update(tui.PasteEvent{Text: "12-34 56"})
	if c, ok := changed(cmd); !ok || c.Value != "1234" || !c.Complete {
		t.Fatalf("paste: %+v", c)
	}
	m.SetValue("")
	m, _ = m.Update(typed("7"))
	m, _ = m.Update(tui.PasteEvent{Text: "89"})
	if m.Value() != "789" || m.Cursor() != 3 {
		t.Fatalf("paste after one typed: %q, cursor %d", m.Value(), m.Cursor())
	}
	if _, cmd := m.Update(tui.PasteEvent{Text: "abc"}); cmd != nil {
		t.Fatal("a paste with nothing the field accepts delivered a change")
	}
}

func TestClearEmptiesTheField(t *testing.T) {
	m := code()
	m.SetValue("12")
	m, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "u", Code: 'u', Mod: input.ModCtrl})
	if c, ok := changed(cmd); !ok || c.Value != "" || m.Cursor() != 0 {
		t.Fatalf("ctrl+u: %+v, cursor %d", c, m.Cursor())
	}
}

func TestSetValueAndALengthThatChanges(t *testing.T) {
	m := New(6)
	m.SetValue("12x34567890")
	if m.Value() != "123456" || !m.Complete() {
		t.Fatalf("SetValue past the length: %q", m.Value())
	}
	m.Length = 4 // shorter: the cells past the end are dropped
	if m.Value() != "1234" || m.Cursor() != 3 {
		t.Fatalf("after Length fell to 4: %q, cursor %d", m.Value(), m.Cursor())
	}
	m.Length = 0 // read as 6
	if m.Width() != 18 {
		t.Fatalf("a Length of 0: width %d, want 18", m.Width())
	}
}

// A Model is a value: changing a copy must not change the original through
// the cells they share.
func TestACopyDoesNotChangeTheOriginal(t *testing.T) {
	m := code()
	m.SetValue("12")
	c := m
	c, _ = c.Update(typed("3"))
	c, _ = c.Update(key(tui.KeyHome))
	c, _ = c.Update(typed("9"))
	if m.Value() != "12" || c.Value() != "923" {
		t.Fatalf("the original is %q and the copy %q, want 12 and 923", m.Value(), c.Value())
	}
}

func TestWithoutFocusOrDisabledItIgnoresInput(t *testing.T) {
	m := New(4)
	if _, cmd := m.Update(typed("1")); cmd != nil {
		t.Fatal("a key filled a field that does not have focus")
	}
	m.Focus()
	m.Blur()
	if m.Focused() {
		t.Fatal("Focused() after Blur")
	}
	m.Focus()
	m.Disabled = true
	m.Mouse, m.Bounds = true, hittest.Rect{W: 12, H: 1}
	if _, cmd := m.Update(typed("1")); cmd != nil {
		t.Fatal("a key filled a disabled field")
	}
	if next, _ := m.Update(tui.MouseEvent{X: 7, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); next.Cursor() != 0 {
		t.Fatal("a click moved the cursor of a disabled field")
	}
	if len(m.Bindings()) != 0 {
		t.Fatal("a disabled field lists keys")
	}
	m.Disabled = false
	if len(m.Bindings()) != 6 {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
	if _, cmd := m.Update(key(tui.KeyTab)); cmd != nil {
		t.Fatal("Tab was handled")
	}
}

func TestViewShowsCellsCursorAndGroups(t *testing.T) {
	m := New(6)
	m.Group = 3
	if got := plain(m); got != "[ ][ ][ ] — [ ][ ][ ]" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("1234")
	if got := plain(m); got != "[1][2][3] — [4][ ][ ]" {
		t.Errorf("four typed = %q", got)
	}
	m.Focus()
	if got := plain(m); got != "[1][2][3] — [4]< >[ ]" {
		t.Errorf("focused = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) || m.Width() != 21 {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	m.Mask = true
	if got := plain(m); got != "[•][•][•] — [•]< >[ ]" {
		t.Errorf("masked = %q", got)
	}
	m.Mask, m.Disabled = false, true
	if got := plain(m); got != "[1][2][3] — [4][ ][ ]" {
		t.Errorf("disabled = %q", got)
	}
	a := New(2)
	a.Group = 1
	a.Theme.Glyphs = theme.ASCIIGlyphSet()
	if got := plain(a); got != "[ ] - [ ]" {
		t.Errorf("ASCII = %q", got)
	}
	plainRow := New(3)
	if got := plain(plainRow); got != "[ ][ ][ ]" || plainRow.Width() != 9 {
		t.Errorf("no groups = %q, width %d", got, plainRow.Width())
	}
}

func TestAClickMovesTheCursor(t *testing.T) {
	m := New(6)
	m.Group = 3
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 2, W: 21, H: 1}
	press := func(x int, b tui.MouseButton) Model {
		next, cmd := m.Update(tui.MouseEvent{X: x, Y: 2, Button: b, Action: tui.MouseActionPress})
		if cmd != nil {
			t.Fatalf("a click at %d delivered a message", x)
		}
		return next
	}
	if got := press(14, tui.MouseButtonLeft).Cursor(); got != 1 { // "[ ]" at local 3..5
		t.Errorf("click on the second cell: cursor %d", got)
	}
	if got := press(23, tui.MouseButtonLeft).Cursor(); got != 3 { // after the separator, local 12..14
		t.Errorf("click on the fourth cell: cursor %d", got)
	}
	if got := press(20, tui.MouseButtonLeft).Cursor(); got != 0 {
		t.Errorf("click on the separator moved the cursor to %d", got)
	}
	if got := press(60, tui.MouseButtonLeft).Cursor(); got != 0 {
		t.Errorf("click outside Bounds moved the cursor to %d", got)
	}
	if got := press(14, tui.MouseButtonRight).Cursor(); got != 0 {
		t.Errorf("the right button moved the cursor to %d", got)
	}
	m.Bounds.W = 40
	if got := press(45, tui.MouseButtonLeft).Cursor(); got != 0 {
		t.Errorf("click past the last cell moved the cursor to %d", got)
	}
	m.Mouse = false
	if got := press(14, tui.MouseButtonLeft).Cursor(); got != 0 {
		t.Errorf("the pointer moved the cursor with Mouse off")
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := code()
	m.SetValue("12")
	m.KeyMap = KeyMap{Back: keymap.NewBinding("delete", "delete")}
	if _, cmd := m.Update(key(tui.KeyBackspace)); cmd != nil {
		t.Fatal("Backspace still deleted with a custom KeyMap")
	}
	if _, cmd := m.Update(key(tui.KeyDelete)); cmd == nil {
		t.Fatal("the custom key did not delete")
	}
	z := Model{Length: 3, Theme: theme.DarkTheme()}
	z.Focus()
	z, _ = z.Update(typed("5"))
	if next, _ := z.Update(key(tui.KeyLeft)); next.Cursor() != 0 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Left")
	}
}

func TestLinearize(t *testing.T) {
	m := New(6)
	if got := m.Linearize(); got != "code field, 0 of 6 entered" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("123")
	if got := m.Linearize(); got != "code field, 3 of 6 entered: 123" {
		t.Errorf("three = %q", got)
	}
	m.Mask, m.Disabled = true, true
	if got := m.Linearize(); got != "code field, 3 of 6 entered, unavailable" {
		t.Errorf("masked and disabled = %q", got)
	}
}

var _ tui.Linearizer = Model{}

func TestThemeTokensAndLayoutNode(t *testing.T) {
	m := New(4).SetTheme(theme.LightTheme())
	if m.Theme.Primary != theme.LightTheme().Primary {
		t.Error("SetTheme did not apply the theme")
	}
	red := ansi.RGB{R: 255}
	m = m.WithTokens(theme.Tokens{Muted: red})
	if m.Tokens().Muted != red {
		t.Error("WithTokens did not override the muted colour")
	}
	n := m.LayoutNode()
	if s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded}); s != (layout.Size{W: 12, H: 1}) {
		t.Fatalf("Measure = %+v, want 12x1", s)
	}
}

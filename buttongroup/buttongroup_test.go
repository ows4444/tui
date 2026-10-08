package buttongroup

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// msgs runs cmd and returns the message it delivers, as a slice that is
// empty for a nil Cmd.
func msgs(t *testing.T, cmd tui.Cmd) []tui.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	return []tui.Msg{cmd()}
}

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func focused(mode Mode, labels ...string) Model {
	m := New(mode, labels...)
	m.ID = "group"
	m.Focus()
	return m
}

func TestNewBuildsOneButtonPerLabel(t *testing.T) {
	m := New(ModeActions, "Back", "Next")
	if len(m.Buttons) != 2 || m.Buttons[1].ID != "Next" || m.Buttons[0].Toggle {
		t.Fatalf("buttons = %+v", m.Buttons)
	}
	if got := plain(m); got != "[ Back ] [ Next ]" {
		t.Errorf("View = %q", got)
	}
	if m.Width() != ansi.Width(m.View()) {
		t.Errorf("Width() = %d, view is %d", m.Width(), ansi.Width(m.View()))
	}
	if !New(ModeSingle, "a").Buttons[0].Toggle || !New(ModeMultiple, "a").Buttons[0].Toggle {
		t.Error("a choice group's buttons are not toggles")
	}
}

func TestOnlyTheButtonUnderTheCursorShowsFocus(t *testing.T) {
	m := focused(ModeActions, "A", "B", "C")
	for i, want := range []bool{true, false, false} {
		if m.Buttons[i].Focused() != want {
			t.Errorf("button %d focused = %v", i, !want)
		}
	}
	m.Blur()
	if m.Focused() || m.Buttons[0].Focused() {
		t.Error("a button still shows focus after Blur")
	}
}

func TestArrowsMoveWrapAndSkipDisabled(t *testing.T) {
	m := focused(ModeActions, "A", "B", "C", "D")
	m.Buttons[1].Disabled = true
	steps := []struct {
		k    tui.KeyType
		want int
	}{
		{tui.KeyRight, 2}, {tui.KeyRight, 3}, {tui.KeyRight, 0}, // wraps
		{tui.KeyLeft, 3}, {tui.KeyLeft, 2}, {tui.KeyLeft, 0}, // skips B
		{tui.KeyEnd, 3}, {tui.KeyHome, 0},
	}
	for i, s := range steps {
		var cmd tui.Cmd
		m, cmd = m.Update(key(s.k))
		if m.Cursor() != s.want || cmd != nil {
			t.Fatalf("step %d: cursor = %d, want %d", i, m.Cursor(), s.want)
		}
		if !m.Buttons[s.want].Focused() {
			t.Fatalf("step %d: the button under the cursor does not show focus", i)
		}
	}
}

func TestACursorOnADisabledButtonMovesOffIt(t *testing.T) {
	m := New(ModeActions, "A", "B")
	m.Buttons[0].Disabled = true
	m.Focus()
	if m.Cursor() != 1 {
		t.Fatalf("cursor = %d, want 1: the first button is disabled", m.Cursor())
	}
	m.SetCursor(0)
	if m.Cursor() != 1 {
		t.Fatal("SetCursor put the cursor on a disabled button")
	}
	m.SetCursor(9)
	if m.Cursor() != 1 {
		t.Fatal("SetCursor took an index out of range")
	}
}

func TestEnterPressesTheButtonUnderTheCursor(t *testing.T) {
	m := focused(ModeActions, "Back", "Next")
	m, _ = m.Update(key(tui.KeyRight))
	_, cmd := m.Update(key(tui.KeyEnter))
	got := msgs(t, cmd)
	if len(got) != 1 || got[0] != (button.PressedMsg{ID: "Next"}) {
		t.Fatalf("messages = %#v, want one PressedMsg for Next", got)
	}
}

func TestAGroupWithoutFocusIgnoresKeys(t *testing.T) {
	m := New(ModeActions, "A", "B")
	next, cmd := m.Update(key(tui.KeyRight))
	if next.Cursor() != 0 || cmd != nil {
		t.Fatal("a key moved the cursor of a group that does not have focus")
	}
	if _, cmd := m.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("Enter pressed a button of a group that does not have focus")
	}
	var empty Model
	empty.Focus()
	if _, cmd := empty.Update(key(tui.KeyEnter)); cmd != nil {
		t.Fatal("an empty group returned a Cmd")
	}
}

func TestSingleModeKeepsAtMostOneOn(t *testing.T) {
	m := focused(ModeSingle, "Left", "Centre", "Right")
	m, cmd := m.Update(key(tui.KeyEnter))
	if !reflect.DeepEqual(m.On(), []string{"Left"}) {
		t.Fatalf("On() = %v after pressing Left", m.On())
	}
	var changed ChangedMsg
	for _, msg := range msgs(t, cmd) {
		if c, ok := msg.(ChangedMsg); ok {
			changed = c
		}
	}
	if changed.ID != "group" || changed.Pressed != "Left" || !reflect.DeepEqual(changed.On, []string{"Left"}) {
		t.Fatalf("ChangedMsg = %+v", changed)
	}
	m, _ = m.Update(key(tui.KeyRight))
	m, _ = m.Update(key(tui.KeySpace))
	if !reflect.DeepEqual(m.On(), []string{"Centre"}) {
		t.Fatalf("On() = %v after pressing Centre; Left should be off", m.On())
	}
	m, cmd = m.Update(key(tui.KeySpace)) // pressing the one that is on turns it off
	if len(m.On()) != 0 {
		t.Fatalf("On() = %v, want none", m.On())
	}
	if got := msgs(t, cmd); len(got) != 1 || got[0].(ChangedMsg).Pressed != "Centre" || len(got[0].(ChangedMsg).On) != 0 {
		t.Fatalf("turning the last one off delivered %#v, want one ChangedMsg with nothing on", got)
	}
}

func TestRequiredSingleModeCannotBeLeftEmpty(t *testing.T) {
	m := focused(ModeSingle, "S", "M", "L")
	m.Required = true
	m, _ = m.Update(key(tui.KeyEnter))
	m, cmd := m.Update(key(tui.KeyEnter))
	if !reflect.DeepEqual(m.On(), []string{"S"}) || cmd != nil {
		t.Fatalf("On() = %v, cmd = %v; pressing the chosen one again should do nothing", m.On(), cmd != nil)
	}
	m, _ = m.Update(key(tui.KeyRight))
	m, _ = m.Update(key(tui.KeyEnter))
	if !reflect.DeepEqual(m.On(), []string{"M"}) {
		t.Fatalf("On() = %v, want M", m.On())
	}
}

func TestMultipleModeTogglesEachByItself(t *testing.T) {
	m := focused(ModeMultiple, "Bold", "Italic", "Underline")
	m, _ = m.Update(key(tui.KeyEnter))
	m, _ = m.Update(key(tui.KeyEnd))
	m, cmd := m.Update(key(tui.KeyEnter))
	if !reflect.DeepEqual(m.On(), []string{"Bold", "Underline"}) {
		t.Fatalf("On() = %v", m.On())
	}
	found := false
	for _, msg := range msgs(t, cmd) {
		if c, ok := msg.(ChangedMsg); ok && reflect.DeepEqual(c.On, []string{"Bold", "Underline"}) {
			found = true
		}
	}
	if !found {
		t.Fatal("no ChangedMsg with both buttons on")
	}
	if got := plain(m); got != "[●Bold ] [ Italic ] [●Underline ]" {
		t.Errorf("View = %q", got)
	}
}

func TestSetOn(t *testing.T) {
	m := New(ModeMultiple, "a", "b", "c")
	m.SetOn("c", "a", "nope")
	if !reflect.DeepEqual(m.On(), []string{"a", "c"}) {
		t.Fatalf("multiple: On() = %v", m.On())
	}
	m.SetOn()
	if len(m.On()) != 0 {
		t.Fatalf("SetOn() with no IDs left %v on", m.On())
	}
	s := New(ModeSingle, "a", "b", "c")
	s.SetOn("b", "c")
	if !reflect.DeepEqual(s.On(), []string{"b"}) {
		t.Fatalf("single: On() = %v, want only the first", s.On())
	}
	a := New(ModeActions, "a")
	a.SetOn("a")
	if len(a.On()) != 0 {
		t.Fatal("an action group has a button on")
	}
}

// A Model is a value: changing a copy must not change the original through
// the Buttons slice they share.
func TestACopyDoesNotChangeTheOriginal(t *testing.T) {
	m := focused(ModeMultiple, "a", "b")
	before := plain(m)
	c := m
	c, _ = c.Update(key(tui.KeyEnter))
	c, _ = c.Update(key(tui.KeyRight))
	c.SetOn("b")
	c = c.SetTheme(theme.LightTheme())
	c.Blur()
	if plain(m) != before || len(m.On()) != 0 || m.Cursor() != 0 || !m.Buttons[0].Focused() {
		t.Fatalf("the original changed: view %q, on %v, cursor %d", plain(m), m.On(), m.Cursor())
	}
}

func TestACustomKeyMapReplacesTheDefault(t *testing.T) {
	m := focused(ModeActions, "A", "B")
	m.KeyMap = KeyMap{Next: keymap.NewBinding("next", "n")}
	if next, _ := m.Update(key(tui.KeyRight)); next.Cursor() != 0 {
		t.Fatal("Right still moved with a custom KeyMap")
	}
	if next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "n"}); next.Cursor() != 1 {
		t.Fatal("the custom key did not move the cursor")
	}
	z := Model{Buttons: New(ModeActions, "A", "B").Buttons}
	z.Focus()
	if next, _ := z.Update(key(tui.KeyRight)); next.Cursor() != 1 {
		t.Fatal("a struct-literal Model with no KeyMap ignored Right")
	}
}

func TestBindingsListTheGroupsKeysThenTheButtons(t *testing.T) {
	m := focused(ModeActions, "A")
	if b := m.Bindings(); len(b) != 5 || b[4].Desc != "press" {
		t.Fatalf("Bindings() = %+v", b)
	}
	m.Buttons[0].Disabled = true
	if b := m.Bindings(); len(b) != 4 {
		t.Fatalf("Bindings() with every button disabled = %+v", b)
	}
}

func TestGapAndLayoutNode(t *testing.T) {
	m := New(ModeActions, "A", "B")
	m.Gap = 0
	if got := plain(m); got != "[ A ][ B ]" {
		t.Errorf("gap 0 = %q", got)
	}
	m.Gap = -3
	if got := plain(m); got != "[ A ][ B ]" || m.Width() != 10 {
		t.Errorf("negative gap = %q, width %d", got, m.Width())
	}
	m.Gap = 3
	n := m.LayoutNode()
	s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded})
	if s != (layout.Size{W: m.Width(), H: 1}) || n.Render(s) != m.View() {
		t.Fatalf("LayoutNode measures %+v and renders %q", s, n.Render(s))
	}
}

func TestSetThemeReachesEveryButton(t *testing.T) {
	m := New(ModeActions, "A", "B").SetTheme(theme.LightTheme())
	for i, b := range m.Buttons {
		if b.Theme.Primary != theme.LightTheme().Primary {
			t.Errorf("button %d did not get the theme", i)
		}
	}
}

func TestLinearize(t *testing.T) {
	m := focused(ModeMultiple, "Bold", "Italic")
	m, _ = m.Update(key(tui.KeyEnter))
	want := "Bold, toggle button, on, focused, 1 of 2\nItalic, toggle button, off, 2 of 2"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
}

var _ tui.Linearizer = Model{}

// "[ Back ] [ Next ]" drawn at column 5, row 2: Back is columns 5-12, the
// gap is 13, Next is 14-21.
func mouseGroup(mode Mode) Model {
	m := New(mode, "Back", "Next")
	m.ID = "group"
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 17, H: 1}
	return m
}

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func TestAClickPressesTheButtonUnderThePointer(t *testing.T) {
	m := mouseGroup(ModeActions)
	m.Focus()
	m, cmd := m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	if len(msgs(t, cmd)) != 0 {
		t.Fatal("the button fired when the pointer went down")
	}
	if m.Cursor() != 1 || !m.Buttons[1].Focused() || !m.Buttons[1].Down() || m.Buttons[0].Down() {
		t.Fatalf("after pointer down: cursor %d, Next focused %v down %v", m.Cursor(), m.Buttons[1].Focused(), m.Buttons[1].Down())
	}
	m, cmd = m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionRelease))
	got := msgs(t, cmd)
	if len(got) != 1 || got[0] != (button.PressedMsg{ID: "Next"}) {
		t.Fatalf("messages = %#v, want one PressedMsg for Next", got)
	}
}

func TestAClickInTheGapOrOutsidePressesNothing(t *testing.T) {
	for _, x := range []int{13, 40} {
		m := mouseGroup(ModeActions)
		m, _ = m.Update(ev(x, 2, tui.MouseButtonLeft, tui.MouseActionPress))
		_, cmd := m.Update(ev(x, 2, tui.MouseButtonLeft, tui.MouseActionRelease))
		if len(msgs(t, cmd)) != 0 {
			t.Errorf("a click at column %d pressed a button", x)
		}
	}
}

func TestAClickChoosesInSingleMode(t *testing.T) {
	m := mouseGroup(ModeSingle)
	click := func(x int) {
		m, _ = m.Update(ev(x, 2, tui.MouseButtonLeft, tui.MouseActionPress))
		m, _ = m.Update(ev(x, 2, tui.MouseButtonLeft, tui.MouseActionRelease))
	}
	click(6)
	click(15)
	if !reflect.DeepEqual(m.On(), []string{"Next"}) {
		t.Fatalf("On() = %v, want only Next", m.On())
	}
	m.Required = true
	click(15)
	if !reflect.DeepEqual(m.On(), []string{"Next"}) {
		t.Fatalf("On() = %v: a click turned off the required choice", m.On())
	}
}

func TestAClickOnADisabledButtonDoesNothing(t *testing.T) {
	m := mouseGroup(ModeActions)
	m.Buttons[1].Disabled = true
	m, _ = m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	_, cmd := m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionRelease))
	if m.Cursor() != 0 || len(msgs(t, cmd)) != 0 {
		t.Fatal("a click acted on a disabled button")
	}
}

func TestHoverFollowsThePointer(t *testing.T) {
	m := mouseGroup(ModeActions)
	m, _ = m.Update(ev(6, 2, tui.MouseButtonNone, tui.MouseActionMotion))
	if !m.Buttons[0].Hovered() || m.Buttons[1].Hovered() {
		t.Fatal("hover is not on Back")
	}
	m, _ = m.Update(ev(15, 2, tui.MouseButtonNone, tui.MouseActionMotion))
	if m.Buttons[0].Hovered() || !m.Buttons[1].Hovered() {
		t.Fatal("hover did not move to Next")
	}
}

func TestMouseOffIgnoresThePointer(t *testing.T) {
	m := mouseGroup(ModeActions)
	m.Mouse = false
	m, _ = m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	next, cmd := m.Update(ev(15, 2, tui.MouseButtonLeft, tui.MouseActionRelease))
	if cmd != nil || next.Cursor() != 0 {
		t.Fatal("the pointer acted on a group with Mouse off")
	}
}

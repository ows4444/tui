package button

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

func altKey(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s, Mod: input.ModAlt} }

func TestAMarkedLabelDrawsWithoutTheMarkerAndUnderlinesTheLetter(t *testing.T) {
	m := New("&Save")
	if got := plain(m); got != "[ Save ]" {
		t.Errorf("View = %q", got)
	}
	if !strings.Contains(m.View(), m.style().Underline().Render("S")) {
		t.Errorf("S is not underlined in %q", m.View())
	}
	if w, plainW := m.Width(), New("Save").Width(); w != plainW || ansi.Width(m.View()) != w {
		t.Errorf("width %d (View %d), want %d: the marker must take no room", w, ansi.Width(m.View()), plainW)
	}
	if got := plain(New("R&&D")); got != "[ R&D ]" {
		t.Errorf("&& = %q", got)
	}
}

func TestALabelWithoutAMarkerDrawsAsBefore(t *testing.T) {
	m := New("Save")
	want := m.style().Render("[ Save ]")
	if m.View() != want {
		t.Errorf("View = %q, want one styled run %q", m.View(), want)
	}
	if m.Mnemonic() != 0 {
		t.Errorf("Mnemonic = %q", m.Mnemonic())
	}
}

func TestAltAndTheLetterPressTheButtonWithOrWithoutFocus(t *testing.T) {
	m := New("&Save")
	m.ID = "save"
	if m.Mnemonic() != 's' {
		t.Fatalf("Mnemonic = %q", m.Mnemonic())
	}
	for _, focus := range []bool{false, true} {
		if focus {
			m.Focus()
		}
		for _, k := range []tui.Key{altKey("s"), altKey("S")} {
			if _, cmd := m.Update(k); func() bool { p, ok := pressed(cmd); return !ok || p.ID != "save" }() {
				t.Errorf("focus %v: %v did not press", focus, k)
			}
		}
	}
	for _, k := range []tui.Key{altKey("x"), {Type: tui.KeyRunes, Text: "s"}, {Type: tui.KeyRunes, Text: "s", Mod: input.ModCtrl}} {
		if _, cmd := m.Update(k); cmd != nil {
			t.Errorf("%v pressed the button", k)
		}
	}
}

func TestAHotKeyOnADisabledOrLoadingButtonDoesNothingAndIsNotShown(t *testing.T) {
	for name, m := range map[string]Model{
		"disabled": func() Model { m := New("&Save"); m.Disabled = true; return m }(),
		"loading":  func() Model { m := New("&Save"); m.Loading = true; return m }(),
	} {
		if _, cmd := m.Update(altKey("s")); cmd != nil {
			t.Errorf("%s: the hot key pressed it", name)
		}
		if m.IsMnemonic(altKey("s")) {
			t.Errorf("%s: IsMnemonic is true", name)
		}
		if strings.Contains(m.View(), "4m") && strings.Contains(m.View(), ";4m") {
			t.Errorf("%s: a hot key that does nothing is drawn underlined", name)
		}
		if strings.Contains(m.Linearize(), "shortcut") {
			t.Errorf("%s: Linearize announces a shortcut", name)
		}
	}
}

func TestHotKeyInToggleAndBindingsAndLinearize(t *testing.T) {
	m := New("&Bold")
	m.Toggle = true
	m, _ = m.Update(altKey("b"))
	if !m.On() {
		t.Error("the hot key did not flip a toggle")
	}
	if got := m.Linearize(); got != "Bold, toggle button, on, shortcut Alt+B" {
		t.Errorf("Linearize = %q", got)
	}
	found := false
	for _, b := range m.Bindings() {
		if len(b.Keys) == 1 && b.Keys[0] == "alt+b" {
			found = true
		}
	}
	if !found {
		t.Errorf("Bindings = %+v, want alt+b", m.Bindings())
	}
	if len(New("Plain").Bindings()) != 1 {
		t.Error("a label without a marker gained a binding")
	}
}

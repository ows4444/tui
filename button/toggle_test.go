package button

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/theme"
)

func toggle(label string) Model {
	m := New(label)
	m.ID = label
	m.Toggle = true
	m.Focus()
	return m
}

func TestAPressTurnsAToggleOnAndOff(t *testing.T) {
	m := toggle("Bold")
	if m.On() {
		t.Fatal("a new toggle is on")
	}
	m, cmd := m.Update(key(tui.KeyEnter))
	if p, ok := pressed(cmd); !ok || !p.On || p.ID != "Bold" || !m.On() {
		t.Fatalf("first press: PressedMsg = %+v, On() = %v; want on", p, m.On())
	}
	m, cmd = m.Update(key(tui.KeySpace))
	if p, ok := pressed(cmd); !ok || p.On || m.On() {
		t.Fatalf("second press: PressedMsg = %+v, On() = %v; want off", p, m.On())
	}
}

func TestAClickTurnsAToggleOn(t *testing.T) {
	m := toggle("Bold")
	m.Mouse, m.Bounds = true, hittest.Rect{W: 8, H: 1}
	m, _ = m.Update(ev(1, 0, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.On() {
		t.Fatal("it turned on when the pointer went down, before it came up")
	}
	m, cmd := m.Update(ev(1, 0, tui.MouseButtonLeft, tui.MouseActionRelease))
	if p, ok := pressed(cmd); !ok || !p.On || !m.On() {
		t.Fatalf("click: PressedMsg = %+v, On() = %v", p, m.On())
	}
}

func TestSetOnDeliversNoPress(t *testing.T) {
	m := toggle("Bold")
	m.SetOn(true)
	if !m.On() {
		t.Fatal("SetOn(true) left it off")
	}
	m.SetOn(false)
	if m.On() {
		t.Fatal("SetOn(false) left it on")
	}
}

func TestAPlainButtonIsNeverOn(t *testing.T) {
	m := New("Save")
	m.Focus()
	m.SetOn(true)
	if m.On() {
		t.Fatal("SetOn turned on a button that is not a toggle")
	}
	m, cmd := m.Update(key(tui.KeyEnter))
	if p, _ := pressed(cmd); p.On || m.On() {
		t.Fatal("a press turned on a button that is not a toggle")
	}
}

func TestADisabledToggleKeepsItsState(t *testing.T) {
	m := toggle("Bold")
	m.SetOn(true)
	m.Disabled = true
	m, cmd := m.Update(key(tui.KeyEnter))
	if cmd != nil || !m.On() {
		t.Fatal("a disabled toggle changed or fired")
	}
}

// The dot that marks a toggle on takes a cell the off state leaves blank, so
// on and off read differently without colour and are the same width.
func TestToggleViewMarksOnWithoutChangingWidth(t *testing.T) {
	for size, want := range map[Size][2]string{
		SizeDefault: {"[ Bold ]", "[●Bold ]"},
		SizeSmall:   {"[ Bold]", "[●Bold]"},
		SizeLarge:   {"[   Bold   ]", "[  ●Bold   ]"},
	} {
		m := New("Bold")
		m.Toggle, m.Size = true, size
		if got := plain(m); got != want[0] {
			t.Errorf("size %d off = %q, want %q", size, got, want[0])
		}
		off := m.Width()
		m.SetOn(true)
		if got := plain(m); got != want[1] {
			t.Errorf("size %d on = %q, want %q", size, got, want[1])
		}
		if m.Width() != off || ansi.Width(m.View()) != off {
			t.Errorf("size %d: width changed from %d to %d (view %d)", size, off, m.Width(), ansi.Width(m.View()))
		}
	}
	m := New("Bold")
	m.Toggle = true
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	m.SetOn(true)
	if got := plain(m); got != "[*Bold ]" {
		t.Errorf("ASCII on = %q", got)
	}
}

func TestToggleLinearize(t *testing.T) {
	m := New("Bold")
	m.Toggle = true
	if got := m.Linearize(); got != "Bold, toggle button, off" {
		t.Errorf("off = %q", got)
	}
	m.SetOn(true)
	m.Focus()
	if got := plain(m); got != "<●Bold >" {
		t.Errorf("on and focused view = %q", got)
	}
	if got := m.Linearize(); got != "Bold, toggle button, on, focused" {
		t.Errorf("on and focused = %q", got)
	}
}

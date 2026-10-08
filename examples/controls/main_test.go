package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/tuitest"
)

// drain applies msg and then every message its Cmd delivers, as a Program
// would, and returns the model that results. A batch cannot be taken apart
// outside package tui, so mouse events, which come back as one, go through
// tuitest in the tests below.
func send(m model, msgs ...tui.Msg) model {
	for _, msg := range msgs {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd != nil {
			if out := cmd(); out != nil {
				next, _ = m.Update(out)
				m = next.(model)
			}
		}
	}
	return m
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

var (
	tab      = key(tui.KeyTab)
	shiftTab = tui.Key{Type: tui.KeyTab, Mod: input.ModShift}
)

func plain(m model) string { return ansi.StripANSI(m.View()) }

func TestTabMovesBetweenTheRowsAndWraps(t *testing.T) {
	m := initialModel()
	if m.focus != rowFormat || !m.format.Focused() {
		t.Fatal("the first row is not focused at start")
	}
	m = send(m, tab)
	if m.focus != rowOptions || !m.options.Focused() || m.format.Focused() {
		t.Fatalf("tab: focus=%d", m.focus)
	}
	m = send(m, shiftTab, shiftTab)
	if m.focus != rowActions || !m.actions.Focused() {
		t.Fatalf("shift+tab past the first row: focus=%d, want the last", m.focus)
	}
	m = send(m, tab)
	if m.focus != rowFormat {
		t.Fatalf("tab past the last row: focus=%d, want the first", m.focus)
	}
}

func TestEachRowTakesItsOwnKeys(t *testing.T) {
	m := initialModel()
	m = send(m, key(tui.KeyRight)) // format: PNG -> JPEG
	if m.format.Value() != "JPEG" {
		t.Fatalf("format = %q", m.format.Value())
	}
	m = send(m, tab, key(tui.KeyRight), key(tui.KeySpace)) // options: Resize on
	if got := strings.Join(m.options.On(), ","); got != "Strip,Resize" {
		t.Fatalf("options = %q", got)
	}
	m = send(m, tab, key(tui.KeyRight), key(tui.KeyRight)) // quality: 80 -> 90
	if m.quality.Value() != 90 {
		t.Fatalf("quality = %v", m.quality.Value())
	}
	m = send(m, tab, tui.Key{Type: tui.KeyRunes, Text: "5"}) // rating: 5
	if m.stars.Value() != 5 {
		t.Fatalf("rating = %d", m.stars.Value())
	}
	m = send(m, tab, key(tui.KeyEnter)) // preview on
	if !m.preview.On() {
		t.Fatal("the preview toggle is off")
	}
	want := "JPEG · strip, resize · quality 90 · 5/5 · preview"
	if got := plain(m); !strings.Contains(got, want) {
		t.Fatalf("the status line does not read %q:\n%s", want, got)
	}
}

func TestExportAndReset(t *testing.T) {
	m := initialModel()
	m = send(m, key(tui.KeyEnd))             // WebP
	m = send(m, shiftTab, key(tui.KeyEnter)) // actions: Export
	if !strings.Contains(plain(m), "Exported: WebP · strip · quality 80 · 3/5") {
		t.Fatalf("no export line:\n%s", plain(m))
	}
	m = send(m, key(tui.KeyRight), key(tui.KeyEnter)) // Reset
	if m.format.Value() != "PNG" || m.exported != "" {
		t.Fatalf("after Reset: format %q, exported %q", m.format.Value(), m.exported)
	}
	if got := send(m, button.PressedMsg{ID: "preview"}); got.exported != "" {
		t.Fatal("the preview button's press was taken for Export")
	}
}

func TestNoOptionsReadsAsSuch(t *testing.T) {
	m := send(initialModel(), tab, key(tui.KeySpace)) // Strip off
	if !strings.Contains(plain(m), "no options") {
		t.Fatalf("status:\n%s", plain(m))
	}
}

func TestEscQuits(t *testing.T) {
	for _, k := range []tui.Key{key(tui.KeyEsc), {Type: tui.KeyRunes, Text: "c", Mod: input.ModCtrl}} {
		if _, cmd := initialModel().Update(k); cmd == nil {
			t.Errorf("%v did not quit", k)
		} else if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Errorf("%v returned a Cmd that is not Quit", k)
		}
	}
}

func TestNoLineIsWiderThanANarrowTerminal(t *testing.T) {
	m := send(initialModel(), tui.ResizeMsg{Width: 30, Height: 8})
	for i, l := range strings.Split(plain(m), "\n") {
		if w := ansi.Width(l); w > 30 {
			t.Errorf("line %d is %d cells wide: %q", i, w, l)
		}
	}
	if (model{}).contentWidth() != 60 {
		t.Error("contentWidth before the first resize is not 60")
	}
}

// session runs the program at 80x24 with the mouse on, as main does.
func session(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(initialModel(), 80, 24, tui.WithMouse(tui.MouseCellMotion))
	t.Cleanup(s.Close)
	return s
}

func waitFor(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, 3*time.Second) {
		t.Fatalf("the screen never showed %q:\n%s", text, strings.Join(s.Screen(), "\n"))
	}
}

// At 80x24 the rows are on screen rows 2 to 7 and the controls start at
// column 9: "<●> PNG  ( ) JPEG  ( ) WebP", so JPEG's mark is at column 18.
func TestTheMouseClicksEveryKindOfControl(t *testing.T) {
	s := session(t)
	s.Click(18, 2) // format: JPEG
	waitFor(t, s, "JPEG · strip")
	s.Click(21, 3) // options: Resize, the second button
	waitFor(t, s, "strip, resize")
	s.Click(10, 4) // quality: the first cell of the track
	waitFor(t, s, "quality 0 ")
	s.Click(14, 5) // rating: the fifth mark
	waitFor(t, s, "5/5")
	s.Click(12, 6) // preview toggle
	waitFor(t, s, "· preview")
	s.Click(12, 7) // actions: Export
	waitFor(t, s, "Exported: JPEG · strip, resize · quality 0 · 5/5 · preview")
	s.Click(22, 7) // actions: Reset
	waitFor(t, s, "PNG · strip · quality 80 · 3/5")
}

func TestAClickMovesFocusToItsRow(t *testing.T) {
	s := session(t)
	s.Click(14, 5)
	waitFor(t, s, "[●●●●●]") // the rating shows focus with brackets
	s.Keys("left")
	waitFor(t, s, "4/5")
}

package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// send passes each key to Update and returns the model that results.
func send(m model, keys ...tui.Key) model {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(model)
	}
	return m
}

// typed is one key press per character of s.
func typed(s string) []tui.Key {
	var ks []tui.Key
	for _, r := range s {
		k := tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r}
		if r == ' ' {
			k.Type = tui.KeySpace
		}
		ks = append(ks, k)
	}
	return ks
}

var (
	tab      = tui.Key{Type: tui.KeyTab}
	shiftTab = tui.Key{Type: tui.KeyTab, Mod: input.ModShift}
	enter    = tui.Key{Type: tui.KeyEnter}
)

func TestTabMovesFocusAndWraps(t *testing.T) {
	m := initialModel()
	if !m.port.Focused() {
		t.Fatal("the first field is not focused at start")
	}
	m = send(m, tab)
	if m.focus != fieldEmail || !m.email.Focused() || m.port.Focused() {
		t.Fatalf("tab: focus=%d, email focused=%v, port focused=%v", m.focus, m.email.Focused(), m.port.Focused())
	}
	m = send(m, shiftTab, shiftTab)
	if m.focus != fieldTags || !m.tags.Input.Focused() {
		t.Fatalf("shift+tab past the first field: focus=%d, want the last", m.focus)
	}
	m = send(m, tab)
	if m.focus != fieldPort {
		t.Fatalf("tab past the last field: focus=%d, want the first", m.focus)
	}
}

func TestEnterMovesOnExceptInTheTagField(t *testing.T) {
	m := send(initialModel(), enter)
	if m.focus != fieldEmail {
		t.Fatalf("enter in the port field: focus=%d, want the email field", m.focus)
	}
}

func TestPortTakesDigitsOnly(t *testing.T) {
	m := send(initialModel(), typed("8a0 8-0")...)
	if got := m.port.Value(); got != "8080" {
		t.Fatalf("port = %q, want %q", got, "8080")
	}
}

func TestEmailRejectsSpacesAndReportsValidity(t *testing.T) {
	m := send(initialModel(), tab)
	m = send(m, typed("ops @example")...)
	if got := m.email.Value(); got != "ops@example" {
		t.Fatalf("email = %q, want the space dropped", got)
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "email incomplete") {
		t.Errorf("status does not say the email is incomplete:\n%s", out)
	}
	m = send(m, typed(".com")...)
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "email ok") {
		t.Errorf("status does not say the email is ok:\n%s", out)
	}
}

func TestTokenIsNeverDrawn(t *testing.T) {
	m := send(initialModel(), tab, tab)
	m = send(m, typed("s3cret")...)
	if got := m.token.Value(); got != "s3cret" {
		t.Fatalf("token = %q", got)
	}
	out := ansi.StripANSI(m.View())
	if strings.Contains(out, "s3cret") {
		t.Fatalf("the token is on screen:\n%s", out)
	}
	if !strings.Contains(out, "token 6 chars") {
		t.Errorf("status does not give the token's length:\n%s", out)
	}
}

func TestTagsAreAddedAndRemoved(t *testing.T) {
	m := send(initialModel(), shiftTab)
	m = send(m, typed("api")...)
	m = send(m, enter)
	m = send(m, typed("eu")...)
	m = send(m, enter)
	if got := strings.Join(m.tags.Tags, ","); got != "api,eu" {
		t.Fatalf("tags = %q, want %q", got, "api,eu")
	}
	if m.focus != fieldTags {
		t.Fatal("enter in the tag field moved focus")
	}
	m = send(m, tui.Key{Type: tui.KeyBackspace})
	if got := strings.Join(m.tags.Tags, ","); got != "api" {
		t.Fatalf("after backspace on an empty input: tags = %q, want %q", got, "api")
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "1 of 5 labels") {
		t.Errorf("status does not count the labels:\n%s", out)
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{"esc": {Type: tui.KeyEsc}, "ctrl+c": {Type: tui.KeyCtrlC}} {
		_, cmd := initialModel().Update(k)
		if cmd == nil {
			t.Fatalf("%s returned no Cmd", name)
		}
		if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Errorf("%s did not quit", name)
		}
	}
}

// A long value scrolls inside its field: no row grows past the content width.
func TestLongValuesStayInsideTheRow(t *testing.T) {
	next, _ := initialModel().Update(tui.ResizeMsg{Width: 40, Height: 10})
	m := send(next.(model), tab)
	m = send(m, typed(strings.Repeat("longname", 8)+"@example.com")...)
	for _, line := range strings.Split(ansi.StripANSI(m.View()), "\n") {
		if w := ansi.Width(line); w > 40 {
			t.Errorf("a line is %d wide in a 40-column terminal: %q", w, line)
		}
	}
}

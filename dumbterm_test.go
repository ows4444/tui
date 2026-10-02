package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ows4444/tui/input"
)

func TestDumbTermDefaults(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if p := NewProgram(nil); !p.accessible || !p.Accessible() {
		t.Fatal("TERM=dumb without mode options should select the plain renderer")
	}
	if p := NewProgram(nil, WithAltScreen(true)); p.accessible || !p.altScreen {
		t.Fatal("explicit WithAltScreen must be honoured under TERM=dumb")
	}
	if p := NewProgram(nil, WithAccessible(false)); p.accessible {
		t.Fatal("explicit WithAccessible(false) must be honoured under TERM=dumb")
	}
	t.Setenv("TERM", "xterm-256color")
	if p := NewProgram(nil); p.accessible || !p.altScreen {
		t.Fatal("non-dumb TERM must keep the defaults")
	}
}

func TestDumbTermWritesNoModeSequences(t *testing.T) {
	t.Setenv("TERM", "dumb")
	var buf bytes.Buffer
	p := NewProgram(nil, WithOutput(&buf))
	p.enterModes()
	for _, bad := range []string{"\x1b[?1049h", "\x1b[?25l", "\x1b[?2026h"} {
		if strings.Contains(buf.String(), bad) {
			t.Fatalf("output contains %q: %q", bad, buf.String())
		}
	}
	if buf.Len() != 0 {
		t.Fatalf("enterModes wrote %q", buf.String())
	}
}

func TestRootAliasesEveryKeyType(t *testing.T) {
	all := map[input.KeyType]KeyType{
		input.KeyRunes: KeyRunes, input.KeyUp: KeyUp, input.KeyDown: KeyDown,
		input.KeyLeft: KeyLeft, input.KeyRight: KeyRight, input.KeyEnter: KeyEnter,
		input.KeyEsc: KeyEsc, input.KeyTab: KeyTab, input.KeyBackspace: KeyBackspace,
		input.KeyDelete: KeyDelete, input.KeySpace: KeySpace, input.KeyHome: KeyHome,
		input.KeyEnd: KeyEnd, input.KeyPgUp: KeyPgUp, input.KeyPgDown: KeyPgDown,
		input.KeyF1: KeyF1, input.KeyF2: KeyF2, input.KeyF3: KeyF3, input.KeyF4: KeyF4,
		input.KeyF5: KeyF5, input.KeyF6: KeyF6, input.KeyF7: KeyF7, input.KeyF8: KeyF8,
		input.KeyF9: KeyF9, input.KeyF10: KeyF10, input.KeyF11: KeyF11, input.KeyF12: KeyF12,
		input.KeyF13: KeyF13, input.KeyF14: KeyF14, input.KeyF15: KeyF15, input.KeyF16: KeyF16,
		input.KeyF17: KeyF17, input.KeyF18: KeyF18, input.KeyF19: KeyF19, input.KeyF20: KeyF20,
		input.KeyF21: KeyF21, input.KeyF22: KeyF22, input.KeyF23: KeyF23, input.KeyF24: KeyF24,
		input.KeyCtrlC: KeyCtrlC, input.KeyCtrl: KeyCtrl, input.KeyUnknown: KeyUnknown,
		input.KeyInsert: KeyInsert,
	}
	// KeyInsert is the last constant; every value 0..KeyInsert must be covered.
	if len(all) != int(input.KeyInsert)+1 {
		t.Fatalf("root aliases %d of %d input.KeyType constants", len(all), int(input.KeyInsert)+1)
	}
	for in, root := range all {
		if in != root {
			t.Errorf("alias mismatch: %v != %v", in, root)
		}
	}
}

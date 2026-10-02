package input

import (
	"strings"
	"testing"
)

func readWith(t *testing.T, report bool, s string, n int) []Key {
	t.Helper()
	rd := NewReader(strings.NewReader(s))
	rd.SetReportEvents(report)
	var out []Key
	for i := 0; i < n; i++ {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent %d: %v", i, err)
		}
		out = append(out, ev.(Key))
	}
	return out
}

// criterion #64: with report-events on, release and repeat are distinct.
func TestKeyActionReportedWhenEnabled(t *testing.T) {
	tests := []struct {
		in   string
		typ  KeyType
		act  KeyAction
		mod  Mod
		name string
	}{
		{"\x1b[97;1:3u", KeyRunes, KeyRelease, ModNone, "release rune"},
		{"\x1b[97;1:2u", KeyRunes, KeyRepeat, ModNone, "repeat rune"},
		{"\x1b[97;1:1u", KeyRunes, KeyPress, ModNone, "explicit press"},
		{"\x1b[13;2:3u", KeyEnter, KeyRelease, ModShift, "release shift+enter"},
		{"\x1b[1;1:3A", KeyUp, KeyRelease, ModNone, "release arrow"},
		{"\x1b[1;5:2A", KeyUp, KeyRepeat, ModCtrl, "repeat ctrl+up"},
		{"\x1b[3;1:3~", KeyDelete, KeyRelease, ModNone, "release delete"},
		{"\x1b[97;5:3u", KeyCtrl, KeyRelease, ModNone, "release ctrl+a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := readWith(t, true, tt.in, 1)[0]
			if k.Type != tt.typ || k.Action != tt.act || k.Mod != tt.mod {
				t.Errorf("got %+v, want type %v action %v mod %v", k, tt.typ, tt.act, tt.mod)
			}
			if k.Type == KeyUnknown {
				t.Error("must not be KeyUnknown")
			}
		})
	}
}

// Off by default: release is KeyUnknown, repeat a press, Action never set.
func TestKeyActionOffKeepsLegacyBehaviour(t *testing.T) {
	ks := readWith(t, false, "\x1b[97;1:3u\x1b[97;1:2u\x1b[1;1:3A\x1b[97;1:1u", 4)
	if ks[0].Type != KeyUnknown {
		t.Errorf("release = %+v, want KeyUnknown", ks[0])
	}
	if ks[1].Type != KeyRunes || ks[1].Action != KeyPress {
		t.Errorf("repeat = %+v, want plain press", ks[1])
	}
	if ks[2].Type != KeyUnknown {
		t.Errorf("arrow release = %+v, want KeyUnknown", ks[2])
	}
	if ks[3].Action != KeyPress {
		t.Errorf("press = %+v", ks[3])
	}
}

func TestKeyActionString(t *testing.T) {
	if KeyPress.String() != "press" || KeyRepeat.String() != "repeat" || KeyRelease.String() != "release" {
		t.Error("KeyAction.String")
	}
	if got := (Key{Type: KeyUp, Action: KeyRelease}).String(); got != "up" {
		t.Errorf("Key.String changed with Action: %q", got)
	}
}

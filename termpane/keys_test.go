package termpane

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
)

func TestEncodeKey(t *testing.T) {
	r := func(s string, m input.Mod) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s, Mod: m} }
	for _, c := range []struct {
		name string
		key  tui.Key
		app  bool
		want string
	}{
		{"letter", r("a", 0), false, "a"},
		{"unicode", r("é", 0), false, "é"},
		{"alt letter", r("a", input.ModAlt), false, "\x1ba"},
		{"ctrl letter", r("a", input.ModCtrl), false, "\x01"},
		{"ctrl bracket", r("[", input.ModCtrl), false, "\x1b"},
		{"ctrl at", r("@", input.ModCtrl), false, "\x00"},
		{"KeyCtrl", tui.Key{Type: tui.KeyCtrl, Code: 'd'}, false, "\x04"},
		{"ctrl+c", tui.Key{Type: tui.KeyCtrlC}, false, "\x03"},
		{"enter", tui.Key{Type: tui.KeyEnter}, false, "\r"},
		{"tab", tui.Key{Type: tui.KeyTab}, false, "\t"},
		{"shift tab", tui.Key{Type: tui.KeyTab, Mod: input.ModShift}, false, "\x1b[Z"},
		{"backspace", tui.Key{Type: tui.KeyBackspace}, false, "\x7f"},
		{"esc", tui.Key{Type: tui.KeyEsc}, false, "\x1b"},
		{"space", tui.Key{Type: tui.KeySpace}, false, " "},
		{"ctrl space", tui.Key{Type: tui.KeySpace, Mod: input.ModCtrl}, false, "\x00"},
		{"up", tui.Key{Type: tui.KeyUp}, false, "\x1b[A"},
		{"up app", tui.Key{Type: tui.KeyUp}, true, "\x1bOA"},
		{"right", tui.Key{Type: tui.KeyRight}, false, "\x1b[C"},
		{"ctrl left", tui.Key{Type: tui.KeyLeft, Mod: input.ModCtrl}, false, "\x1b[1;5D"},
		{"ctrl left app", tui.Key{Type: tui.KeyLeft, Mod: input.ModCtrl}, true, "\x1b[1;5D"},
		{"shift alt down", tui.Key{Type: tui.KeyDown, Mod: input.ModShift | input.ModAlt}, false, "\x1b[1;4B"},
		{"home", tui.Key{Type: tui.KeyHome}, false, "\x1b[H"},
		{"end app", tui.Key{Type: tui.KeyEnd}, true, "\x1bOF"},
		{"delete", tui.Key{Type: tui.KeyDelete}, false, "\x1b[3~"},
		{"insert", tui.Key{Type: tui.KeyInsert}, false, "\x1b[2~"},
		{"pgup", tui.Key{Type: tui.KeyPgUp}, false, "\x1b[5~"},
		{"ctrl pgdn", tui.Key{Type: tui.KeyPgDown, Mod: input.ModCtrl}, false, "\x1b[6;5~"},
		{"f1", tui.Key{Type: tui.KeyF1}, false, "\x1bOP"},
		{"shift f4", tui.Key{Type: tui.KeyF4, Mod: input.ModShift}, false, "\x1b[1;2S"},
		{"f5", tui.Key{Type: tui.KeyF5}, false, "\x1b[15~"},
		{"f12", tui.Key{Type: tui.KeyF12}, false, "\x1b[24~"},
		{"unknown", tui.Key{Type: tui.KeyUnknown}, false, ""},
	} {
		if got := encodeKey(c.key, c.app); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

func TestEncodePaste(t *testing.T) {
	if got := encodePaste("a\nb", true); got != "\x1b[200~a\nb\x1b[201~" {
		t.Errorf("bracketed: %q", got)
	}
	if got := encodePaste("a\r\nb\nc", false); got != "a\rb\rc" {
		t.Errorf("plain: %q", got)
	}
	if got := encodePaste("x\x1b[201~y\x1b[200~z", true); got != "\x1b[200~xyz\x1b[201~" {
		t.Errorf("markers in the text: %q", got)
	}
}

func TestEncodeMouse(t *testing.T) {
	ev := func(b tui.MouseButton, a tui.MouseAction, m input.Mod) tui.MouseEvent {
		return tui.MouseEvent{Button: b, Action: a, Mod: m}
	}
	for _, c := range []struct {
		name string
		ev   tui.MouseEvent
		mode int
		want string
	}{
		{"press", ev(tui.MouseButtonLeft, tui.MouseActionPress, 0), 1, "\x1b[<0;5;3M"},
		{"release", ev(tui.MouseButtonLeft, tui.MouseActionRelease, 0), 1, "\x1b[<0;5;3m"},
		{"right", ev(tui.MouseButtonRight, tui.MouseActionPress, 0), 1, "\x1b[<2;5;3M"},
		{"wheel up", ev(tui.MouseButtonWheelUp, tui.MouseActionPress, 0), 1, "\x1b[<64;5;3M"},
		{"drag in click mode", ev(tui.MouseButtonLeft, tui.MouseActionMotion, 0), 1, ""},
		{"drag", ev(tui.MouseButtonLeft, tui.MouseActionMotion, 0), 2, "\x1b[<32;5;3M"},
		{"modifiers", ev(tui.MouseButtonLeft, tui.MouseActionPress, input.ModShift|input.ModAlt|input.ModCtrl), 1, "\x1b[<28;5;3M"},
	} {
		if got := encodeMouse(c.ev, 4, 2, c.mode); got != c.want {
			t.Errorf("%s: %q; want %q", c.name, got, c.want)
		}
	}
}

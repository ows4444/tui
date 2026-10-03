package tuitest

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/input"
)

// decodeOne decodes the first key of seq with the library's own decoder.
func decodeOne(t *testing.T, seq string) input.Key {
	t.Helper()
	ev, err := input.NewReader(strings.NewReader(seq)).ReadEvent()
	if err != nil {
		t.Fatalf("decode %q: %v", seq, err)
	}
	k, ok := ev.(input.Key)
	if !ok {
		t.Fatalf("decode %q: %T, want a Key", seq, ev)
	}
	return k
}

// Every name input.Key.String produces is one Keys understands: encoding the
// name and decoding the bytes gives a key with that same name. The xterm
// sequences of the navigation and function keys have no super bit, so those
// combinations are the one exception.
func TestKeysEncodesEveryKeyName(t *testing.T) {
	var keys []input.Key
	named := []input.KeyType{
		input.KeyUp, input.KeyDown, input.KeyLeft, input.KeyRight, input.KeyHome, input.KeyEnd,
		input.KeyPgUp, input.KeyPgDown, input.KeyInsert, input.KeyDelete,
		input.KeyEnter, input.KeyTab, input.KeyEsc, input.KeyBackspace, input.KeySpace,
	}
	for kt := input.KeyF1; kt <= input.KeyF4; kt++ {
		named = append(named, kt)
	}
	for kt := input.KeyF5; kt <= input.KeyF24; kt++ {
		named = append(named, kt)
	}
	for kt := input.KeyMediaPlay; kt <= input.KeyMediaMute; kt++ {
		named = append(named, kt)
	}
	mods := []input.Mod{input.ModShift, input.ModAlt, input.ModCtrl, input.ModSuper}
	for combo := 0; combo < 1<<len(mods); combo++ {
		var m input.Mod
		for i, bit := range mods {
			if combo&(1<<i) != 0 {
				m |= bit
			}
		}
		for _, kt := range named {
			keys = append(keys, input.Key{Type: kt, Mod: m})
		}
		for _, r := range "a1+" {
			keys = append(keys, input.Key{Type: input.KeyRunes, Text: string(r), Code: r, Mod: m})
		}
	}
	for r := 'a'; r <= 'z'; r++ {
		keys = append(keys, decodeOne(t, string(r-'a'+1))) // the ctrl+<letter> keys
	}

	checked := 0
	for _, k := range keys {
		name := k.String()
		enc := encodeKey(name)
		if enc == name && len([]rune(name)) > 1 {
			xtermOnly := k.Mod.Super() && (csiLetterKeys[strings.TrimPrefix(name, k.Mod.String()+"+")] != 0 || tildeKeys[strings.TrimPrefix(name, k.Mod.String()+"+")] != 0)
			if !xtermOnly {
				t.Errorf("%q is typed as text, not sent as a key", name)
			}
			continue
		}
		if got := decodeOne(t, enc).String(); got != name {
			t.Errorf("Keys(%q) sends %q, which decodes to %q", name, enc, got)
		}
		checked++
	}
	if checked < 600 {
		t.Fatalf("only %d key names were checked", checked)
	}
}

// The keys the audit named arrive as keys, not as their letters.
func TestKeysSendsNamedKeysToTheModel(t *testing.T) {
	for _, name := range []string{"f1", "f12", "ctrl+left", "shift+up", "insert", "ctrl+shift+a", "alt+x", "ctrl+c"} {
		if got := decodeOne(t, encodeKey(name)).String(); got != name {
			t.Errorf("%q arrives as %q", name, got)
		}
	}
}

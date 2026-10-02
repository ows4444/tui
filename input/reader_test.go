package input

import (
	"strings"
	"testing"
)

func readAll(t *testing.T, s string, n int) []Event {
	t.Helper()
	rd := NewReader(strings.NewReader(s))
	events := make([]Event, 0, n)
	for i := 0; i < n; i++ {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatalf("ReadEvent() #%d: unexpected error: %v (got %d events so far: %v)", i, err, len(events), events)
		}
		events = append(events, ev)
	}
	return events
}

func readAllKeys(t *testing.T, s string, n int) []Key {
	t.Helper()
	events := readAll(t, s, n)
	keys := make([]Key, len(events))
	for i, ev := range events {
		k, ok := ev.(Key)
		if !ok {
			t.Fatalf("event #%d = %#v (%T), want Key", i, ev, ev)
		}
		keys[i] = k
	}
	return keys
}

func TestReadKeySingle(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Key
	}{
		{"lowercase rune", "a", Key{Type: KeyRunes, Text: "a", Code: 'a'}},
		{"digit rune", "5", Key{Type: KeyRunes, Text: "5", Code: '5'}},
		{"utf8 2-byte", "é", Key{Type: KeyRunes, Text: "é", Code: 'é'}},
		{"utf8 3-byte", "€", Key{Type: KeyRunes, Text: "€", Code: '€'}},
		{"utf8 4-byte emoji", "😀", Key{Type: KeyRunes, Text: "😀", Code: '😀'}},
		{"space", " ", Key{Type: KeySpace, Text: " ", Code: ' '}},
		{"enter (CR)", "\r", Key{Type: KeyEnter}},
		{"enter (LF)", "\n", Key{Type: KeyEnter}},
		{"tab", "\t", Key{Type: KeyTab}},
		{"backspace (DEL)", "\x7f", Key{Type: KeyBackspace}},
		{"backspace (BS)", "\x08", Key{Type: KeyBackspace}},
		{"ctrl+c", "\x03", Key{Type: KeyCtrlC}},
		{"ctrl+a", "\x01", Key{Type: KeyCtrl, Code: 'a'}},
		{"ctrl+z", "\x1a", Key{Type: KeyCtrl, Code: 'z'}},
		{"bare esc", "\x1b", Key{Type: KeyEsc}},
		{"up arrow", "\x1b[A", Key{Type: KeyUp}},
		{"down arrow", "\x1b[B", Key{Type: KeyDown}},
		{"right arrow", "\x1b[C", Key{Type: KeyRight}},
		{"left arrow", "\x1b[D", Key{Type: KeyLeft}},
		{"home (letter form)", "\x1b[H", Key{Type: KeyHome}},
		{"end (letter form)", "\x1b[F", Key{Type: KeyEnd}},
		{"home (SS3)", "\x1bOH", Key{Type: KeyHome}},
		{"end (SS3)", "\x1bOF", Key{Type: KeyEnd}},
		{"home (numeric ~)", "\x1b[1~", Key{Type: KeyHome}},
		{"home (numeric 7~)", "\x1b[7~", Key{Type: KeyHome}},
		{"delete", "\x1b[3~", Key{Type: KeyDelete}},
		{"end (numeric 4~)", "\x1b[4~", Key{Type: KeyEnd}},
		{"pgup", "\x1b[5~", Key{Type: KeyPgUp}},
		{"pgdown", "\x1b[6~", Key{Type: KeyPgDown}},
		{"f1", "\x1bOP", Key{Type: KeyF1}},
		{"f2", "\x1bOQ", Key{Type: KeyF2}},
		{"f3", "\x1bOR", Key{Type: KeyF3}},
		{"f4", "\x1bOS", Key{Type: KeyF4}},
		{"alt+a", "\x1ba", Key{Type: KeyRunes, Text: "a", Code: 'a', Mod: ModAlt}},
		{"ctrl+up (modifier CSI)", "\x1b[1;5A", Key{Type: KeyUp, Mod: ModCtrl}},
		{"ctrl+delete (modifier CSI)", "\x1b[3;5~", Key{Type: KeyDelete, Mod: ModCtrl}},
		{"shift+right (modifier CSI)", "\x1b[1;2C", Key{Type: KeyRight, Mod: ModShift}},
		{"ctrl+alt+left (modifier CSI)", "\x1b[1;7D", Key{Type: KeyLeft, Mod: ModCtrl | ModAlt}},
		{"unknown CSI final byte", "\x1b[W", Key{Type: KeyUnknown}},
		{"back-tab is shift+tab, not unknown", "\x1b[Z", Key{Type: KeyTab, Mod: ModShift}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readAllKeys(t, tt.input, 1)[0]
			if got.Type != tt.want.Type || got.Mod != tt.want.Mod || got.Text != tt.want.Text {
				t.Errorf("ReadEvent(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// TestModifierSequenceDoesNotLeakBytes is a regression test for the bug
// where a modifier-prefixed CSI sequence (e.g. Ctrl+Up = ESC[1;5A) left
// trailing bytes unconsumed, causing them to be misread as literal
// keypresses on subsequent ReadKey calls.
// TestReadKeyKittyProtocol proves task #15's acceptance criterion: kitty
// keyboard protocol (CSI u) sequences correctly decode Shift+Enter,
// Ctrl+Shift+letter, and other modifier combinations the legacy xterm
// CSI-modifier encoding cannot represent, while existing legacy-format
// sequences (already covered by TestReadKeySingle) continue to parse
// exactly as before -- the format's final byte ('u') never collides with
// any legacy final byte this reader already handles.
func TestReadKeyKittyProtocol(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Key
	}{
		{"plain 'a', no modifier group", "\x1b[97u", Key{Type: KeyRunes, Text: "a", Code: 'a'}},
		{"shift+enter", "\x1b[13;2u", Key{Type: KeyEnter, Mod: ModShift}},
		{"ctrl+shift+a (the legacy-unrepresentable case)", "\x1b[97;6u", Key{Type: KeyRunes, Text: "a", Code: 'a', Mod: ModCtrl | ModShift}},
		{"ctrl+a alone still maps to legacy KeyCtrl for compatibility", "\x1b[97;5u", Key{Type: KeyCtrl, Code: 'a'}},
		{"ctrl+A (uppercase codepoint) alone still maps to legacy KeyCtrl", "\x1b[65;5u", Key{Type: KeyCtrl, Code: 'a'}},
		{"ctrl+c alone still maps to legacy KeyCtrlC for compatibility", "\x1b[99;5u", Key{Type: KeyCtrlC}},
		{"ctrl+shift+c is NOT collapsed into KeyCtrlC", "\x1b[99;6u", Key{Type: KeyRunes, Text: "c", Code: 'c', Mod: ModCtrl | ModShift}},
		{"alt+tab", "\x1b[9;3u", Key{Type: KeyTab, Mod: ModAlt}},
		{"ctrl+backspace", "\x1b[127;5u", Key{Type: KeyBackspace, Mod: ModCtrl}},
		{"shift+esc", "\x1b[27;2u", Key{Type: KeyEsc, Mod: ModShift}},
		{"super+space (super-only, no legacy encoding exists at all)", "\x1b[32;9u", Key{Type: KeySpace, Text: " ", Code: ' ', Mod: ModSuper}},
		{"digit with shift", "\x1b[53;2u", Key{Type: KeyRunes, Text: "5", Code: '5', Mod: ModShift}},
		{"sub-params (shifted-key alternate) are consumed, not leaked", "\x1b[97:65;6u", Key{Type: KeyRunes, Text: "a", Code: 'a', Mod: ModCtrl | ModShift}},
		{"event-type=3 (release) yields Unknown, not a phantom press", "\x1b[97;5:3u", Key{Type: KeyUnknown}},
		{"event-type=2 (repeat) is treated like a press", "\x1b[97;5:2u", Key{Type: KeyCtrl, Code: 'a'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := readAllKeys(t, tt.input, 1)[0]
			if got.Type != tt.want.Type || got.Mod != tt.want.Mod || got.Text != tt.want.Text {
				t.Errorf("ReadEvent(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// TestKittySequenceFollowedByAnotherKeyDoesNotLeakBytes proves a CSI-u
// sequence's parameters (including ':' sub-values) are fully consumed —
// the same leaked-bytes risk the legacy numeric-CSI path already guards
// against (see TestModifierSequenceDoesNotLeakBytes) — by checking a
// second, ordinary key typed right after still decodes cleanly.
func TestKittySequenceFollowedByAnotherKeyDoesNotLeakBytes(t *testing.T) {
	keys := readAllKeys(t, "\x1b[97:65;6ub", 2)
	if keys[0].Type != KeyRunes || keys[0].Text != "a" || keys[0].Mod != ModCtrl|ModShift {
		t.Errorf("first key = %+v, want ctrl+shift+a", keys[0])
	}
	if keys[1].Type != KeyRunes || keys[1].Text != "b" || keys[1].Mod != ModNone {
		t.Errorf("second key = %+v, want plain 'b' (leaked bytes from the kitty sequence would corrupt this)", keys[1])
	}
}

func TestModifierSequenceDoesNotLeakBytes(t *testing.T) {
	// Ctrl+Up followed by a plain 'x' - if the sequence leaks bytes, the
	// second ReadEvent call will return '5' or 'A' instead of 'x'.
	keys := readAllKeys(t, "\x1b[1;5Ax", 2)

	if keys[0].Type != KeyUp {
		t.Fatalf("first key = %+v, want KeyUp", keys[0])
	}
	want := Key{Type: KeyRunes, Text: "x", Code: 'x'}
	if keys[1].Type != want.Type || keys[1].Text != want.Text {
		t.Fatalf("second key = %+v, want %+v (modifier sequence leaked bytes)", keys[1], want)
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		key  Key
		want string
	}{
		{Key{Type: KeyRunes, Text: "a", Code: 'a'}, "a"},
		{Key{Type: KeyRunes, Text: "a", Code: 'a', Mod: ModAlt}, "alt+a"},
		{Key{Type: KeyCtrl, Code: 'x'}, "ctrl+x"},
		{Key{Type: KeyCtrlC}, "ctrl+c"},
		{Key{Type: KeyUp}, "up"},
		{Key{Type: KeyUp, Mod: ModCtrl}, "ctrl+up"},
		{Key{Type: KeyRight, Mod: ModCtrl | ModAlt | ModShift}, "ctrl+alt+shift+right"},
		{Key{Type: KeyDown}, "down"},
		{Key{Type: KeyLeft}, "left"},
		{Key{Type: KeyRight}, "right"},
		{Key{Type: KeyEnter}, "enter"},
		{Key{Type: KeyEsc}, "esc"},
		{Key{Type: KeyEsc, Mod: ModAlt}, "alt+esc"},
		{Key{Type: KeyTab}, "tab"},
		{Key{Type: KeyBackspace}, "backspace"},
		{Key{Type: KeyDelete}, "delete"},
		{Key{Type: KeySpace}, "space"},
		{Key{Type: KeyHome}, "home"},
		{Key{Type: KeyEnd}, "end"},
		{Key{Type: KeyPgUp}, "pgup"},
		{Key{Type: KeyPgDown}, "pgdown"},
		{Key{Type: KeyF1}, "f1"},
		{Key{Type: KeyF2}, "f2"},
		{Key{Type: KeyF3}, "f3"},
		{Key{Type: KeyF4}, "f4"},
		{Key{Type: KeyUnknown}, "unknown"},
	}
	for _, tt := range tests {
		if got := tt.key.String(); got != tt.want {
			t.Errorf("Key{%+v}.String() = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestReadEventMouse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  MouseEvent
	}{
		{"left press", "\x1b[<0;10;20M", MouseEvent{X: 9, Y: 19, Button: MouseButtonLeft, Action: MouseActionPress}},
		{"left release", "\x1b[<0;10;20m", MouseEvent{X: 9, Y: 19, Button: MouseButtonLeft, Action: MouseActionRelease}},
		{"right press", "\x1b[<2;1;1M", MouseEvent{X: 0, Y: 0, Button: MouseButtonRight, Action: MouseActionPress}},
		{"drag with left button held", "\x1b[<32;5;5M", MouseEvent{X: 4, Y: 4, Button: MouseButtonLeft, Action: MouseActionMotion}},
		{"wheel up", "\x1b[<64;1;1M", MouseEvent{X: 0, Y: 0, Button: MouseButtonWheelUp, Action: MouseActionPress}},
		{"wheel down", "\x1b[<65;1;1M", MouseEvent{X: 0, Y: 0, Button: MouseButtonWheelDown, Action: MouseActionPress}},
		{"ctrl+shift+left press", "\x1b[<20;3;4M", MouseEvent{X: 2, Y: 3, Button: MouseButtonLeft, Action: MouseActionPress, Mod: ModShift | ModCtrl}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := readAll(t, tt.input, 1)
			got, ok := events[0].(MouseEvent)
			if !ok {
				t.Fatalf("event = %#v (%T), want MouseEvent", events[0], events[0])
			}
			if got != tt.want {
				t.Errorf("ReadEvent(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// TestMalformedMouseSequenceDoesNotLeakBytes is the same class of
// regression as TestModifierSequenceDoesNotLeakBytes, for the mouse parser:
// an unrecognized SGR mouse sequence must still consume every byte it
// looked at.
func TestMalformedMouseSequenceDoesNotLeakBytes(t *testing.T) {
	events := readAll(t, "\x1b[<0;10Zx", 2)

	if k, ok := events[0].(Key); !ok || k.Type != KeyUnknown {
		t.Fatalf("first event = %#v, want Key{Type: KeyUnknown}", events[0])
	}
	want := Key{Type: KeyRunes, Text: "x", Code: 'x'}
	if k, ok := events[1].(Key); !ok || k.Type != want.Type || k.Text != want.Text {
		t.Fatalf("second event = %#v, want %+v (malformed mouse sequence leaked bytes)", events[1], want)
	}
}

func TestReadEventBracketedPaste(t *testing.T) {
	events := readAll(t, "\x1b[200~hello, world\x1b[201~", 1)
	got, ok := events[0].(PasteEvent)
	if !ok {
		t.Fatalf("event = %#v (%T), want PasteEvent", events[0], events[0])
	}
	if got.Text != "hello, world" {
		t.Errorf("PasteEvent.Text = %q, want %q", got.Text, "hello, world")
	}
}

func TestReadEventPasteThenKey(t *testing.T) {
	// A key typed right after a paste ends must not be swallowed or
	// corrupted by the paste end-marker scan.
	events := readAll(t, "\x1b[200~hi\x1b[201~q", 2)

	p, ok := events[0].(PasteEvent)
	if !ok || p.Text != "hi" {
		t.Fatalf("first event = %#v, want PasteEvent{Text: %q}", events[0], "hi")
	}
	k, ok := events[1].(Key)
	if !ok || k.Type != KeyRunes || k.Text != "q" {
		t.Fatalf("second event = %#v, want Key 'q'", events[1])
	}
}

func TestReadEventPasteWithUnicodeAndEmbeddedNewline(t *testing.T) {
	events := readAll(t, "\x1b[200~héllo\nwörld\x1b[201~", 1)
	got, ok := events[0].(PasteEvent)
	if !ok {
		t.Fatalf("event = %#v, want PasteEvent", events[0])
	}
	want := "héllo\nwörld"
	if got.Text != want {
		t.Errorf("PasteEvent.Text = %q, want %q", got.Text, want)
	}
}

func TestReadEventPasteTruncatedByEOF(t *testing.T) {
	rd := NewReader(strings.NewReader("\x1b[200~partial"))
	ev, err := rd.ReadEvent()
	if err == nil {
		t.Fatal("expected an error for a paste with no end marker before EOF")
	}
	p, ok := ev.(PasteEvent)
	if !ok || p.Text != "partial" {
		t.Fatalf("event = %#v, want PasteEvent{Text: %q} alongside the EOF error", ev, "partial")
	}
}

func TestShiftTabDecodesFromCSIZ(t *testing.T) {
	got := readAllKeys(t, "\x1b[Za\x1b[1;2Zb", 4)
	want := []string{"shift+tab", "a", "shift+tab", "b"}
	for i, k := range got {
		if k.String() != want[i] {
			t.Errorf("key %d = %q, want %q (%#v)", i, k.String(), want[i], k)
		}
	}
	if got[0].Type != KeyTab || !got[0].Mod.Shift() {
		t.Errorf("ESC[Z = %#v, want KeyTab with ModShift", got[0])
	}
}

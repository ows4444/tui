package input

import (
	"reflect"
	"testing"
)

// decodeFix is one regression case from the S02 decode fixes (audit rows C1,
// C2, C3, C6, C9). The same table seeds FuzzReadEvent, so every sequence is
// both asserted exactly here and fed to the fuzz target.
type decodeFix struct {
	name string
	in   string
	want []Event
}

var decodeFixes = []decodeFix{
	// C1: horizontal wheel.
	{"C1 wheel left", "\x1b[<66;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonWheelLeft, Action: MouseActionPress}}},
	{"C1 wheel right", "\x1b[<67;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonWheelRight, Action: MouseActionPress}}},
	{"C1 wheel up unchanged", "\x1b[<64;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonWheelUp, Action: MouseActionPress}}},
	{"C1 wheel down unchanged", "\x1b[<65;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonWheelDown, Action: MouseActionPress}}},
	{"C1 wheel left with shift", "\x1b[<70;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonWheelLeft, Action: MouseActionPress, Mod: ModShift}}},
	// C2: buttons 8-11.
	{"C2 back", "\x1b[<128;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonBack, Action: MouseActionPress}}},
	{"C2 forward", "\x1b[<129;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonForward, Action: MouseActionPress}}},
	{"C2 button 10", "\x1b[<130;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButton10, Action: MouseActionPress}}},
	{"C2 button 11", "\x1b[<131;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButton11, Action: MouseActionPress}}},
	{"C2 back release", "\x1b[<128;10;5m", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonBack, Action: MouseActionRelease}}},
	{"C2 back drag", "\x1b[<160;10;5M", []Event{MouseEvent{X: 9, Y: 4, Button: MouseButtonBack, Action: MouseActionMotion}}},
	// C3: ESC + control byte.
	{"C3 alt+ctrl+a", "\x1b\x01", []Event{Key{Type: KeyCtrl, Code: 'a', Mod: ModAlt}}},
	{"C3 alt+ctrl+z", "\x1b\x1a", []Event{Key{Type: KeyCtrl, Code: 'z', Mod: ModAlt}}},
	{"C3 alt+ctrl+c", "\x1b\x03", []Event{Key{Type: KeyCtrlC, Mod: ModAlt}}},
	{"C3 alt+ctrl+space", "\x1b\x00", []Event{Key{Type: KeyCtrl, Code: ' ', Mod: ModAlt}}},
	{"C3 alt+enter unchanged", "\x1b\r", []Event{Key{Type: KeyEnter, Mod: ModAlt}}},
	// C6: kitty associated text.
	{"C6 shift+a text", "\x1b[97;2;65u", []Event{Key{Type: KeyRunes, Text: "A", Code: 'A'}}},
	{"C6 no text keeps mod", "\x1b[97;2u", []Event{Key{Type: KeyRunes, Text: "a", Code: 'a', Mod: ModShift}}},
	{"C6 multi-codepoint text", "\x1b[97;1;104:105u", []Event{Key{Type: KeyRunes, Text: "hi", Code: 'h'}}},
	{"C6 alt keeps alt", "\x1b[97;4;65u", []Event{Key{Type: KeyRunes, Text: "A", Code: 'A', Mod: ModAlt}}},
	// C9: nF sequences are consumed.
	{"C9 charset G0", "\x1b(Bx", []Event{Key{Type: KeyUnknown}, Key{Type: KeyRunes, Text: "x", Code: 'x'}}},
	{"C9 charset G1", "\x1b)0x", []Event{Key{Type: KeyUnknown}, Key{Type: KeyRunes, Text: "x", Code: 'x'}}},
	{"C9 two intermediates", "\x1b(!Bx", []Event{Key{Type: KeyUnknown}, Key{Type: KeyRunes, Text: "x", Code: 'x'}}},
	{"C9 lone alt+paren", "\x1b(", []Event{Key{Type: KeyRunes, Text: "(", Code: '(', Mod: ModAlt}}},
}

func TestDecodeFixesRegressionTable(t *testing.T) {
	for _, c := range decodeFixes {
		t.Run(c.name, func(t *testing.T) {
			got := readAll(t, c.in, len(c.want))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("%q decoded to %#v, want %#v", c.in, got, c.want)
			}
		})
	}
}

func TestNoKeyRunesEventFromEscCharset(t *testing.T) {
	for _, e := range readAll(t, "\x1b(Bx", 2)[:1] {
		if k, ok := e.(Key); ok && k.Type == KeyRunes {
			t.Fatalf("charset sequence produced %#v", k)
		}
	}
}

package input

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func decodeOne(t *testing.T, in string) Event {
	t.Helper()
	ev, err := NewReader(strings.NewReader(in)).ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	return ev
}

// Criterion #67: keypad Enter under kitty flag 8 is KeyEnter with Keypad.
func TestKittyKeypadEnter(t *testing.T) {
	got := decodeOne(t, "\x1b[57414u")
	if want := (Key{Type: KeyEnter, Keypad: true}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if k := decodeOne(t, "\x1b[13u").(Key); k.Keypad {
		t.Fatal("main Enter must not be Keypad")
	}
	got = decodeOne(t, "\x1b[57414;5u")
	if want := (Key{Type: KeyEnter, Mod: ModCtrl, Keypad: true}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestKittyKeypadKeys(t *testing.T) {
	for i := 0; i < 10; i++ {
		got := decodeOne(t, "\x1b["+strconv.Itoa(57399+i)+"u")
		want := Key{Type: KeyRunes, Text: string(rune('0' + i)), Code: rune('0' + i), Keypad: true}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("KP_%d: got %#v", i, got)
		}
	}
	got := decodeOne(t, "\x1b[57413u")
	if want := (Key{Type: KeyRunes, Text: "+", Code: '+', Keypad: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("KP_ADD: got %#v", got)
	}
	got = decodeOne(t, "\x1b[57419u")
	if want := (Key{Type: KeyUp, Keypad: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("KP_UP: got %#v", got)
	}
	if k := decodeOne(t, "\x1b[57427u").(Key); k.Type != KeyUnknown {
		t.Errorf("KP_BEGIN: got %#v", k)
	}
}

func TestKittyMediaKeys(t *testing.T) {
	cases := map[string]KeyType{
		"\x1b[57428u": KeyMediaPlay, "\x1b[57430u": KeyMediaPlayPause,
		"\x1b[57435u": KeyMediaTrackNext, "\x1b[57439u": KeyMediaVolumeUp,
		"\x1b[57440u": KeyMediaMute,
	}
	for in, want := range cases {
		got := decodeOne(t, in).(Key)
		if got.Type != want || got.Keypad || got.String() == "unknown" {
			t.Errorf("%q: got %#v (%s)", in, got, got.String())
		}
	}
}

package input

import "testing"

// Task 42: ESC ESC sequences, SS3 modifiers, rxvt $/^/@ keys.

func TestEscEscArrowIsAltUp(t *testing.T) {
	ks := readAllKeys(t, "\x1b\x1b[A", 1)
	if ks[0].Type != KeyUp || ks[0].Mod != ModAlt {
		t.Fatalf("got %#v, want Up+Alt", ks[0])
	}
	ks = readAllKeys(t, "\x1b\x1b[1;5Ax", 2)
	if ks[0].Type != KeyUp || ks[0].Mod != ModAlt|ModCtrl || ks[1].Type != KeyRunes {
		t.Fatalf("got %#v", ks)
	}
	ks = readAllKeys(t, "\x1b\x1bOP", 1)
	if ks[0].Type != KeyF1 || ks[0].Mod != ModAlt {
		t.Fatalf("got %#v", ks[0])
	}
}

func TestSS3Modifier(t *testing.T) {
	for _, in := range []string{"\x1bO5P", "\x1bO1;5P"} {
		ks := readAllKeys(t, in, 1)
		if ks[0].Type != KeyF1 || ks[0].Mod != ModCtrl {
			t.Errorf("%q = %#v, want F1+Ctrl", in, ks[0])
		}
	}
	if k := readAllKeys(t, "\x1bOP", 1)[0]; k.Type != KeyF1 || k.Mod != ModNone {
		t.Errorf("plain SS3 = %#v", k)
	}
}

func TestRxvtModifiedKeys(t *testing.T) {
	cases := []struct {
		in  string
		typ KeyType
		mod Mod
	}{
		{"\x1b[23$", KeyF11, ModShift},
		{"\x1b[23^", KeyF11, ModCtrl},
		{"\x1b[23@", KeyF11, ModCtrl | ModShift},
		{"\x1b[3^", KeyDelete, ModCtrl},
		{"\x1b[2$", KeyInsert, ModShift},
	}
	for _, c := range cases {
		k := readAllKeys(t, c.in+"x", 2)
		if k[0].Type != c.typ || k[0].Mod != c.mod {
			t.Errorf("%q = %#v", c.in, k[0])
		}
		if k[1].Type != KeyRunes || k[1].Text != "x" {
			t.Errorf("%q swallowed following key: %#v", c.in, k[1])
		}
	}
}

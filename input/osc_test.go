package input

import "testing"

func TestBackgroundColorReply(t *testing.T) {
	cases := map[string]BackgroundColorEvent{
		"\x1b]11;rgb:1e1e/1e1e/1e1e\x07":   {R: 30, G: 30, B: 30},
		"\x1b]11;rgb:ffff/ffff/ffff\x1b\\": {R: 255, G: 255, B: 255},
		"\x1b]11;rgb:00/80/ff\x07":         {R: 0, G: 128, B: 255},
		"\x1b]11;rgb:f/0/8\x07":            {R: 255, G: 0, B: 136},
	}
	for in, want := range cases {
		got := readAll(t, in+"z", 2)
		if got[0] != want {
			t.Errorf("%q = %#v, want %#v", in, got[0], want)
		}
		if k, ok := got[1].(Key); !ok || k.String() != "z" {
			t.Errorf("%q leaked/lost trailing key: %#v", in, got[1])
		}
	}
}

func TestMalformedOrOtherOSCIsConsumed(t *testing.T) {
	for _, in := range []string{
		"\x1b]11;rgb:zz/00/00\x07", "\x1b]11;rgb:00/00\x07", "\x1b]11;rgb:12345/0/0\x07",
		"\x1b]10;rgb:00/00/00\x07", "\x1b]52;c;abc\x1b\\", "\x1b]11;hello\x07",
	} {
		got := readAll(t, in+"z", 2)
		if _, isBG := got[0].(BackgroundColorEvent); isBG {
			t.Errorf("%q produced a BackgroundColorEvent", in)
		}
		if k, ok := got[1].(Key); !ok || k.String() != "z" {
			t.Errorf("%q did not consume whole sequence; next = %#v", in, got[1])
		}
	}
}

func TestAltCloseBracketStillAltKey(t *testing.T) {
	got := readAll(t, "\x1b]x", 2)
	k, ok := got[0].(Key)
	if !ok || k.Type != KeyRunes || k.Code != ']' || !k.Mod.Alt() {
		t.Errorf("ESC ] = %#v, want Alt+]", got[0])
	}
}

func TestPaletteReply(t *testing.T) {
	got := readAll(t, "\x1b]4;1;rgb:cccc/0000/1111\x07\x1b]4;15;rgb:ff/ff/ff\x1b\\z", 3)
	if got[0] != (PaletteColorEvent{Index: 1, R: 204, G: 0, B: 17}) {
		t.Errorf("got %#v", got[0])
	}
	if got[1] != (PaletteColorEvent{Index: 15, R: 255, G: 255, B: 255}) {
		t.Errorf("got %#v", got[1])
	}
	for _, in := range []string{"\x1b]4;300;rgb:00/00/00\x07", "\x1b]4;x;rgb:00/00/00\x07", "\x1b]4;1;bad\x07"} {
		g := readAll(t, in+"z", 2)
		if _, ok := g[0].(PaletteColorEvent); ok {
			t.Errorf("%q decoded", in)
		}
	}
}

package input

import (
	"strings"
	"testing"
)

// Criterion #78: a printable key sets Text and Code without allocating.
func TestTextOfNoAllocASCII(t *testing.T) {
	if n := testing.AllocsPerRun(100, func() { _ = runeKey('a', ModNone) }); n != 0 {
		t.Fatalf("runeKey ASCII allocs = %v, want 0", n)
	}
	for _, r := range []rune{'a', ' ', '~', 0x7f, 'é', '世'} {
		k := runeKey(r, ModAlt)
		if k.Type != KeyRunes || k.Text != string(r) || k.Code != r || k.Mod != ModAlt {
			t.Fatalf("runeKey(%q) = %+v", r, k)
		}
	}
}

func TestReadEventPrintableDecodes(t *testing.T) {
	rd := NewReader(strings.NewReader("a é"))
	for _, want := range []rune{'a', ' ', 'é'} {
		ev, err := rd.ReadEvent()
		if err != nil {
			t.Fatal(err)
		}
		k := ev.(Key)
		if want == ' ' {
			if k.Type != KeySpace || k.Text != " " || k.Code != ' ' {
				t.Fatalf("space decoded as %+v", k)
			}
			continue
		}
		if k.Type != KeyRunes || k.Text != string(want) || k.Code != want {
			t.Fatalf("got %+v want %q", k, want)
		}
	}
}

// Criterion #79: text a terminal reports for one key, a grapheme cluster, is
// kept whole in Text, with Code its first rune.
func TestKeyTextHoldsAWholeCluster(t *testing.T) {
	// kitty flag 16: CSI 101 ; 1 ; 101:769 u is the key "e" producing "e" + U+0301.
	rd := NewReader(strings.NewReader("\x1b[101;1;101:769u"))
	ev, err := rd.ReadEvent()
	if err != nil {
		t.Fatal(err)
	}
	k := ev.(Key)
	if k.Type != KeyRunes || k.Text != "é" || k.Code != 'e' {
		t.Errorf("got %+v, want Text %q and Code 'e'", k, "é")
	}
	if k.String() != "é" {
		t.Errorf("String() = %q", k.String())
	}
}

func TestKeyIsComparable(t *testing.T) {
	a, b := runeKey('x', ModNone), runeKey('x', ModNone)
	if a != b {
		t.Error("equal keys must compare equal now that Key has no slice field")
	}
}

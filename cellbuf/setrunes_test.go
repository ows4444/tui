package cellbuf

import "testing"

func TestSetRunesMatchesSetString(t *testing.T) {
	for _, s := range []string{"", "hello, world", "café 世界", "tab\there", "x"} {
		a, b := New(10, 1), New(10, 1)
		na := a.SetString(1, 0, s, 0)
		nb := b.SetRunes(1, 0, []rune(s), 0)
		if na != nb || a.String() != b.String() {
			t.Errorf("%q: SetString %d %q, SetRunes %d %q", s, na, a.String(), nb, b.String())
		}
	}
}

func TestSetStringFillSubNoAllocs(t *testing.T) {
	b := New(80, 24)
	rs := []rune("the quick brown fox jumps over the lazy dog")
	id := b.StyleID(Style{Attrs: 1})
	if n := testing.AllocsPerRun(50, func() {
		v := b.Sub(Rect{0, 1, 60, 10})
		v.SetString(0, 0, "the quick brown fox", id)
		v.SetRunes(0, 1, rs, id)
		v.Fill(Rect{0, 2, 60, 3}, " ", 0)
	}); n != 0 {
		t.Errorf("allocs = %v, want 0", n)
	}
}

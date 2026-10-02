package ansi

import (
	"math/rand"
	"strings"
	"testing"
)

// TestTruncateRTLProperty: on random Hebrew/Arabic mixed with ASCII, Truncate
// keeps the logical text (output is a prefix of the input), fits the width,
// and Width equals Width of the stripped text.
func TestTruncateRTLProperty(t *testing.T) {
	pools := [][]rune{
		[]rune("אבגדהוזחטיכלמנסעפצקרשת"),
		[]rune("ابتثجحخدذرزسشصضطظعغفقكلمنهوي"),
		[]rune("abcXYZ019 .,-_"),
		{'ְ', 'ً', '‏', '‎'}, // marks and bidi controls
	}
	rng := rand.New(rand.NewSource(101))
	for iter := 0; iter < 3000; iter++ {
		var b strings.Builder
		for n := rng.Intn(30); n > 0; n-- {
			p := pools[rng.Intn(len(pools))]
			b.WriteRune(p[rng.Intn(len(p))])
		}
		in := b.String()
		if Width(in) != Width(StripANSI(in)) {
			t.Fatalf("Width(%q)=%d != Width(stripped)=%d", in, Width(in), Width(StripANSI(in)))
		}
		for w := 0; w <= Width(in)+2; w++ {
			out := Truncate(in, w)
			if !strings.HasPrefix(in, out) {
				t.Fatalf("Truncate(%q,%d)=%q is not a prefix", in, w, out)
			}
			if Width(out) > w {
				t.Fatalf("Truncate(%q,%d)=%q width %d exceeds", in, w, out, Width(out))
			}
			if Width(out) != Width(StripANSI(out)) {
				t.Fatalf("Width(%q) != Width(stripped)", out)
			}
		}
	}
}

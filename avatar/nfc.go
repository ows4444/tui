package avatar

import "sort"

// Unicode Normalization Form C (UAX #15): decompose every rune canonically,
// put combining marks in canonical order, then compose again. It makes the
// spellings of a name that a reader cannot tell apart, "e" followed by a
// combining acute and the single rune U+00E9, hash to one avatar. The
// standard library has no normalizer, so this is one, over generated tables
// (nfc_tables.go).

// Hangul syllables are composed and decomposed by arithmetic (the Unicode
// standard, section 3.12): a syllable is a leading consonant, a vowel and an
// optional trailing consonant.
const (
	hangulS, hangulL, hangulV, hangulT = 0xAC00, 0x1100, 0x1161, 0x11A7
	hangulLs, hangulVs, hangulTs       = 19, 21, 28
	hangulSs                           = hangulLs * hangulVs * hangulTs
)

// nfc returns s in Normalization Form C.
func nfc(s string) string {
	// No rune below U+0300 has a combining class or a decomposition that NFC
	// would not undo, so text made only of them is already normalized. That
	// is nearly every name.
	plain := true
	for _, r := range s {
		if r >= 0x300 {
			plain = false
			break
		}
	}
	if plain {
		return s
	}
	var d []rune
	for _, r := range s {
		d = decompose(d, r)
	}
	reorder(d)
	return string(compose(d))
}

// class returns the canonical combining class of r; 0 for a starter.
func class(r rune) uint8 {
	i := sort.Search(len(nfcClasses), func(i int) bool { return nfcClasses[i].hi >= r })
	if i < len(nfcClasses) && nfcClasses[i].lo <= r {
		return nfcClasses[i].ccc
	}
	return 0
}

// decompose appends the full canonical decomposition of r to d.
func decompose(d []rune, r rune) []rune {
	if s := r - hangulS; s >= 0 && s < hangulSs {
		d = append(d, hangulL+s/(hangulVs*hangulTs), hangulV+s%(hangulVs*hangulTs)/hangulTs)
		if t := s % hangulTs; t != 0 {
			d = append(d, hangulT+t)
		}
		return d
	}
	i := sort.Search(len(nfcDecomps), func(i int) bool { return nfcDecomps[i][0] >= r })
	if i == len(nfcDecomps) || nfcDecomps[i][0] != r {
		return append(d, r)
	}
	d = decompose(d, nfcDecomps[i][1])
	if b := nfcDecomps[i][2]; b != 0 {
		d = decompose(d, b)
	}
	return d
}

// reorder puts each run of combining marks in order of combining class,
// keeping marks of one class in the order they came (a stable sort: the runs
// are short).
func reorder(d []rune) {
	for i := 1; i < len(d); i++ {
		c := class(d[i])
		if c == 0 {
			continue
		}
		for j := i; j > 0 && class(d[j-1]) > c; j-- {
			d[j], d[j-1] = d[j-1], d[j]
		}
	}
}

// pair returns the rune a and b compose to, if they do.
func pair(a, b rune) (rune, bool) {
	if l := a - hangulL; l >= 0 && l < hangulLs {
		if v := b - hangulV; v >= 0 && v < hangulVs {
			return hangulS + (l*hangulVs+v)*hangulTs, true
		}
	}
	if s := a - hangulS; s >= 0 && s < hangulSs && s%hangulTs == 0 {
		if t := b - hangulT; t > 0 && t < hangulTs {
			return a + t, true
		}
	}
	i := sort.Search(len(nfcComposites), func(i int) bool {
		c := nfcComposites[i]
		return c[0] > a || c[0] == a && c[1] >= b
	})
	if i < len(nfcComposites) && nfcComposites[i][0] == a && nfcComposites[i][1] == b {
		return nfcComposites[i][2], true
	}
	return 0, false
}

// compose combines each starter with the marks after it that it composes
// with, in place. A mark is blocked from its starter when a rune between
// them is a starter or has a combining class at least as high.
func compose(d []rune) []rune {
	out := d[:0]
	starter := -1 // index in out of the last starter
	var last uint8
	for _, r := range d {
		c := class(r)
		if starter >= 0 && (len(out)-1 == starter || last < c) {
			if p, ok := pair(out[starter], r); ok {
				out[starter] = p
				continue
			}
		}
		out = append(out, r)
		if c == 0 {
			starter = len(out) - 1
		}
		last = c
	}
	return out
}

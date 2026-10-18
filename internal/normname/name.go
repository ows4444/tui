// Package normname puts a name in the one form it is hashed in: Unicode
// Normalization Form C, trimmed, lowercased, and cut to a fixed length. It is
// what makes "Alain", " alain " and the two spellings of an accented letter one
// name. Everything it needs is in tables generated from one version of the
// Unicode Character Database (UnicodeVersion), not read from the standard
// library's, so a new Go release cannot change what a name becomes.
package normname

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxName is the most bytes of a name that count; the rest is dropped after
// normalizing, at a rune boundary. Hashing must not cost more for a long name
// than for a short one.
const MaxName = 256

// maxInput is how much of a name is even normalized: four times what counts,
// so a combining sequence at the end of the kept part is whole.
const maxInput = 4 * MaxName

// NFC returns s in Normalization Form C.
func NFC(s string) string { return nfc(s) }

// Name returns s as it is hashed: at most maxInput bytes of it put in NFC,
// trimmed, lowercased, and cut to MaxName bytes.
func Name(s string) string {
	s = cut(s, maxInput)
	return cut(lower(strings.TrimFunc(nfc(s), trimmed)), MaxName)
}

// Cap returns the first MaxName bytes of s, or fewer so as not to split a rune.
// It is Name without the normalizing, for a name that is hashed as written.
func Cap(s string) string { return cut(s, MaxName) }

// cut returns the first n bytes of s, or fewer so as not to split a rune.
func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// inRuns reports whether r is in one of the sorted runs.
func inRuns(runs [][2]rune, r rune) bool {
	i := sort.Search(len(runs), func(i int) bool { return runs[i][1] >= r })
	return i < len(runs) && runs[i][0] <= r
}

// trimmed reports whether r is trimmed from the ends of a name: the ASCII
// whitespace, the line and paragraph separators, the byte order mark and every
// space separator (Zs). It differs from unicode.IsSpace on U+0085 (not
// trimmed) and U+FEFF (trimmed).
func trimmed(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', 0xFEFF, 0x2028, 0x2029:
		return true
	}
	return inRuns(spaceRuns[:], r)
}

// lowerRune is the simple lowercase mapping of r.
func lowerRune(r rune) rune {
	i := sort.Search(len(lowerMap), func(i int) bool { return lowerMap[i][0] >= r })
	if i < len(lowerMap) && lowerMap[i][0] == r {
		return lowerMap[i][1]
	}
	return r
}

// lower is the Unicode default lowercasing: the simple lowercase mapping of
// each rune, plus the two rules of SpecialCasing.txt that are not locale bound.
// U+0130 becomes "i" and a combining dot, and a capital sigma at the end of a
// word becomes a final sigma.
func lower(s string) string {
	rs := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range rs {
		switch {
		case r == 0x130:
			b.WriteRune('i')
			b.WriteRune(0x307)
		case r == 0x3A3 && finalSigma(rs, i):
			b.WriteRune(0x3C2)
		default:
			b.WriteRune(lowerRune(r))
		}
	}
	return b.String()
}

// finalSigma is the Final_Sigma condition of the Unicode standard (3.13): the
// sigma follows a cased letter and no cased letter follows it, where
// case-ignorable runes on either side are skipped.
func finalSigma(rs []rune, i int) bool {
	before := false
	for j := i - 1; j >= 0; j-- {
		if inRuns(ignorableRuns[:], rs[j]) {
			continue
		}
		before = inRuns(casedRuns[:], rs[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(rs); j++ {
		if inRuns(ignorableRuns[:], rs[j]) {
			continue
		}
		return !inRuns(casedRuns[:], rs[j])
	}
	return true
}

package avatar

import (
	"math"
	"math/bits"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The seed is hashed once into a 32-bit state, and each trait continues from
// that state under its own string key. A trait therefore depends on the name
// and its key only, never on which other traits were read before it.

// sep separates the seed from a trait key; it is a byte UTF-8 never produces.
const sep = 0xff

// feed mixes one byte into the state.
func feed(h uint32, b byte) uint32 {
	return bits.RotateLeft32((h^uint32(b))*3432918353, 13)
}

func feedString(h uint32, s string) uint32 {
	for i := 0; i < len(s); i++ {
		h = feed(h, s[i])
	}
	return h
}

// finalize is the murmur3 fmix32 step: a bijection on uint32 with full
// avalanche, so two names one letter apart share nothing.
func finalize(h uint32) uint32 {
	h = (h ^ h>>16) * 2246822507
	h = (h ^ h>>13) * 3266489909
	return h ^ h>>16
}

// seedState hashes s. The length mixed in is counted in UTF-16 code units, so
// a rune outside the BMP counts twice.
func seedState(s string) uint32 {
	// Invalid UTF-8 is hashed as U+FFFD.
	s = strings.ToValidUTF8(s, string(utf8.RuneError))
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return feedString(1779033703^uint32(n), s) // #nosec G115 -- the length only seeds a hash; wrapping is harmless
}

// stream is the uniform float in [0, 1) for key under state.
func stream(state uint32, key string) float64 {
	return float64(finalize(feedString(feed(state, sep), key))) / 4294967296
}

// normalize trims and lowercases a name, so "Alain" and " alain " are one
// avatar. It does not apply Unicode NFC: the standard library has no
// normalizer, so a name spelled with combining marks is hashed as written.
func normalize(s string) string {
	return lower(strings.TrimFunc(s, trimmed))
}

// trimmed reports whether r is trimmed from the ends of a name: the ASCII
// whitespace, the no-break space, the byte order mark, the line and paragraph
// separators and every space separator (Zs). It differs from unicode.IsSpace
// on U+0085 (not trimmed) and U+FEFF (trimmed), and the vectors pin both.
func trimmed(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xA0, 0xFEFF, 0x2028, 0x2029:
		return true
	}
	return unicode.Is(unicode.Zs, r)
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
			b.WriteRune(unicode.ToLower(r))
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
		if caseIgnorable(rs[j]) {
			continue
		}
		before = cased(rs[j])
		break
	}
	if !before {
		return false
	}
	for j := i + 1; j < len(rs); j++ {
		if caseIgnorable(rs[j]) {
			continue
		}
		return !cased(rs[j])
	}
	return true
}

func cased(r rune) bool {
	return unicode.IsUpper(r) || unicode.IsLower(r) || unicode.IsTitle(r) ||
		unicode.Is(unicode.Other_Lowercase, r) || unicode.Is(unicode.Other_Uppercase, r)
}

func caseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', '^', '`', 0xA8, 0xAF, 0xB4, 0xB7, 0xB8, 0x2018, 0x2019, 0x2024, 0x2027,
		0xFE13, 0xFE52, 0xFE55, 0xFF07, 0xFF0E, 0xFF1A:
		return true
	}
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
}

// traits reads the values a name hashes to. Every value is addressed by a
// string key, so the keys are an append-only namespace.
type traits struct {
	state uint32
	// fixed pins a key to a position in [0, 1) in place of the hash. A
	// Model's Silhouette and Tone set it; nil pins nothing.
	fixed map[string]float64
}

func newTraits(name string, raw bool) traits {
	if !raw {
		name = normalize(name)
	}
	return traits{state: seedState(name)}
}

// at is the uniform float in [0, 1) for key.
func (t traits) at(key string) float64 {
	if v, ok := t.fixed[key]; ok {
		switch {
		case v > 0 && v < 1:
			return v
		case v >= 1:
			return 0.999999
		}
		return 0
	}
	return stream(t.state, key)
}

// num is the uniform float in [lo, hi) for key.
func (t traits) num(key string, lo, hi float64) float64 { return lo + t.at(key)*(hi-lo) }

// count is the uniform integer in [lo, hi] for key.
func (t traits) count(key string, lo, hi int) int {
	return lo + int(math.Floor(t.at(key)*float64(hi-lo+1)))
}

// jitter is the symmetric offset in [-amount, amount) for key.
func (t traits) jitter(key string, amount float64) float64 { return (t.at(key)*2 - 1) * amount }

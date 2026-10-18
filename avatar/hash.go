package avatar

import (
	"math"
	"math/bits"
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/internal/normname"
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

// normalize puts a name in the form it is hashed in (see package normname):
// Normalization Form C, trimmed, lowercased and cut to 256 bytes, so "Alain"
// and " alain " are one avatar, as are the two spellings of an accented letter.
func normalize(s string) string { return normname.Name(s) }

// traits reads the values a name hashes to. Every value is addressed by a
// string key, so the keys are an append-only namespace.
type traits struct {
	state uint32
	// fixed pins a key to a position in [0, 1) in place of the hash. A
	// Model's Silhouette, Tone and traitPins set it; nil pins nothing.
	fixed map[string]float64
}

func newTraits(name string, raw bool) traits {
	if raw {
		name = normname.Cap(name)
	} else {
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

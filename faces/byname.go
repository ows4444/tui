package faces

import (
	"hash/fnv"
	"math"
	"strings"

	"github.com/ows4444/tui/ansi"
)

// IndexFor returns the catalog index of the face that stands for name: a
// username, an id, any string. The same name always gets the same face, on
// every run and every machine, so a list can give each of its members a
// character of their own; pass it to Model.Set. Case and surrounding space
// are ignored, so "Ada" and " ada " share a face.
//
// The choice is spread evenly over the catalog. It depends on the catalog's
// size: a release that adds faces gives names different ones.
func IndexFor(name string) int {
	return int(nameHash(name, "") % uint32(len(catalog))) // #nosec G115 -- the catalog is far smaller than a uint32
}

// nameHash hashes name, ignoring case and surrounding space, with salt mixed
// in so that what one use of a name picks says nothing about another.
func nameHash(name, salt string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(name)))) // a hash.Hash never returns an error
	if salt != "" {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(salt))
	}
	// FNV alone leaves names that differ in one letter close together; the
	// murmur3 finalizer spreads them over the whole range.
	x := h.Sum32()
	x = (x ^ x>>16) * 2246822507
	x = (x ^ x>>13) * 3266489909
	return x ^ x>>16
}

// For returns the face that stands for name; see IndexFor.
func For(name string) Face { return catalog[IndexFor(name)] }

// ColorFor returns a colour that stands for name, the same one every time:
// a hue the name chose, at a saturation and lightness that read on a dark
// and on a light background. Set it as Model.Color, with or without For, to
// give a name a face in a colour of its own. The colour is chosen
// independently of the face, so two names that share a face rarely share
// both.
func ColorFor(name string) ansi.RGB {
	hue := float64(nameHash(name, "colour")%360) / 60
	const sat, light = 0.62, 0.6
	c := (1 - math.Abs(2*light-1)) * sat
	x := c * (1 - math.Abs(math.Mod(hue, 2)-1))
	var r, g, b float64
	switch int(hue) {
	case 0:
		r, g = c, x
	case 1:
		r, g = x, c
	case 2:
		g, b = c, x
	case 3:
		g, b = x, c
	case 4:
		r, b = x, c
	default:
		r, b = c, x
	}
	m := light - c/2
	to := func(v float64) uint8 { return uint8(math.Round((v + m) * 255)) } // #nosec G115 -- v+m is in [0, 1]
	return ansi.RGB{R: to(r), G: to(g), B: to(b)}
}

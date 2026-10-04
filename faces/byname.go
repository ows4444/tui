package faces

import (
	"hash/fnv"
	"strings"
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
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(name)))) // a hash.Hash never returns an error
	// FNV alone leaves names that differ in one letter close together; the
	// murmur3 finalizer spreads them over the whole range.
	x := h.Sum32()
	x = (x ^ x>>16) * 2246822507
	x = (x ^ x>>13) * 3266489909
	x ^= x >> 16
	return int(x % uint32(len(catalog))) // #nosec G115 -- the catalog is far smaller than a uint32
}

// For returns the face that stands for name; see IndexFor.
func For(name string) Face { return catalog[IndexFor(name)] }

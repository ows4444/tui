package ansi

import (
	"os"
	"sync/atomic"
)

// clusterWidthOn reports whether Width, Truncate and TrimLeftWidth treat an
// extended grapheme cluster as one unit. It is on unless SetClusterWidth(false)
// was called or, if it never was, the TUI_NO_CLUSTERS environment variable is
// set to a non-empty value the first time a width is measured. Nothing reads
// the environment at import time.
var clusterWidthOn clusterSetting

// clusterSetting is an atomic bool whose default is resolved from the
// environment on first use.
type clusterSetting struct {
	v atomic.Int32 // clustersUnset until decided
}

const (
	clustersUnset int32 = iota
	clustersOn
	clustersOff
)

// Load reports whether cluster handling is on.
func (c *clusterSetting) Load() bool {
	switch c.v.Load() {
	case clustersOn:
		return true
	case clustersOff:
		return false
	}
	def := clustersOn
	if os.Getenv("TUI_NO_CLUSTERS") != "" {
		def = clustersOff
	}
	// An explicit Store that raced this read wins.
	c.v.CompareAndSwap(clustersUnset, def)
	return c.v.Load() == clustersOn
}

// Store sets cluster handling, overriding the environment.
func (c *clusterSetting) Store(on bool) {
	if on {
		c.v.Store(clustersOn)
	} else {
		c.v.Store(clustersOff)
	}
}

// SetClusterWidth turns grapheme-cluster handling on or off for Width,
// Truncate, TrimLeftWidth and everything built on them. It is on by default:
// a ZWJ emoji sequence, a flag, an emoji with a skin-tone modifier or a base
// letter with combining marks is measured, truncated and trimmed as one unit,
// the way current terminals draw it. Turn it off for a terminal that does not
// join clusters, to count each rune on its own as before. The environment
// variable TUI_NO_CLUSTERS=1 does the same, read when the first width is measured. It is safe to call from
// any goroutine.
//
// Deprecated: for anything tied to one terminal use a Measurer. This function
// stays as the process-wide default that the zero Measurer follows, for an
// application to set once at start-up; a Program no longer calls it (the
// capability probe's answer is Program.Measurer and ResizeMsg.Measurer), so
// libraries must not either.
//
// A cluster is measured as its widest rune (at most 2 columns), except that an
// emoji followed by U+FE0F (variation selector 16) takes 2. Segmentation
// follows Unicode Standard Annex #29, except that CR LF is two clusters, so
// text measures as it always did.
func SetClusterWidth(on bool) { clusterWidthOn.Store(on) }

// clusterState is the extended grapheme cluster segmenter (UAX #29) fed one
// rune at a time. The zero value is the start of a string.
type clusterState struct {
	prev    gbClass
	pict    uint8 // 0 none, 1 after ExtPict Extend*, 2 after ExtPict Extend* ZWJ (GB11)
	incb    uint8 // 0 none, 1 after Consonant [Extend]*, 2 once a Linker followed (GB9c)
	ri      uint8 // regional indicators in the current run (GB12, GB13)
	base    bool  // the cluster's first rune is Extended_Pictographic
	keycap  uint8 // 0 none, 1 after a keycap base [0-9#*], 2 after base U+FE0F, 3 after base U+FE0F U+20E3
	started bool
}

// classify returns r's Grapheme_Cluster_Break class, whether it is
// Extended_Pictographic, and its Indic_Conjunct_Break class. Runes below U+0300
// skip the tables: only controls, CR, LF and the two ExtPict runes U+00A9 and
// U+00AE are not plain.
func classify(r rune) (gbClass, bool, incbClass) {
	if r < 0x300 {
		switch {
		case r == '\r':
			return gbCR, false, incbNone
		case r == '\n':
			return gbLF, false, incbNone
		case r < 0x20 || r >= 0x7f && r < 0xa0 || r == 0xad:
			return gbControl, false, incbNone
		}
		return gbOther, r == 0xa9 || r == 0xae, incbNone
	}
	if r < clusterTableLimit {
		p := clusterTable[r]
		return gbClass(p & 0xf), p&0x10 != 0, incbClass(p >> 5)
	}
	return graphemeBreakOf(r), isExtPict(r), incbOf(r)
}

// clusterTableLimit is the first rune not in clusterTable. Every rune with a
// non-default property below it is covered; above it (tag characters and
// variation selectors at U+E0000) classify falls back to the binary searches.
const clusterTableLimit = 0x30000

// clusterTable packs the three properties of each rune below
// clusterTableLimit into one byte, so classifying a rune is one load: the low
// four bits are the Grapheme_Cluster_Break class, bit 4 is
// Extended_Pictographic and bits 5 and 6 are the Indic_Conjunct_Break class.
var clusterTable [clusterTableLimit]uint8

func init() {
	for _, e := range generatedGraphemeBreak {
		for r := e.lo; r <= e.hi && r < clusterTableLimit; r++ {
			clusterTable[r] |= uint8(e.class)
		}
	}
	for _, e := range generatedExtPict {
		for r := e.lo; r <= e.hi && r < clusterTableLimit; r++ {
			clusterTable[r] |= 0x10
		}
	}
	for _, e := range generatedIndicConjunctBreak {
		for r := e.lo; r <= e.hi && r < clusterTableLimit; r++ {
			clusterTable[r] |= uint8(e.class) << 5
		}
	}
}

// advance feeds r and reports whether it belongs to the cluster in progress
// (false when r starts a new one, and for the first rune).
func (st *clusterState) advance(r rune) bool {
	cls, pict, ib := classify(r)
	join := st.started && st.joins(cls, pict, ib)
	switch {
	case cls == gbRegionalIndicator && join && st.prev == gbRegionalIndicator:
		st.ri++
	case cls == gbRegionalIndicator:
		st.ri = 1
	default:
		st.ri = 0
	}
	switch {
	case pict:
		st.pict = 1
	case cls == gbExtend && st.pict == 1:
	case cls == gbZWJ && st.pict == 1:
		st.pict = 2
	default:
		st.pict = 0
	}
	switch {
	case ib == incbConsonant:
		st.incb = 1
	case st.incb > 0 && ib == incbLinker:
		st.incb = 2
	case st.incb > 0 && ib == incbExtend:
	default:
		st.incb = 0
	}
	if !join {
		st.base = pict
		st.keycap = 0
		if isKeycapBase(r) {
			st.keycap = 1
		}
	} else {
		switch {
		case st.keycap == 1 && r == 0xFE0F:
			st.keycap = 2
		case st.keycap == 2 && r == 0x20E3:
			st.keycap = 3
		default:
			st.keycap = 0
		}
	}
	st.prev, st.started = cls, true
	return join
}

// joins applies the UAX #29 rules GB3 to GB13 to the boundary between the
// previous rune and one of class cur.
func (st *clusterState) joins(cur gbClass, pict bool, ib incbClass) bool {
	prev := st.prev
	switch {
	case prev == gbCR && cur == gbLF: // GB3
		return true
	case prev == gbCR || prev == gbLF || prev == gbControl: // GB4
		return false
	case cur == gbCR || cur == gbLF || cur == gbControl: // GB5
		return false
	case prev == gbL && (cur == gbL || cur == gbV || cur == gbLV || cur == gbLVT): // GB6
		return true
	case (prev == gbLV || prev == gbV) && (cur == gbV || cur == gbT): // GB7
		return true
	case (prev == gbLVT || prev == gbT) && cur == gbT: // GB8
		return true
	case cur == gbExtend || cur == gbZWJ || cur == gbSpacingMark: // GB9, GB9a
		return true
	case prev == gbPrepend: // GB9b
		return true
	case ib == incbConsonant && st.incb == 2: // GB9c
		return true
	case pict && st.pict == 2: // GB11
		return true
	case prev == gbRegionalIndicator && cur == gbRegionalIndicator && st.ri%2 == 1: // GB12, GB13
		return true
	}
	return false
}

// isKeycapBase reports whether r can start an emoji keycap sequence.
func isKeycapBase(r rune) bool { return r >= '0' && r <= '9' || r == '#' || r == '*' }

// asciiRunState is the clusterState after a run of printable ASCII ending in
// last, which starts the cluster in progress.
func asciiRunState(last byte) clusterState {
	st := clusterState{started: true}
	if isKeycapBase(rune(last)) {
		st.keycap = 1
	}
	return st
}

// joinWidth is the width of a cluster of width cur after r joins it: the
// widest rune so far, and 2 for an emoji made presentation by U+FE0F.
func (st *clusterState) joinWidth(cur, rw int, r rune) int {
	if r == 0xFE0F && st.base {
		return 2
	}
	if r == 0x20E3 && st.keycap == 3 {
		return 2 // keycap sequence base U+FE0F U+20E3 is emoji presentation
	}
	return max(cur, rw)
}

// next is advance with the one deviation from UAX #29 that Width keeps: CR LF
// is two clusters, so text with Windows line endings measures as it always did.
func (st *clusterState) next(r rune) bool {
	crlf := st.started && st.prev == gbCR && r == '\n'
	return st.advance(r) && !crlf
}

// splitClusters returns the extended grapheme clusters of s exactly as UAX #29
// defines them (CR LF is one). It ignores escape sequences: pass text with them
// stripped.
func splitClusters(s string) []string {
	var out []string
	var st clusterState
	start := 0
	for i, r := range s {
		if st.advance(r) || i == 0 {
			continue
		}
		out = append(out, s[start:i])
		start = i
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

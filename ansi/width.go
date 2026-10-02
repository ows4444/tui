package ansi

import (
	"strings"
	"unicode/utf8"
)

// StripANSI removes CSI escape sequences (as emitted by Style, e.g.
// "\x1b[1;31m...\x1b[0m") and string sequences (OSC, DCS, APC, SOS, PM,
// terminated by BEL or ST) from s, leaving the visible text behind. An
// unterminated string sequence runs to the end of s.
func StripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == 0x1b {
			if end, _ := escEnd(s, i); end > i {
				i = end
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// Width returns the display width of s, ignoring any embedded ANSI escape
// sequences. Each extended grapheme cluster counts for the columns it
// occupies: 2 for East Asian Wide/Fullwidth and emoji, 0 for combining and
// zero-width runes, 1 otherwise, and a ZWJ sequence, flag, emoji with a
// skin-tone modifier or base with combining marks is one cluster, the widest of
// its runes (see SetClusterWidth to count each rune on its own instead, and
// runeWidth for the per-rune widths).
//
// It measures in place, skipping CSI sequences as it goes, and allocates
// nothing; it agrees with measuring StripANSI(s) on every input. The one case
// that cannot be measured in place is a multi-byte rune whose bytes an escape
// sequence splits: StripANSI would join them, so Width falls back to it.
func Width(s string) int { return width(s, clusterWidthOn.Load()) }

// width is Width with the cluster setting passed in, so a Measurer can measure
// differently from the process default.
func width(s string, clusters bool) int {
	var st clusterState
	w, pending := 0, 0 // finished clusters, and the cluster in progress
	for i := 0; i < len(s); {
		c := s[i]
		if c == 0x1b {
			// SGR sequences are by far the commonest escape in styled text,
			// so they are skipped here without the general scanner.
			if i+2 < len(s) && s[i+1] == '[' {
				j := i + 2
				for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == ';' || s[j] == ':') {
					j++
				}
				if j < len(s) && s[j] == 'm' {
					i = j + 1
					continue
				}
			}
			if end, _ := escEnd(s, i); end > i {
				i = end
				continue
			}
		}
		if c >= 0x20 && c < 0x7f && st.prev != gbPrepend {
			// A run of printable ASCII: each byte starts a cluster (nothing
			// joins it, and only a Prepend rune before it could), so count the
			// run without the segmenter.
			j := i + 1
			for j < len(s) && s[j] >= 0x20 && s[j] < 0x7f {
				j++
			}
			w += pending + (j - i - 1)
			pending = 1
			st = asciiRunState(s[j-1])
			i = j
			continue
		}
		r, size := rune(c), 1
		if c >= utf8.RuneSelf {
			r, size = utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 && escapeWithin(s[i+1:], 3) {
				return measurePlain(StripANSI(s), clusters) // a rune may be split by an escape sequence
			}
		}
		rw := runeWidth(r)
		if clusters && st.next(r) {
			pending = st.joinWidth(pending, rw, r)
		} else {
			if !clusters {
				st = clusterState{}
			}
			w += pending
			pending = rw
		}
		i += size
	}
	return w + pending
}

// escapeWithin reports whether an ESC byte occurs in the first n bytes of s.
func escapeWithin(s string, n int) bool {
	if len(s) < n {
		n = len(s)
	}
	return strings.IndexByte(s[:n], 0x1b) >= 0
}

// slowWidth is Width defined as literally as possible: measure the text left
// after StripANSI. Width falls back to it for split runes.
func slowWidth(s string) int {
	return plainWidth(StripANSI(s))
}

// plainWidth measures text with no escape sequences, cluster by cluster.
func plainWidth(s string) int { return measurePlain(s, clusterWidthOn.Load()) }

func measurePlain(s string, clusters bool) int {
	var st clusterState
	w, pending := 0, 0
	for _, r := range s {
		rw := runeWidth(r)
		if clusters && st.next(r) {
			pending = st.joinWidth(pending, rw, r)
		} else {
			w += pending
			pending = rw
		}
	}
	return w + pending
}

// escEnd scans the escape sequence starting at s[i] (which must be ESC) and
// returns the index just past it, and whether it was terminated. It
// recognises CSI (ESC [ ... final byte 0x40-0x7E) and the string sequences OSC
// (ESC ]), DCS (ESC P), SOS (ESC X), PM (ESC ^) and APC (ESC _), which end at
// BEL or ST (ESC \), plus the short forms: a two-byte sequence (ESC and one
// byte 0x30-0x7E, such as ESC 7, ESC 8, ESC = or ESC c) and an nF sequence
// (ESC, intermediate bytes 0x20-0x2F, one final byte 0x30-0x7E, such as
// ESC ( B). An unterminated sequence runs to len(s). If s[i] does not begin a
// recognised sequence it returns i, false. It never allocates.
func escEnd(s string, i int) (end int, terminated bool) {
	if i+1 >= len(s) {
		return i, false
	}
	switch s[i+1] {
	case '[':
		// The final byte is the first one in 0x40-0x7E after the introducer
		// ('[' itself is in that range, so it is consumed first).
		for j := i + 2; j < len(s); j++ {
			if c := s[j]; c >= 0x40 && c <= 0x7E {
				return j + 1, true
			}
		}
		return len(s), false
	case ']', 'P', 'X', '^', '_':
		for j := i + 2; j < len(s); j++ {
			switch s[j] {
			case 0x07:
				return j + 1, true
			case 0x1b:
				if j+1 < len(s) && s[j+1] == '\\' {
					return j + 2, true
				}
			}
		}
		return len(s), false
	}
	if c := s[i+1]; c >= 0x30 && c <= 0x7E {
		return i + 2, true // ESC 7, ESC 8, ESC =, ESC c, ...
	} else if c >= 0x20 && c <= 0x2F {
		for j := i + 2; j < len(s); j++ {
			if f := s[j]; f >= 0x30 && f <= 0x7E {
				return j + 1, true // ESC ( B, ESC # 8, ...
			} else if f < 0x20 || f > 0x2F {
				return i, false // not an nF sequence
			}
		}
		return len(s), false
	}
	return i, false
}

// osc8 reports whether seq (a complete escape sequence) is an OSC 8
// hyperlink command, and if so whether it opens a link (non-empty URI).
func osc8(seq string) (isLink, opens bool) {
	if len(seq) < 5 || seq[1] != ']' || seq[2] != '8' || seq[3] != ';' {
		return false, false
	}
	body := seq[4:]
	k := strings.IndexByte(body, ';')
	if k < 0 {
		return false, false
	}
	uri := body[k+1:]
	if n := len(uri); n > 0 && uri[n-1] == 0x07 {
		uri = uri[:n-1]
	} else if n > 1 && uri[n-2] == 0x1b && uri[n-1] == '\\' {
		uri = uri[:n-2]
	}
	return true, uri != ""
}

// linkClose closes an open OSC 8 hyperlink.
const linkClose = "\x1b]8;;\x1b\\"

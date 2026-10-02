package ansi

import (
	"strings"
	"unicode/utf8"
)

// Truncate returns s cut to at most width visible columns, the same
// counting Width uses (ignoring escape sequences, grapheme clusters weighted
// as Width weighs them). A wide rune or a cluster that would straddle the cut is
// dropped whole, so a ZWJ sequence, flag or base with its marks is never split.
// Escape sequences are preserved up through the cut point, so styling
// applied before it still renders; if that leaves styling "open" (no
// Reset was reached before the cut), a Reset is appended so it can't
// bleed into whatever's rendered after this string. OSC, DCS, APC, SOS and PM
// sequences are skipped like CSI; an OSC 8 hyperlink left open at the cut is
// closed with an empty OSC 8, so the output is balanced.
func Truncate(s string, width int) string { return truncate(s, width, clusterWidthOn.Load()) }

// truncate is Truncate with the cluster setting passed in, for Measurer.
func truncate(s string, width int, clusters bool) string {
	if width <= 0 {
		return ""
	}

	// Everything Truncate keeps (escape sequences and the runes that fit) is a
	// verbatim prefix of s, so scan to find where it ends and slice, instead
	// of copying into a new string. Only leaving styling open costs an
	// allocation, for the Reset appended after it.
	var st clusterState
	visible, pending := 0, 0 // finished clusters, and the cluster in progress
	open, openAtStart := false, false
	link, linkAtStart := false, false
	clusterStart := 0
	cut := len(s)
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			if j, ok := escEnd(s, i); j > i {
				if !ok {
					// A sequence cut off by the end of the string: keeping it
					// would fuse with whatever follows (e.g. the Reset appended
					// below), so drop it.
					cut = i
					break
				}
				if s[i+1] == '[' {
					open = s[i:j] != Reset
				} else if isLink, opens := osc8(s[i:j]); isLink {
					link = opens
				}
				i = j
				continue
			}
		}
		if c := s[i]; c >= 0x20 && c < 0x7f && st.prev != gbPrepend {
			// A run of printable ASCII: every byte is its own one-column
			// cluster (only a Prepend rune before it could join it), so keep as
			// many as fit without running the segmenter.
			j := i + 1
			for j < len(s) && s[j] >= 0x20 && s[j] < 0x7f {
				j++
			}
			visible += pending
			if fit := width - visible; fit < j-i {
				if fit > 0 {
					cut = i + fit
				} else {
					cut = i
				}
				break
			}
			visible += j - i - 1
			pending, clusterStart, openAtStart, linkAtStart = 1, j-1, open, link
			st = asciiRunState(s[j-1])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if clusters && st.next(r) {
			// r joins the cluster in progress; it stays whole or goes whole.
			grown := st.joinWidth(pending, rw, r)
			if visible+grown > width {
				cut, open, link = clusterStart, openAtStart, linkAtStart
				break
			}
			pending = grown
		} else {
			visible += pending
			if visible+rw > width {
				cut = i
				break
			}
			pending, clusterStart, openAtStart, linkAtStart = rw, i, open, link
		}
		i += size
	}
	switch {
	case open && link:
		return s[:cut] + Reset + linkClose
	case open:
		return s[:cut] + Reset
	case link:
		return s[:cut] + linkClose
	}
	return s[:cut]
}

// TrimLeftWidth returns s with its first width visible columns removed —
// the inverse of Truncate. If the cut lands inside a wide rune, that rune is
// dropped and a space stands in for its remaining column so the remainder
// keeps its column alignment. Zero-width runes at the cut boundary stay with
// the remainder, for the other half of a string once it's been
// split at a column (e.g. layout.Overlay, splicing content into the
// middle of an existing line).
//
// Unlike Truncate, this can't just carry escape sequences through
// unmodified: if the cut lands in the middle of a styled span, the
// discarded prefix carried the SGR sequence that made the rest of the
// span styled, so it has to be re-emitted at the start of the remainder.
// This tracks only the single most recently seen non-Reset sequence as
// "the active style" and re-applies exactly that — correct for spans built
// the way Style.Render does (one opening sequence, content, one closing
// Reset, never nested or overlapping), which is the only shape this
// module's own code ever produces, but not a general ANSI state machine.
func TrimLeftWidth(s string, width int) string {
	return trimLeftWidth(s, width, clusterWidthOn.Load())
}

// trimLeftWidth is TrimLeftWidth with the cluster setting passed in, for Measurer.
func trimLeftWidth(s string, width int, clusters bool) string {
	if width <= 0 {
		return s
	}

	var st clusterState
	var active string
	done, pending := 0, 0 // finished clusters, and the cluster in progress
	i := 0
	for i < len(s) {
		if s[i] == 0x1b {
			j, _ := escEnd(s, i)
			if j > i && s[i+1] != '[' {
				i = j // string sequences don't affect the tracked SGR style
				continue
			}
		}
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) {
				c := s[j]
				j++
				if c >= 0x40 && c <= 0x7E {
					break
				}
			}
			seq := s[i:j]
			if seq == Reset {
				active = ""
			} else {
				active = seq
			}
			i = j
			continue
		}
		if c := s[i]; c >= 0x20 && c < 0x7f && st.prev != gbPrepend {
			// A run of printable ASCII: each byte is a one-column cluster, so
			// trim as many as the width still needs without the segmenter.
			j := i + 1
			for j < len(s) && s[j] >= 0x20 && s[j] < 0x7f {
				j++
			}
			need := width - (done + pending)
			if need <= 0 {
				break
			}
			k := min(need, j-i)
			done += pending + k - 1
			pending = 1
			st = asciiRunState(s[i+k-1])
			i += k
			if i < j {
				break // the width is used up part way through the run
			}
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runeWidth(r)
		if clusters && st.next(r) {
			// The rest of a cluster that began inside the trimmed part goes
			// with it.
			pending = st.joinWidth(pending, rw, r)
		} else {
			if done+pending >= width {
				break
			}
			done += pending
			pending = rw
		}
		i += size
	}

	pad := ""
	if over := done + pending - width; over > 0 {
		pad = strings.Repeat(" ", over)
	}
	return active + pad + s[i:]
}

package ansi

// runeWidth returns the number of terminal columns r occupies: 0 for
// combining and zero-width runes, 2 for East Asian Wide/Fullwidth and
// emoji-presentation runes, 1 otherwise. The tables are generated from Unicode
// unicodeVersion by internal/tools/genwidth (EastAsianWidth W/F, Emoji_Presentation,
// general category Mn/Me/Cf); see runewidth_tables.go. It is the width of a
// single rune: Width, Truncate and TrimLeftWidth combine runes into grapheme
// clusters (see cluster.go) unless SetClusterWidth(false).
func runeWidth(r rune) int {
	if r < 0x300 {
		// Fast path: ASCII and Latin-1/Extended-A/B. C0 controls (including
		// tab and newline), DEL and C1 controls occupy no column; expand tabs
		// first (ExpandTabs) to measure them as spaces. This includes U+00AD
		// SOFT HYPHEN, a Cf rune the tables list as zero width: terminals
		// (xterm, glibc's wcwidth) draw it in one column, so it stays one.
		if r < 0x20 || (r >= 0x7f && r < 0xa0) {
			return 0
		}
		return 1
	}
	// One binary search: the table lists every rune that is not one column
	// wide, with its width (zero wins where a rune is both zero and wide).
	lo, hi := 0, len(generatedWidthRanges)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch e := &generatedWidthRanges[mid]; {
		case r < e.lo:
			hi = mid - 1
		case r > e.hi:
			lo = mid + 1
		default:
			return int(e.w)
		}
	}
	return 1
}

// widthRange is an inclusive range of runes that share a column width, one row
// of generatedWidthRanges.
type widthRange struct {
	lo, hi rune
	w      uint8
}

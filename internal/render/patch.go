package render

import (
	"math/bits"
)

// patchRow updates prevRow, the cells of oldSrc, in place to the cells of
// newSrc when the two lines have the same length and differ only in printable
// ASCII bytes that lie outside their escape sequences: every escape sequence,
// and so every style, is then identical, and each differing byte is one cell
// whose character changes and whose style does not. info is the old row's
// escape layout; the old row must be simple (see rowInfo). It returns the
// indices of the cells that changed and their byte offsets in newSrc. It
// reports false, and leaves prevRow untouched, when the lines are not that
// simple and must be parsed: a difference inside an escape sequence or of a
// control or non-ASCII byte, a changed cell beyond the row (trailing blanks
// are trimmed from rows), or a change that would leave a trailing blank to
// trim. The cells hold no pointers, so updating them in place costs a store
// per changed cell and retains nothing.
func patchRow(oldSrc, newSrc string, prevRow []cell, info rowInfo, idx, pos []int32) (cells, offs []int32, ok bool) {
	n := len(newSrc)
	if len(oldSrc) != n || !info.simple {
		return nil, nil, false
	}
	idx, pos = idx[:0], pos[:0]
	esc := info.esc
	ep, escBytes := 0, 0
	// check validates the differing byte at k and records its cell.
	check := func(k int) bool {
		b := newSrc[k]
		if b < 0x20 || b >= 0x7f {
			return false
		}
		for ep < len(esc) && int(esc[ep+1]) <= k {
			escBytes += int(esc[ep+1] - esc[ep])
			ep += 2
		}
		if ep < len(esc) && int(esc[ep]) <= k {
			return false // the byte is inside an escape sequence
		}
		cellIdx := k - escBytes
		if cellIdx >= len(prevRow) || b == ' ' && prevRow[cellIdx].st == 0 {
			return false
		}
		idx = append(idx, int32(cellIdx)) // #nosec G115 -- bounded by the line length
		pos = append(pos, int32(k))       // #nosec G115 -- bounded by the line length
		return true
	}
	k := 0
	for ; k+8 <= n; k += 8 {
		x := word(oldSrc, k) ^ word(newSrc, k)
		for x != 0 {
			tz := bits.TrailingZeros64(x) &^ 7 // bit offset of the lowest differing byte
			if !check(k + tz>>3) {
				return nil, nil, false
			}
			x &^= 0xff << tz
		}
	}
	for ; k < n; k++ {
		if oldSrc[k] != newSrc[k] && !check(k) {
			return nil, nil, false
		}
	}
	if len(idx) == 0 {
		return nil, nil, false // the lines differ somewhere, so a byte must have
	}
	for j, ci := range idx {
		prevRow[ci].ch = uint32(newSrc[pos[j]])
	}
	return idx, pos, true
}

// word loads the 8 bytes of s at k as a little-endian integer.
func word(s string, k int) uint64 {
	_ = s[k+7]
	return uint64(s[k]) | uint64(s[k+1])<<8 | uint64(s[k+2])<<16 | uint64(s[k+3])<<24 |
		uint64(s[k+4])<<32 | uint64(s[k+5])<<40 | uint64(s[k+6])<<48 | uint64(s[k+7])<<56
}

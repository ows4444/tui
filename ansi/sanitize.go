package ansi

import (
	"strings"
	"unicode/utf8"
)

// Sanitize neutralises untrusted text for display: it removes every escape
// sequence (CSI, OSC, DCS, SOS, PM, APC), a lone ESC, and all C0 and C1
// control characters (and DEL) except '\n'. Tabs are dropped too; call
// ExpandTabs first to keep them as spaces. Printable text, including invalid
// UTF-8, is returned unchanged.
func Sanitize(s string) string { return sanitize(s, false) }

// SanitizeKeepSGR is Sanitize but keeps SGR (style) sequences, ESC [ params m
// where params contain only digits, ';' and ':'. Every other escape sequence
// and control character is removed. An unterminated SGR-looking sequence is
// dropped.
func SanitizeKeepSGR(s string) string { return sanitize(s, true) }

func sanitize(s string, keepSGR bool) string {
	if !needsSanitize(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		c := s[i]
		switch {
		case c == 0x1b:
			end, ok := escEnd(s, i)
			if end > i {
				if keepSGR && ok && isSGR(s[i:end]) {
					b.WriteString(s[i:end])
				}
				i = end
			} else {
				i++ // lone ESC or unrecognised introducer: drop the ESC
			}
		case c == '\n':
			b.WriteByte(c)
			i++
		case c < 0x20 || c == 0x7f:
			i++
		case c < utf8.RuneSelf:
			b.WriteByte(c)
			i++
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			if r < 0xa0 && r >= 0x80 && !(r == utf8.RuneError && size == 1) {
				i += size // C1 control
				continue
			}
			b.WriteString(s[i : i+size])
			i += size
		}
	}
	return b.String()
}

const (
	lo7  = 0x0101010101010101
	m7f  = 0x7f7f7f7f7f7f7f7f
	hi8  = 0x8080808080808080
	bE0  = lo7 * 0xe0
	bNL  = lo7 * '\n'
	bDEL = lo7 * 0x7f
	bC2  = lo7 * 0xc2
)

// zeroBytes returns 0x80 in each byte of v that is zero, exactly.
func zeroBytes(v uint64) uint64 { return ^(((v & m7f) + m7f) | v | m7f) }

// needsSanitize reports whether s has a byte Sanitize would act on: one below
// 0x20 other than '\n', 0x7f, or 0xc2 (the lead byte of every C1 control). It
// tests eight bytes at a time.
func needsSanitize(s string) bool {
	i := 0
	for ; i+8 <= len(s); i += 8 {
		x := uint64(s[i]) | uint64(s[i+1])<<8 | uint64(s[i+2])<<16 | uint64(s[i+3])<<24 |
			uint64(s[i+4])<<32 | uint64(s[i+5])<<40 | uint64(s[i+6])<<48 | uint64(s[i+7])<<56
		d, c := x^bDEL, x^bC2
		if ((x-lo7*0x20)&^x|(d-lo7)&^d|(c-lo7)&^c)&hi8 == 0 {
			continue // cheap test: no candidate byte in this word
		}
		// A candidate may be a '\n' (or sit above a true match); decide exactly.
		if zeroBytes(x&bE0)&^zeroBytes(x^bNL)|zeroBytes(d)|zeroBytes(c) != 0 {
			return true
		}
	}
	for ; i < len(s); i++ {
		c := s[i]
		if (c < 0x20 && c != '\n') || c == 0x7f || c == 0xc2 {
			return true
		}
	}
	return false
}

// isSGR reports whether seq (a complete CSI sequence) is ESC [ params m with
// params made only of digits, ';' and ':'.
func isSGR(seq string) bool {
	if len(seq) < 3 || seq[1] != '[' || seq[len(seq)-1] != 'm' {
		return false
	}
	for _, c := range []byte(seq[2 : len(seq)-1]) {
		if !(c >= '0' && c <= '9') && c != ';' && c != ':' {
			return false
		}
	}
	return true
}

// tabStop is the tab width used by ExpandTabs.
const tabStop = 8

// ExpandTabs replaces each tab with spaces up to the next multiple of 8
// columns. It is ANSI-aware: escape sequences take no columns, and the column
// counter restarts after '\n' and '\r'. Text without tabs is returned as is.
func ExpandTabs(s string) string {
	if strings.IndexByte(s, '\t') < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	col, start := 0, 0 // start: beginning of the unmeasured plain run
	flush := func(end int) {
		if end > start {
			col += plainWidth(s[start:end])
			b.WriteString(s[start:end])
		}
	}
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == 0x1b:
			end, _ := escEnd(s, i)
			if end == i {
				i++
				continue
			}
			flush(i)
			b.WriteString(s[i:end])
			i, start = end, end
		case c == '\t':
			flush(i)
			n := tabStop - col%tabStop
			b.WriteString(strings.Repeat(" ", n))
			col += n
			i++
			start = i
		case c == '\n' || c == '\r':
			flush(i)
			b.WriteByte(c)
			col = 0
			i++
			start = i
		default:
			i++
		}
	}
	flush(len(s))
	return b.String()
}

// PlainASCIIWidth reports whether every byte of s is printable ASCII (0x20
// through 0x7e) and, if so, returns its display width, which is len(s). For
// such text Sanitize, SanitizeKeepSGR and ExpandTabs are all the identity, so
// a caller that would sanitize and then measure can do both in this one pass
// and skip them. It returns 0, false for anything else, including text with a
// tab, an escape, a DEL or a non-ASCII byte. The empty string is plain, width 0.
func PlainASCIIWidth(s string) (int, bool) {
	i := 0
	for ; i+8 <= len(s); i += 8 {
		x := uint64(s[i]) | uint64(s[i+1])<<8 | uint64(s[i+2])<<16 | uint64(s[i+3])<<24 |
			uint64(s[i+4])<<32 | uint64(s[i+5])<<40 | uint64(s[i+6])<<48 | uint64(s[i+7])<<56
		d := x ^ bDEL
		// A high bit (non-ASCII), a byte below 0x20 or a 0x7f: all three are
		// exact "some byte matches" tests (a false positive needs a true
		// match in a lower byte).
		if (x|(x-lo7*0x20)&^x|(d-lo7)&^d)&hi8 != 0 {
			return 0, false
		}
	}
	for ; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c >= 0x7f {
			return 0, false
		}
	}
	return len(s), true
}

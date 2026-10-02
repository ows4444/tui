package ansi

import (
	"strconv"
	"strings"
)

// DowngradeString rewrites the colour codes in s's SGR sequences to what p
// can display: truecolor and 256-colour codes become the nearest 256- or
// 16-colour ones, and under NoColor every foreground, background and
// underline colour is removed. Attributes (bold, dim, italic, underline
// shapes, reverse, ...) are left alone, as NO_COLOR asks for no colour, not
// no styling. Text, non-SGR sequences (cursor movement, OSC hyperlinks) and
// TrueColor input pass through untouched, so the visible text never
// changes. It is what Program applies to each frame under WithColorProfile.
func DowngradeString(s string, p Profile) string {
	if p >= TrueColor || !strings.Contains(s, "\x1b[") {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if s[i] != 0x1b || i+1 >= len(s) || s[i+1] != '[' {
			b = append(b, s[i])
			i++
			continue
		}
		// Same CSI scan as StripANSI: the final byte is the first one in
		// 0x40-0x7E after the introducer.
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
			j++
		}
		if j >= len(s) { // unterminated: leave as is
			b = append(b, s[i:]...)
			break
		}
		if s[j] == 'm' {
			seq := rewriteSGR(s[i+2:j], p)
			if seq == "" {
				// Stray ESCs right before a sequence we are removing would
				// otherwise fuse with whatever follows (ESC + "[" is a new
				// CSI). They did nothing on their own, so drop them.
				for len(b) > 0 && b[len(b)-1] == 0x1b {
					b = b[:len(b)-1]
				}
			}
			b = append(b, seq...)
		} else {
			b = append(b, s[i:j+1]...)
		}
		i = j + 1
	}
	return string(b)
}

// rewriteSGR returns the full escape sequence (or "" if every code was
// removed) for the SGR parameter string params, downgraded to p.
func rewriteSGR(params string, p Profile) string {
	if params == "" { // ESC[m is a reset
		return "\x1b[m"
	}
	toks := strings.Split(params, ";")
	out := make([]string, 0, len(toks))
	for i := 0; i < len(toks); i++ {
		t := toks[i]

		// Colon form: 38:2::r:g:b / 38:5:n (and 48, 58). Other colon
		// parameters (4:3 curly underline) pass through.
		if k := strings.IndexByte(t, ':'); k > 0 {
			if base := t[:k]; base == "38" || base == "48" || base == "58" {
				if c, ok := parseColonColor(t[k+1:]); ok {
					out = appendColor(out, base, c, p)
				}
				continue
			}
			out = append(out, t)
			continue
		}

		n, err := strconv.Atoi(t)
		if err != nil {
			out = append(out, t)
			continue
		}
		switch {
		case n == 38 || n == 48 || n == 58:
			c, used, ok := parseSemiColor(toks[i+1:])
			i += used
			if ok {
				out = appendColor(out, t, c, p)
			}
		case isBasicColorCode(n):
			if p == NoColor {
				continue
			}
			out = append(out, t)
		case n == 39 || n == 49 || n == 59: // default fg/bg/underline colour
			if p == NoColor {
				continue
			}
			out = append(out, t)
		default:
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(out, ";") + "m"
}

func isBasicColorCode(n int) bool {
	return (n >= 30 && n <= 37) || (n >= 40 && n <= 47) || (n >= 90 && n <= 97) || (n >= 100 && n <= 107)
}

// parseSemiColor reads the arguments after 38/48/58 in ';' form ("5;n" or
// "2;r;g;b"), returning the colour and how many tokens it consumed. A
// truncated or invalid spec consumes what is there and reports !ok.
func parseSemiColor(rest []string) (c Color, used int, ok bool) {
	if len(rest) == 0 {
		return nil, 0, false
	}
	switch rest[0] {
	case "5":
		if len(rest) < 2 {
			return nil, len(rest), false
		}
		n, err := strconv.Atoi(rest[1])
		if err != nil || n < 0 || n > 255 {
			return nil, 2, false
		}
		return Color256(n), 2, true
	case "2":
		if len(rest) < 4 {
			return nil, len(rest), false
		}
		var v [3]int
		for k := 0; k < 3; k++ {
			n, err := strconv.Atoi(rest[1+k])
			if err != nil || n < 0 || n > 255 {
				return nil, 4, false
			}
			v[k] = n
		}
		return RGB{uint8(v[0]), uint8(v[1]), uint8(v[2])}, 4, true // #nosec G115 -- range-checked above
	}
	return nil, 0, false
}

// parseColonColor reads the part after "38:" in colon form: "5:n",
// "2::r:g:b" (empty colour-space id) or "2:r:g:b".
func parseColonColor(s string) (Color, bool) {
	f := strings.Split(s, ":")
	switch {
	case len(f) == 2 && f[0] == "5":
		n, err := strconv.Atoi(f[1])
		if err != nil || n < 0 || n > 255 {
			return nil, false
		}
		return Color256(n), true
	case len(f) >= 4 && f[0] == "2":
		v := f[len(f)-3:]
		var c [3]int
		for k, x := range v {
			n, err := strconv.Atoi(x)
			if err != nil || n < 0 || n > 255 {
				return nil, false
			}
			c[k] = n
		}
		return RGB{uint8(c[0]), uint8(c[1]), uint8(c[2])}, true // #nosec G115 -- range-checked above
	}
	return nil, false
}

// appendColor appends the code(s) for colour c in the slot named by base
// ("38" foreground, "48" background, "58" underline) at profile p.
func appendColor(out []string, base string, c Color, p Profile) []string {
	if p == NoColor {
		return out
	}
	switch v := c.(type) {
	case RGB:
		switch p {
		case ANSI256:
			return append(out, base, "5", strconv.Itoa(int(rgbTo256(v))))
		case ANSI16:
			return appendBasic(out, base, rgbTo16(v))
		}
	case Color256:
		if p == ANSI16 {
			if v < 16 {
				return appendBasic(out, base, BasicColor(v))
			}
			return appendBasic(out, base, rgbTo16(color256RGB(v)))
		}
		return append(out, base, "5", strconv.Itoa(int(v)))
	}
	return out
}

// appendBasic appends the 16-colour code for c in the given slot. The
// underline slot has no 16-colour form, so it is dropped.
func appendBasic(out []string, base string, c BasicColor) []string {
	switch base {
	case "38":
		return append(out, c.fgCode())
	case "48":
		return append(out, c.bgCode())
	}
	return out
}

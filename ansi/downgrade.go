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
			n := len(b)
			b = rewriteSGR(b, s[i+2:j], p)
			if len(b) == n {
				// Stray ESCs right before a sequence we are removing would
				// otherwise fuse with whatever follows (ESC + "[" is a new
				// CSI). They did nothing on their own, so drop them.
				for len(b) > 0 && b[len(b)-1] == 0x1b {
					b = b[:len(b)-1]
				}
			}
		} else {
			b = append(b, s[i:j+1]...)
		}
		i = j + 1
	}
	return string(b)
}

// sgrParams walks the ';'-separated parameters of an SGR sequence the way
// strings.Split would, without allocating: "" yields one empty parameter
// and a trailing ';' yields a final empty one.
type sgrParams struct {
	s   string
	pos int // past the last parameter when pos > len(s)
}

func (t *sgrParams) next() (string, bool) {
	if t.pos > len(t.s) {
		return "", false
	}
	rest := t.s[t.pos:]
	if k := strings.IndexByte(rest, ';'); k >= 0 {
		t.pos += k + 1
		return rest[:k], true
	}
	t.pos = len(t.s) + 1
	return rest, true
}

// sgrWriter appends the parameters of one SGR sequence to b, opening the
// sequence at the first parameter, so a sequence that keeps none adds
// nothing.
type sgrWriter struct {
	b []byte
	n int
}

func (w *sgrWriter) sep() {
	if w.n == 0 {
		w.b = append(w.b, "\x1b["...)
	} else {
		w.b = append(w.b, ';')
	}
	w.n++
}

func (w *sgrWriter) param(s string) {
	w.sep()
	w.b = append(w.b, s...)
}

func (w *sgrWriter) int(n int) {
	w.sep()
	w.b = strconv.AppendInt(w.b, int64(n), 10)
}

// rewriteSGR appends to dst the full escape sequence for the SGR parameter
// string params, downgraded to p, or nothing if every code was removed. It
// writes straight into dst: it runs on every SGR of every frame under a
// downgraded profile, NO_COLOR included.
func rewriteSGR(dst []byte, params string, p Profile) []byte {
	if params == "" { // ESC[m is a reset
		return append(dst, "\x1b[m"...)
	}
	w := sgrWriter{b: dst}
	toks := sgrParams{s: params}
	for t, ok := toks.next(); ok; t, ok = toks.next() {
		// Colon form: 38:2::r:g:b / 38:5:n (and 48, 58). Other colon
		// parameters (4:3 curly underline) pass through.
		if k := strings.IndexByte(t, ':'); k > 0 {
			if base := t[:k]; base == "38" || base == "48" || base == "58" {
				if c, ok := parseColonColor(t[k+1:]); ok {
					w.color(base, c, p)
				}
				continue
			}
			w.param(t)
			continue
		}

		n, err := strconv.Atoi(t)
		if err != nil {
			w.param(t)
			continue
		}
		switch {
		case n == 38 || n == 48 || n == 58:
			if c, ok := parseSemiColor(&toks); ok {
				w.color(t, c, p)
			}
		case isBasicColorCode(n):
			if p == NoColor {
				continue
			}
			w.param(t)
		case n == 39 || n == 49 || n == 59: // default fg/bg/underline colour
			if p == NoColor {
				continue
			}
			w.param(t)
		default:
			w.param(t)
		}
	}
	if w.n > 0 {
		w.b = append(w.b, 'm')
	}
	return w.b
}

func isBasicColorCode(n int) bool {
	return (n >= 30 && n <= 37) || (n >= 40 && n <= 47) || (n >= 90 && n <= 97) || (n >= 100 && n <= 107)
}

// sgrColor is a colour read from an SGR sequence: an RGB triple when isRGB,
// else a 256-colour index. It is a struct rather than a Color so that
// reading one does not box it.
type sgrColor struct {
	rgb   RGB
	idx   Color256
	isRGB bool
}

// parseSemiColor reads the arguments after 38/48/58 in ';' form ("5;n" or
// "2;r;g;b") from toks. A truncated or invalid spec consumes what is there
// and reports !ok; an unknown colour space consumes nothing.
func parseSemiColor(toks *sgrParams) (c sgrColor, ok bool) {
	pos := toks.pos
	space, ok := toks.next()
	switch {
	case !ok:
		return sgrColor{}, false
	case space == "5":
		t, ok := toks.next()
		if !ok {
			return sgrColor{}, false
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 || n > 255 {
			return sgrColor{}, false
		}
		return sgrColor{idx: Color256(n)}, true
	case space == "2":
		var f [3]string
		for k := range f {
			if f[k], ok = toks.next(); !ok {
				return sgrColor{}, false
			}
		}
		var v [3]uint8
		for k, t := range f {
			n, err := strconv.Atoi(t)
			if err != nil || n < 0 || n > 255 {
				return sgrColor{}, false
			}
			v[k] = uint8(n) // #nosec G115 -- range-checked above
		}
		return sgrColor{rgb: RGB{v[0], v[1], v[2]}, isRGB: true}, true
	}
	toks.pos = pos
	return sgrColor{}, false
}

// parseColonColor reads the part after "38:" in colon form: "5:n",
// "2::r:g:b" (empty colour-space id) or "2:r:g:b".
func parseColonColor(s string) (sgrColor, bool) {
	space, rest, _ := strings.Cut(s, ":")
	switch fields := strings.Count(s, ":") + 1; {
	case fields == 2 && space == "5":
		n, err := strconv.Atoi(rest)
		if err != nil || n < 0 || n > 255 {
			return sgrColor{}, false
		}
		return sgrColor{idx: Color256(n)}, true
	case fields >= 4 && space == "2":
		// The last three fields are r, g and b.
		var v [3]uint8
		for k := 2; k >= 0; k-- {
			i := strings.LastIndexByte(rest, ':')
			n, err := strconv.Atoi(rest[i+1:])
			if err != nil || n < 0 || n > 255 {
				return sgrColor{}, false
			}
			v[k] = uint8(n) // #nosec G115 -- range-checked above
			rest = rest[:max(i, 0)]
		}
		return sgrColor{rgb: RGB{v[0], v[1], v[2]}, isRGB: true}, true
	}
	return sgrColor{}, false
}

// color writes the code(s) for colour c in the slot named by base ("38"
// foreground, "48" background, "58" underline) at profile p.
func (w *sgrWriter) color(base string, c sgrColor, p Profile) {
	if p == NoColor {
		return
	}
	switch v := c.idx; {
	case c.isRGB && p == ANSI16:
		w.basic(base, rgbTo16(c.rgb))
	case c.isRGB:
		w.param(base)
		w.param("5")
		w.int(int(rgbTo256(c.rgb)))
	case p != ANSI16:
		w.param(base)
		w.param("5")
		w.int(int(v))
	case v < 16:
		w.basic(base, BasicColor(v))
	default:
		w.basic(base, rgbTo16(color256RGB(v)))
	}
}

// basic writes the 16-colour code for c in the given slot. The underline
// slot has no 16-colour form, so it is dropped.
func (w *sgrWriter) basic(base string, c BasicColor) {
	switch base {
	case "38":
		w.sep()
		w.b = c.appendSGR(w.b, slotFg)
	case "48":
		w.sep()
		w.b = c.appendSGR(w.b, slotBg)
	}
}

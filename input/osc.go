package input

import (
	"strings"
)

// BackgroundUnknownMsg is sent once, to Update, when background detection is
// on and the terminal did not answer the background-colour query within the
// timeout. tui.BackgroundUnknownMsg is an alias of it. It lives here, below the
// root package, so theme.Detect can recognise it (through BackgroundUnknown)
// without importing either.
type BackgroundUnknownMsg struct{}

// BackgroundUnknown marks m as the "no answer" message; theme.Detect looks for
// this method.
func (BackgroundUnknownMsg) BackgroundUnknown() {}

// BackgroundColorEvent is the terminal's answer to an OSC 11 background
// query (ansi.QueryBackgroundColor), with each channel scaled to 8 bits.
type BackgroundColorEvent struct {
	R, G, B uint8
}

// BackgroundRGB returns the colour; theme.Detect looks for this method.
func (e BackgroundColorEvent) BackgroundRGB() (r, g, b uint8) { return e.R, e.G, e.B }

// maxStringLen bounds how much of a string sequence (OSC, DCS, SOS, PM, APC)
// is retained. Bytes past it are discarded through the terminator, never
// leaked as keys.
const maxStringLen = 64 * 1024

// ReplyEvent is a terminal string-sequence reply that is not otherwise
// decoded (a DCS XTGETTCAP answer, an APC kitty graphics answer, a non-11 OSC
// reply, ...). Kind is the introducer's final byte: ']' OSC, 'P' DCS,
// 'X' SOS, '^' PM, '_' APC. Data is the payload without introducer or
// terminator, truncated to 64 KiB.
type ReplyEvent struct {
	Kind byte
	Data string
}

// parseString consumes a string sequence (after ESC and its introducer kind)
// through BEL or ST (ESC \), so its bytes never leak out as keypresses. An
// OSC 11 reply of the form "11;rgb:R/G/B" yields a BackgroundColorEvent;
// everything else yields a ReplyEvent. An unterminated string ends at EOF.
func (rd *Reader) parseString(kind byte) (Event, error) {
	var sb strings.Builder
	for {
		b, err := rd.r.ReadByte()
		if err != nil {
			break
		}
		if b == 0x07 {
			break
		}
		if b == 0x1b {
			// ST is ESC \; any other ESC ends the string (it is a fresh
			// sequence's introducer, which is dropped).
			if n, err := rd.r.Peek(1); err == nil && n[0] == '\\' {
				_, _ = rd.r.ReadByte()
			}
			break
		}
		if sb.Len() < maxStringLen {
			sb.WriteByte(b)
		}
	}
	if kind == ']' {
		if ev, ok := parseBackgroundReply(sb.String()); ok {
			return ev, nil
		}
		if ev, ok := parsePaletteReply(sb.String()); ok {
			return ev, nil
		}
	}
	return ReplyEvent{Kind: kind, Data: sb.String()}, nil
}

// PaletteColorEvent is the terminal's answer to an OSC 4 palette query
// (ansi.QueryPalette) for one colour, with each channel scaled to 8 bits.
type PaletteColorEvent struct {
	Index   uint8 // 0-255, the palette slot
	R, G, B uint8
}

// PaletteColor returns the slot and colour; theme.Palette.Observe looks for
// this method.
func (e PaletteColorEvent) PaletteColor() (index, r, g, b uint8) {
	return e.Index, e.R, e.G, e.B
}

// parsePaletteReply decodes "4;N;rgb:R/G/B".
func parsePaletteReply(s string) (PaletteColorEvent, bool) {
	rest, ok := strings.CutPrefix(s, "4;")
	if !ok {
		return PaletteColorEvent{}, false
	}
	num, spec, ok := strings.Cut(rest, ";")
	if !ok || len(num) < 1 || len(num) > 3 {
		return PaletteColorEvent{}, false
	}
	n := 0
	for _, ch := range num {
		if ch < '0' || ch > '9' {
			return PaletteColorEvent{}, false
		}
		n = n*10 + int(ch-'0')
	}
	if n > 255 {
		return PaletteColorEvent{}, false
	}
	r, g, b, ok := parseRGBSpec(spec)
	if !ok {
		return PaletteColorEvent{}, false
	}
	return PaletteColorEvent{Index: uint8(n), R: r, G: g, B: b}, true // #nosec G115 -- n <= 255
}

func parseBackgroundReply(s string) (BackgroundColorEvent, bool) {
	spec, ok := strings.CutPrefix(s, "11;")
	if !ok {
		return BackgroundColorEvent{}, false
	}
	r, g, b, ok := parseRGBSpec(spec)
	if !ok {
		return BackgroundColorEvent{}, false
	}
	return BackgroundColorEvent{R: r, G: g, B: b}, true
}

// parseRGBSpec decodes "rgb:R/G/B" with 1 to 4 hex digits per channel.
func parseRGBSpec(spec string) (r, g, b uint8, ok bool) {
	spec, ok = strings.CutPrefix(spec, "rgb:")
	if !ok {
		return 0, 0, 0, false
	}
	parts := strings.Split(spec, "/")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var c [3]uint8
	for i, p := range parts {
		if len(p) < 1 || len(p) > 4 {
			return 0, 0, 0, false
		}
		v, max := 0, 0
		for _, ch := range p {
			d := hexDigit(ch)
			if d < 0 {
				return 0, 0, 0, false
			}
			v = v*16 + d
			max = max*16 + 15
		}
		c[i] = clamp8((v*255 + max/2) / max)
	}
	return c[0], c[1], c[2], true
}

func hexDigit(r rune) int {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0')
	case r >= 'a' && r <= 'f':
		return int(r-'a') + 10
	case r >= 'A' && r <= 'F':
		return int(r-'A') + 10
	}
	return -1
}

// clamp8 converts v to uint8, saturating at 0 and 255.
func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

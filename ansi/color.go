package ansi

import "strconv"

// Color renders itself as the SGR parameter list for foreground/background.
type Color interface {
	fgCode() string
	bgCode() string
	ulCode() string
}

// The SGR slots a Color can fill: foreground, background, underline.
const (
	slotFg = iota
	slotBg
	slotUl
)

// appendSGRColor appends c's SGR parameters for slot to dst. It switches on the
// concrete type instead of calling through the Color interface: a call
// through an interface makes the compiler assume dst may be retained, which
// forces Style.Render's stack buffer onto the heap. Color has unexported
// methods, so these three are the only implementations.
func appendSGRColor(dst []byte, c Color, slot int) []byte {
	switch v := c.(type) {
	case BasicColor:
		return v.appendSGR(dst, slot)
	case Color256:
		return v.appendSGR(dst, slot)
	case RGB:
		return v.appendSGR(dst, slot)
	}
	switch slot {
	case slotFg:
		return append(dst, c.fgCode()...)
	case slotBg:
		return append(dst, c.bgCode()...)
	}
	return append(dst, c.ulCode()...)
}

// extPrefix is the introducer of an extended (256 or RGB) colour for slot:
// "38;" foreground, "48;" background, "58;" underline.
func extPrefix(slot int) string {
	switch slot {
	case slotFg:
		return "38;"
	case slotBg:
		return "48;"
	}
	return "58;"
}

// BasicColor is one of the 16 standard ANSI colors.
type BasicColor uint8

// The 16 standard ANSI colours in SGR order: the eight normal colours
// (Black to White) followed by their bright variants.
const (
	Black BasicColor = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	BrightBlack
	BrightRed
	BrightGreen
	BrightYellow
	BrightBlue
	BrightMagenta
	BrightCyan
	BrightWhite
)

func (c BasicColor) appendSGR(dst []byte, slot int) []byte {
	switch slot {
	case slotFg:
		if c < 8 {
			return strconv.AppendInt(dst, 30+int64(c), 10)
		}
		return strconv.AppendInt(dst, 82+int64(c), 10) // 90 + (c-8)
	case slotBg:
		if c < 8 {
			return strconv.AppendInt(dst, 40+int64(c), 10)
		}
		return strconv.AppendInt(dst, 92+int64(c), 10) // 100 + (c-8)
	}
	return strconv.AppendInt(append(dst, "58;5;"...), int64(c), 10)
}

func (c BasicColor) fgCode() string {
	if c < 8 {
		return strconv.Itoa(30 + int(c))
	}
	return strconv.Itoa(82 + int(c)) // 90 + (c-8)
}

func (c BasicColor) bgCode() string {
	if c < 8 {
		return strconv.Itoa(40 + int(c))
	}
	return strconv.Itoa(92 + int(c)) // 100 + (c-8)
}

// ulCode uses the extended underline-color sequence's 256-color form (SGR
// 58;5;n): the basic 16 ANSI colors occupy indices 0-15 of that same
// palette, so this is exact, not an approximation.
func (c BasicColor) ulCode() string { return "58;5;" + strconv.Itoa(int(c)) }

// Color256 is an index into the 256-color xterm palette.
type Color256 uint8

func (c Color256) appendSGR(dst []byte, slot int) []byte {
	return strconv.AppendInt(append(append(dst, extPrefix(slot)...), "5;"...), int64(c), 10)
}

func (c Color256) fgCode() string { return "38;5;" + strconv.Itoa(int(c)) }
func (c Color256) bgCode() string { return "48;5;" + strconv.Itoa(int(c)) }
func (c Color256) ulCode() string { return "58;5;" + strconv.Itoa(int(c)) }

// RGB is a 24-bit truecolor value.
type RGB struct{ R, G, B uint8 }

func (c RGB) appendSGR(dst []byte, slot int) []byte {
	dst = append(append(dst, extPrefix(slot)...), "2;"...)
	dst = strconv.AppendInt(dst, int64(c.R), 10)
	dst = strconv.AppendInt(append(dst, ';'), int64(c.G), 10)
	return strconv.AppendInt(append(dst, ';'), int64(c.B), 10)
}

func (c RGB) fgCode() string {
	return "38;2;" + strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
}
func (c RGB) bgCode() string {
	return "48;2;" + strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
}
func (c RGB) ulCode() string {
	return "58;2;" + strconv.Itoa(int(c.R)) + ";" + strconv.Itoa(int(c.G)) + ";" + strconv.Itoa(int(c.B))
}

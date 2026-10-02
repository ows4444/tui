package cellbuf

// Attr is a set of text attributes, as bit flags.
type Attr uint16

// The text attributes of a Style.
const (
	// AttrBold is SGR 1.
	AttrBold Attr = 1 << iota
	// AttrFaint is SGR 2.
	AttrFaint
	// AttrItalic is SGR 3.
	AttrItalic
	// AttrUnderline is SGR 4.
	AttrUnderline
	// AttrBlink is SGR 5.
	AttrBlink
	// AttrReverse is SGR 7.
	AttrReverse
	// AttrConceal is SGR 8.
	AttrConceal
	// AttrStrike is SGR 9.
	AttrStrike
)

// Color is a terminal colour in the form a view used (basic, bright, 256 or
// RGB), so the terminal receives the same code family. The zero Color is the
// terminal's default colour. Colors are comparable.
type Color uint32

const (
	kindBasic  = 1
	kindBright = 2
	kind256    = 3
	kindRGB    = 4
)

// Basic returns the basic colour n (0-7: black, red, green, yellow, blue,
// magenta, cyan, white); n is taken modulo 8.
func Basic(n int) Color { return kindBasic<<24 | Color(n&7) }

// Bright returns the bright colour n (0-7); n is taken modulo 8.
func Bright(n int) Color { return kindBright<<24 | Color(n&7) }

// Indexed returns the 256-colour palette entry n; n is taken modulo 256.
func Indexed(n int) Color { return kind256<<24 | Color(n&0xff) }

// RGB returns the 24-bit colour with the given components.
func RGB(r, g, b uint8) Color {
	return kindRGB<<24 | Color(r)<<16 | Color(g)<<8 | Color(b)
}

// Style is the look of a cell: attributes, colours and an optional hyperlink.
// The zero Style is the terminal default. Styles are comparable.
type Style struct {
	// Attrs are the text attributes.
	Attrs Attr
	// FG and BG are the foreground and background colours; zero is default.
	FG, BG Color
	// UnderlineStyle selects an extended underline shape (SGR 4:n): 2 double,
	// 3 curly, 4 dotted, 5 dashed. 0 is the plain underline, if AttrUnderline
	// is set.
	UnderlineStyle uint8
	// Link is the OSC 8 hyperlink target URI, "" for none. Control characters
	// are removed from it when the Style is added to a Buffer. A parsed
	// hyperlink's id and other parameters are not kept.
	Link string
}

// StyleID names a Style in one Buffer's style table (and in every Sub view of
// it). ID 0 is always the zero Style. IDs are not portable between Buffers.
type StyleID uint16

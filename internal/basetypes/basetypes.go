// Package basetypes holds the small value types that theme and layout share
// (the Border), so theme can stay a dependency leaf that does not import
// layout. layout and theme re-export Border as a type alias; the public API is
// unchanged.
package basetypes

// Border is the set of characters drawn around a Box. layout.Border is an
// alias of it.
type Border struct {
	Top, Bottom, Left, Right                   string
	TopLeft, TopRight, BottomLeft, BottomRight string
}

// The border styles theme's presets use. They equal the layout package's
// NormalBorder, RoundedBorder and ASCIIBorder (a theme test asserts it);
// they are written with escapes so this file stays ASCII.
var (
	NormalBorder  = Border{"─", "─", "│", "│", "┌", "┐", "└", "┘"}
	RoundedBorder = Border{"─", "─", "│", "│", "╭", "╮", "╰", "╯"}
	ASCIIBorder   = Border{"-", "-", "|", "|", "+", "+", "+", "+"}
)

package theme

import (
	"os"
	"reflect"
	"strings"

	"github.com/ows4444/tui/internal/basetypes"
)

// Glyphs are the non-letter symbols widgets draw: progress fill, tree and
// accordion markers, spinner frames, status dots, check and cross marks,
// bullets, rules, cursors, heat and spark ramps, the password mask and a few
// typographic marks. Two sets exist, UnicodeGlyphSet (the default) and
// ASCIIGlyphSet for terminals, fonts or locales without box and block
// characters. A Glyphs value is comparable, so Theme stays comparable.
//
// Empty fields fall back to UnicodeGlyphSet, so the zero Glyphs, and every Theme
// that never sets one, renders exactly as before. Every ASCII glyph takes the
// same columns as its Unicode counterpart, except Arrow ("->" against "→"),
// so a widget's layout does not shift.
type Glyphs struct {
	// BarFull and BarEmpty are the filled and unfilled cell of a progress bar.
	BarFull, BarEmpty string
	// Collapsed and Expanded mark a closed and an open tree node or section.
	Collapsed, Expanded string
	// Spinner holds the spinner frames, one rune each.
	Spinner string

	// Dot and DotEmpty are a filled and an empty status or page marker.
	Dot, DotEmpty string
	// Check and Cross mark a done or failed step; Warning and Info are the
	// other two alert icons.
	Check, Cross, Warning, Info string
	// Bullet and BulletNested mark list items, at the top level and nested.
	Bullet, BulletNested string
	// Arrow separates steps.
	Arrow string
	// RuleH and RuleV are a horizontal rule and a vertical gutter or quote bar.
	RuleH, RuleV string
	// Cursor is a thin cursor and CursorBlock a full block cursor.
	Cursor, CursorBlock string
	// Shades holds four runes from light to dark (heat maps); Sparks holds
	// eight runes from low to high (sparklines).
	Shades, Sparks string
	// Mask replaces each character of a password.
	Mask string
	// Ellipsis marks truncated text and takes exactly one column.
	Ellipsis string
	// Dash and Middot are the separators in "name - description" and
	// "a . b" text.
	Dash, Middot string
}

// unicodeGlyphs backs UnicodeGlyphSet.
var unicodeGlyphs = Glyphs{
	BarFull: "█", BarEmpty: "░",
	Collapsed: "▸", Expanded: "▾",
	Spinner: "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏",
	Dot:     "●", DotEmpty: "○",
	Check: "✓", Cross: "✗", Warning: "⚠", Info: "ℹ",
	Bullet: "•", BulletNested: "◦",
	Arrow: "→",
	RuleH: "─", RuleV: "│",
	Cursor: "▌", CursorBlock: "█",
	Shades: "░▒▓█", Sparks: "▁▂▃▄▅▆▇█",
	Mask:     "•",
	Ellipsis: "…",
	Dash:     "—", Middot: "·",
}

// asciiGlyphs backs ASCIIGlyphSet; it uses only 7-bit ASCII.
var asciiGlyphs = Glyphs{
	BarFull: "#", BarEmpty: "-",
	Collapsed: ">", Expanded: "v",
	Spinner: `|/-\`,
	Dot:     "*", DotEmpty: "o",
	Check: "v", Cross: "x", Warning: "!", Info: "i",
	Bullet: "*", BulletNested: "-",
	Arrow: "->",
	RuleH: "-", RuleV: "|",
	Cursor: "|", CursorBlock: "#",
	Shades: ".:+#", Sparks: "_.-:=+*#",
	Mask:     "*",
	Ellipsis: "~",
	Dash:     "-", Middot: ".",
}

// NerdGlyphSet is UnicodeGlyphSet with Nerd Font (Font Awesome range) icons for
// the status and marker glyphs that have one; the other fields keep their
// Unicode value. The icons are private-use code points, each one column, so
// use it only when the terminal font is a Nerd Font (see DetectGlyphsEnv).
var nerdGlyphs = Glyphs{
	BarFull: unicodeGlyphs.BarFull, BarEmpty: unicodeGlyphs.BarEmpty,
	Collapsed: "\uf105", Expanded: "\uf107", // nf-fa-angle-right, nf-fa-angle-down
	Spinner: unicodeGlyphs.Spinner,
	Dot:     "\uf111", DotEmpty: "\uf10c", // nf-fa-circle, nf-fa-circle_o
	Check: "\uf00c", Cross: "\uf00d", Warning: "\uf071", Info: "\uf05a", // check, times, warning, info_circle
	Bullet: unicodeGlyphs.Bullet, BulletNested: unicodeGlyphs.BulletNested,
	Arrow: "\uf061", // nf-fa-arrow_right
	RuleH: unicodeGlyphs.RuleH, RuleV: unicodeGlyphs.RuleV,
	Cursor: unicodeGlyphs.Cursor, CursorBlock: unicodeGlyphs.CursorBlock,
	Shades: unicodeGlyphs.Shades, Sparks: unicodeGlyphs.Sparks,
	Mask:     unicodeGlyphs.Mask,
	Ellipsis: unicodeGlyphs.Ellipsis,
	Dash:     unicodeGlyphs.Dash, Middot: unicodeGlyphs.Middot,
}

// Resolved returns g with every empty field taken from UnicodeGlyphSet.
func (g Glyphs) Resolved() Glyphs {
	pick := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}
	u := unicodeGlyphs
	return Glyphs{
		BarFull:      pick(g.BarFull, u.BarFull),
		BarEmpty:     pick(g.BarEmpty, u.BarEmpty),
		Collapsed:    pick(g.Collapsed, u.Collapsed),
		Expanded:     pick(g.Expanded, u.Expanded),
		Spinner:      pick(g.Spinner, u.Spinner),
		Dot:          pick(g.Dot, u.Dot),
		DotEmpty:     pick(g.DotEmpty, u.DotEmpty),
		Check:        pick(g.Check, u.Check),
		Cross:        pick(g.Cross, u.Cross),
		Warning:      pick(g.Warning, u.Warning),
		Info:         pick(g.Info, u.Info),
		Bullet:       pick(g.Bullet, u.Bullet),
		BulletNested: pick(g.BulletNested, u.BulletNested),
		Arrow:        pick(g.Arrow, u.Arrow),
		RuleH:        pick(g.RuleH, u.RuleH),
		RuleV:        pick(g.RuleV, u.RuleV),
		Cursor:       pick(g.Cursor, u.Cursor),
		CursorBlock:  pick(g.CursorBlock, u.CursorBlock),
		Shades:       pick(g.Shades, u.Shades),
		Sparks:       pick(g.Sparks, u.Sparks),
		Mask:         pick(g.Mask, u.Mask),
		Ellipsis:     pick(g.Ellipsis, u.Ellipsis),
		Dash:         pick(g.Dash, u.Dash),
		Middot:       pick(g.Middot, u.Middot),
	}
}

// ASCII reports whether every glyph in g, after empty fields are filled from
// UnicodeGlyphSet, is 7-bit ASCII. Widgets that draw with characters computed
// at run time (braille dot charts) use it to switch to a plain density ramp.
func (g Glyphs) ASCII() bool {
	v := reflect.ValueOf(g.Resolved())
	for i := 0; i < v.NumField(); i++ {
		for _, r := range v.Field(i).String() {
			if r >= 0x80 {
				return false
			}
		}
	}
	return true
}

// SpinnerFrames returns the spinner frames, one per rune of g.Spinner.
func (g Glyphs) SpinnerFrames() []string {
	frames := make([]string, 0, len(g.Spinner))
	for _, r := range g.Spinner {
		frames = append(frames, string(r))
	}
	return frames
}

// GlyphSet returns the theme's glyphs with any empty field filled from
// UnicodeGlyphSet. Widgets call it; a theme that never set Glyphs gets the
// Unicode set.
func (t Theme) GlyphSet() Glyphs { return t.Glyphs.Resolved() }

// ASCII returns a copy of t for ASCII-only terminals: ASCIIGlyphSet and
// layout.ASCIIBorder. Colours are unchanged; pair with ForProfile for a
// colourless terminal.
func (t Theme) ASCII() Theme {
	t.Glyphs = asciiGlyphs
	t.Border = basetypes.ASCIIBorder
	return t
}

// DetectGlyphs chooses ASCIIGlyphSet when the environment says the terminal
// cannot be trusted with Unicode, NerdGlyphSet when TUI_NERD_FONT=1, and
// UnicodeGlyphSet otherwise; see
// DetectGlyphsEnv. Typical use:
//
//	th := theme.Dark
//	th.Glyphs = theme.DetectGlyphs()
func DetectGlyphs() Glyphs { return DetectGlyphsEnv(os.Getenv) }

// DetectGlyphsEnv is DetectGlyphs with an injectable getenv. It returns
// ASCIIGlyphSet when TERM is "dumb", or when the locale (the first non-empty of
// LC_ALL, LC_CTYPE, LANG) is "C" or "POSIX" or names a charset other than
// UTF-8; those ASCII checks always win. Otherwise it returns NerdGlyphSet when
// TUI_NERD_FONT is exactly "1" (an explicit opt-in, valid with or without a
// locale), and UnicodeGlyphSet. With no locale set at all it cannot tell and
// falls through to the Nerd or Unicode choice.
func DetectGlyphsEnv(getenv func(string) string) Glyphs {
	if g := detectBase(getenv); g != unicodeGlyphs {
		return g
	}
	if getenv("TUI_NERD_FONT") == "1" {
		return nerdGlyphs
	}
	return unicodeGlyphs
}

func detectBase(getenv func(string) string) Glyphs {
	if strings.EqualFold(getenv("TERM"), "dumb") {
		return asciiGlyphs
	}
	for _, name := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		loc := getenv(name)
		if loc == "" {
			continue
		}
		if isUTF8Locale(loc) {
			return unicodeGlyphs
		}
		return asciiGlyphs
	}
	return unicodeGlyphs
}

// isUTF8Locale reports whether a locale name such as "en_US.UTF-8" selects
// UTF-8. "C", "POSIX" and any other charset do not; "C.UTF-8" does.
func isUTF8Locale(loc string) bool {
	l := strings.ToLower(loc)
	return strings.Contains(l, "utf-8") || strings.Contains(l, "utf8")
}

// UnicodeGlyphSet returns the default glyph set. It returns a copy, so a
// caller cannot change what every widget draws.
func UnicodeGlyphSet() Glyphs { return unicodeGlyphs }

// ASCIIGlyphSet returns the glyph set that uses only 7-bit ASCII, as a copy.
func ASCIIGlyphSet() Glyphs { return asciiGlyphs }

// NerdGlyphSet returns UnicodeGlyphSet with Nerd Font icons for the status
// and marker glyphs that have one, as a copy. Use it only when the terminal
// font is a Nerd Font (see DetectGlyphsEnv).
func NerdGlyphSet() Glyphs { return nerdGlyphs }

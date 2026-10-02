// Package theme is a Go translation of InkUI's theming model
// (https://inkui-lib.vercel.app/docs/getting-started/theming): a small
// struct of semantic colors plus a border style, with Dark (the default)
// and Light built-in values.
//
// InkUI's theme reaches components via React context (a ThemeProvider/
// useTheme pair) so nothing needs it passed explicitly. Go has no
// equivalent ambient-value mechanism, and the alternative — a mutable
// package-level "current theme" — is a footgun the moment more than one
// Program runs in a process, and would make widget tests depend on global
// state instead of their arguments. So themed functions in this module
// (widgets.Badge, widgets.StatusIndicator, ...) take a Theme argument
// explicitly, the same way they already take an ansi.Style or similar.
package theme

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/basetypes"
)

// Border is the set of characters drawn around a Box. layout.Border is the
// same type, so a Theme's Border can be given to layout.Box.Border directly.
type Border = basetypes.Border

// Theme is the Go shape of InkUI's InkUITheme: 12 semantic colors plus a
// border style. BorderColor is the color token named "border" in InkUI's
// theme (the color to draw border lines in); Border is InkUI's separate
// 'single'/'double'/'rounded'/'bold'/'ascii' style token, represented here
// with the layout.Border values that already existed before theming did.
type Theme struct {
	Primary     ansi.Color
	Secondary   ansi.Color
	Success     ansi.Color
	Warning     ansi.Color
	Error       ansi.Color
	Info        ansi.Color
	Muted       ansi.Color
	Text        ansi.Color
	TextInverse ansi.Color
	BorderColor ansi.Color
	// Focus is the color used to highlight the focused control; consumed
	// by widgets.Checkbox and widgets.Toggle.
	Focus     ansi.Color
	Selection ansi.Color
	Border    Border
	// Background is the terminal background the theme is designed for; Check
	// measures text roles against it. Unset (nil) falls back to TextInverse.
	Background ansi.Color
	// Surface is the fill of panels and cards, one step above Background.
	Surface ansi.Color
	// Overlay is the fill of floating layers (menus, dialogs, toasts), one step
	// above Surface.
	Overlay ansi.Color
	// Components are per-widget colour overrides; set them with WithTokens and
	// read them with TokensFor or ForComponent.
	Components *ComponentTokens
	// Glyphs are the symbols widgets draw (progress fill, tree markers,
	// spinner frames). The zero value means UnicodeGlyphSet; see Glyphs and
	// Theme.ASCII.
	Glyphs Glyphs
	// Spacing is the spacing scale. Zero fields mean the defaults; read it
	// through Theme.ResolvedSpacing.
	Spacing SpacingScale
	// States are the styles for focus, hover, disabled and selected. Empty
	// styles fall back to the colour roles; read them through
	// Theme.ResolvedStates.
	States States
	// Typography are the heading and inline text styles. Empty styles fall
	// back to the colour roles; read them through Theme.ResolvedTypography.
	Typography Typography
}

// SpacingScale is a comparable set of spacing tokens, in cells. A zero
// field means "use the default" (see Theme.Spacing), so an unset scale
// leaves widget output unchanged.
type SpacingScale struct {
	XS, S, M, L int
}

// Default spacing values: S is today's box padding (1).
var defaultSpacing = SpacingScale{XS: 1, S: 1, M: 2, L: 3}

// ResolvedSpacing returns t's spacing with zero fields replaced by defaults.
func (t Theme) ResolvedSpacing() SpacingScale {
	s := t.Spacing
	if s.XS == 0 {
		s.XS = defaultSpacing.XS
	}
	if s.S == 0 {
		s.S = defaultSpacing.S
	}
	if s.M == 0 {
		s.M = defaultSpacing.M
	}
	if s.L == 0 {
		s.L = defaultSpacing.L
	}
	return s
}

// Dark is the default theme: bright colors (for visibility against a dark
// terminal background) and single-line borders, matching InkUI's
// described dark theme (cyan primary, white text, single-line borders).
var dark = Theme{
	Primary:     ansi.BrightCyan,
	Secondary:   ansi.BrightMagenta,
	Success:     ansi.BrightGreen,
	Warning:     ansi.BrightYellow,
	Error:       ansi.BrightRed,
	Info:        ansi.BrightBlue,
	Muted:       ansi.BrightBlack,
	Text:        ansi.White,
	TextInverse: ansi.Black,
	BorderColor: ansi.BrightBlack,
	Focus:       ansi.BrightCyan,
	Selection:   ansi.BrightBlue,
	Border:      basetypes.NormalBorder,
}

// Plain returns a copy of t with Border cleared to the zero value, so any
// widget that draws a border via t.Border (layout.Box.Border checks
// exactly this — the zero value means "no border") renders flat,
// undecorated text instead of box-drawing characters. Every other field
// — colors included — is unchanged. Pairs with (*tui.Program).ReducedMotion:
// an app honoring reduced motion typically wants to drop decorative
// framing as well as animation, not just one or the other, and Plain
// works with any existing Theme value (Dark, a preset, ...) rather than
// requiring a separate "plain" theme per palette.
func (t Theme) Plain() Theme {
	t.Border = Border{}
	return t
}

// Light uses non-bright colors (for visibility against a light terminal
// background) and rounded borders, matching InkUI's described light theme
// (blue primary, black text, rounded borders).
var light = Theme{
	Primary:   ansi.Blue,
	Secondary: ansi.Magenta,
	Success:   ansi.Green,
	Warning:   ansi.Yellow,
	Error:     ansi.Red,
	Info:      ansi.Cyan,
	// BrightBlack (not White, which is near-invisible on a light
	// background) — the same gray-toned convention Dark.Muted uses, since
	// ANSI's BrightBlack is a mid-gray that reads against both light and
	// dark backgrounds.
	Muted:       ansi.BrightBlack,
	Text:        ansi.Black,
	TextInverse: ansi.BrightWhite,
	BorderColor: ansi.Black,
	Focus:       ansi.Cyan,
	Selection:   ansi.Blue,
	Border:      basetypes.RoundedBorder,
}

// ForProfile returns a copy of t with every color downgraded to what p can
// display (ansi.Downgrade), so a truecolor preset degrades gracefully on a
// 256- or 16-color terminal. Under ansi.NoColor all colors become nil
// (unset) while Border is kept. Pair with ansi.DetectColorProfile:
//
//	th := theme.DarkTheme().ForProfile(ansi.DetectColorProfile())
func (t Theme) ForProfile(p ansi.Profile) Theme {
	for _, c := range []*ansi.Color{
		&t.Primary, &t.Secondary, &t.Success, &t.Warning, &t.Error, &t.Info,
		&t.Muted, &t.Text, &t.TextInverse, &t.BorderColor, &t.Focus, &t.Selection,
		&t.Background, &t.Surface, &t.Overlay,
	} {
		*c = ansi.Downgrade(*c, p)
	}
	return t
}

// ForBackground returns Light for a light terminal background and Dark for
// a dark one, judged by the background's perceived luminance (Rec. 709).
// Feed it the color from an OSC 11 reply (tui.BackgroundColorEvent).
func ForBackground(bg ansi.RGB) Theme {
	if isLight(bg) {
		return light
	}
	return dark
}

// isLight reports whether bg's perceived luminance (Rec. 709) is above the
// midpoint.
func isLight(bg ansi.RGB) bool {
	return 0.2126*float64(bg.R)+0.7152*float64(bg.G)+0.0722*float64(bg.B) > 127.5
}

// Detect is the receiving end of tui.WithBackgroundDetection: call it from
// Update with each Msg and it turns the terminal's answer into a theme.
// A tui.BackgroundColorEvent returns ForBackground of its colour and true. A
// tui.BackgroundUnknownMsg (no answer before the timeout) returns fallback
// and true, so the app can stop waiting. Any other Msg returns fallback and
// false.
//
//	case tui.BackgroundColorEvent, tui.BackgroundUnknownMsg:
//		if th, ok := theme.Detect(msg, theme.DarkTheme()); ok {
//			m.theme = th
//		}
func Detect(msg any, fallback Theme) (Theme, bool) {
	switch m := msg.(type) {
	case backgroundColor:
		r, g, b := m.BackgroundRGB()
		return ForBackground(ansi.RGB{R: r, G: g, B: b}), true
	case backgroundUnknown:
		return fallback, true
	}
	return fallback, false
}

// backgroundColor and backgroundUnknown are the two shapes Detect recognises,
// implemented by input.BackgroundColorEvent and input.BackgroundUnknownMsg.
// Matching on methods keeps theme a leaf: it needs no import of input.
type (
	backgroundColor   interface{ BackgroundRGB() (r, g, b uint8) }
	backgroundUnknown interface{ BackgroundUnknown() }
)

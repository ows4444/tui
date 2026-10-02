package theme

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/basetypes"
)

// This file adds a curated subset of well-known, publicly documented
// terminal/editor color-theme presets as additional Theme values, following
// Dark/Light's complete-struct convention in theme.go. Named ANSI-16 colors
// (ansi.BrightCyan etc.) can't represent most of these palettes' specific
// hues, so these presets use ansi.RGB with each theme's real, documented
// hex values instead. Sources (canonical palette hex codes, checked against
// the model's training knowledge of each project's own published spec):
//
//   - Dracula:        https://draculatheme.com/contribute (official spec)
//   - Nord:            https://www.nordtheme.com/docs/colors-and-palettes
//   - Gruvbox:         https://github.com/morhetz/gruvbox (dark, hard/medium contrast)
//   - Tokyo Night:     https://github.com/enkia/tokyo-night-vscode-theme (Night variant)
//   - Monokai:         https://monokai.pro / the original Monokai classic palette
//   - Solarized:       https://ethanschoonover.com/solarized/ (base0-3, accent colors)
//   - Catppuccin:      https://github.com/catppuccin/catppuccin (Mocha variant)
//   - One Dark:        https://github.com/atom/atom/tree/master/packages/one-dark-ui
//   - Night Owl:       https://github.com/sdras/night-owl-vscode-theme
//   - Rosé Pine:       https://rosepinetheme.com (main variant)
//   - Everforest:      https://github.com/sainnhe/everforest (dark, medium contrast)
//   - GitHub Dark:     https://primer.style/primitives (Primer's dark color scales)
//
// This remains a curated subset (per docs/audits/PLAN.md's own "Phase H" scoping), not
// an attempt at parity with termcn's ~40 bundled themes — each addition
// here is a widely-used, well-documented palette, not a mechanical import
// of the full list.

// Dracula is the Dracula theme (https://draculatheme.com), a dark theme
// with a signature purple/pink/cyan palette. Single-line borders match its
// generally sharp, dark-background aesthetic.
var dracula = Theme{
	Primary:     ansi.RGB{R: 0xbd, G: 0x93, B: 0xf9}, // purple
	Secondary:   ansi.RGB{R: 0xff, G: 0x79, B: 0xc6}, // pink
	Success:     ansi.RGB{R: 0x50, G: 0xfa, B: 0x7b}, // green
	Warning:     ansi.RGB{R: 0xf1, G: 0xfa, B: 0x8c}, // yellow
	Error:       ansi.RGB{R: 0xff, G: 0x55, B: 0x55}, // red
	Info:        ansi.RGB{R: 0x8b, G: 0xe9, B: 0xfd}, // cyan
	Muted:       ansi.RGB{R: 0x62, G: 0x72, B: 0xa4}, // comment
	Text:        ansi.RGB{R: 0xf8, G: 0xf8, B: 0xf2}, // foreground
	TextInverse: ansi.RGB{R: 0x28, G: 0x2a, B: 0x36}, // background
	BorderColor: ansi.RGB{R: 0x44, G: 0x47, B: 0x5a}, // current line
	Focus:       ansi.RGB{R: 0xbd, G: 0x93, B: 0xf9}, // purple
	Selection:   ansi.RGB{R: 0x44, G: 0x47, B: 0x5a}, // selection/current line
	Border:      basetypes.NormalBorder,
}

// Nord is the Nord theme (https://www.nordtheme.com), an arctic, muted
// blue-toned palette. Rounded borders match its soft, understated look.
var nord = Theme{
	Primary:     ansi.RGB{R: 0x88, G: 0xc0, B: 0xd0}, // nord8 frost (cyan)
	Secondary:   ansi.RGB{R: 0x81, G: 0xa1, B: 0xc1}, // nord9 frost (blue)
	Success:     ansi.RGB{R: 0xa3, G: 0xbe, B: 0x8c}, // nord14 aurora (green)
	Warning:     ansi.RGB{R: 0xeb, G: 0xcb, B: 0x8b}, // nord13 aurora (yellow)
	Error:       ansi.RGB{R: 0xbf, G: 0x61, B: 0x6a}, // nord11 aurora (red)
	Info:        ansi.RGB{R: 0x5e, G: 0x81, B: 0xac}, // nord10 frost (blue)
	Muted:       ansi.RGB{R: 0x4c, G: 0x56, B: 0x6a}, // nord3 polar night
	Text:        ansi.RGB{R: 0xe5, G: 0xe9, B: 0xf0}, // nord5 snow storm
	TextInverse: ansi.RGB{R: 0x2e, G: 0x34, B: 0x40}, // nord0 polar night
	BorderColor: ansi.RGB{R: 0x43, G: 0x4c, B: 0x5e}, // nord2 polar night
	Focus:       ansi.RGB{R: 0x88, G: 0xc0, B: 0xd0}, // nord8 frost
	Selection:   ansi.RGB{R: 0x43, G: 0x4c, B: 0x5e}, // nord2 polar night
	Border:      basetypes.RoundedBorder,
}

// Gruvbox is the Gruvbox dark theme (https://github.com/morhetz/gruvbox),
// a warm, retro-groove palette with earthy, low-contrast tones. Normal
// borders match its terminal-native, boxy feel.
var gruvbox = Theme{
	Primary:     ansi.RGB{R: 0x83, G: 0xa5, B: 0x98}, // bright blue
	Secondary:   ansi.RGB{R: 0xd3, G: 0x86, B: 0x9b}, // bright purple
	Success:     ansi.RGB{R: 0xb8, G: 0xbb, B: 0x26}, // bright green
	Warning:     ansi.RGB{R: 0xfa, G: 0xbd, B: 0x2f}, // bright yellow
	Error:       ansi.RGB{R: 0xfb, G: 0x49, B: 0x34}, // bright red
	Info:        ansi.RGB{R: 0x8e, G: 0xc0, B: 0x7c}, // bright aqua
	Muted:       ansi.RGB{R: 0x92, G: 0x83, B: 0x74}, // gray
	Text:        ansi.RGB{R: 0xeb, G: 0xdb, B: 0xb2}, // fg1
	TextInverse: ansi.RGB{R: 0x28, G: 0x28, B: 0x28}, // bg0
	BorderColor: ansi.RGB{R: 0x50, G: 0x49, B: 0x45}, // bg2
	Focus:       ansi.RGB{R: 0xfe, G: 0x80, B: 0x19}, // bright orange
	Selection:   ansi.RGB{R: 0x3c, G: 0x38, B: 0x36}, // bg1
	Border:      basetypes.NormalBorder,
}

// TokyoNight is the Tokyo Night theme's "Night" variant
// (https://github.com/enkia/tokyo-night-vscode-theme), a clean, cool-blue
// dark palette modeled on Tokyo at night. Normal borders match its crisp,
// editor-native aesthetic.
var tokyoNight = Theme{
	Primary:     ansi.RGB{R: 0x7a, G: 0xa2, B: 0xf7}, // blue
	Secondary:   ansi.RGB{R: 0xbb, G: 0x9a, B: 0xf7}, // purple
	Success:     ansi.RGB{R: 0x9e, G: 0xce, B: 0x6a}, // green
	Warning:     ansi.RGB{R: 0xe0, G: 0xaf, B: 0x68}, // yellow/orange
	Error:       ansi.RGB{R: 0xf7, G: 0x76, B: 0x8e}, // red
	Info:        ansi.RGB{R: 0x7d, G: 0xcf, B: 0xff}, // cyan
	Muted:       ansi.RGB{R: 0x56, G: 0x5f, B: 0x89}, // comment
	Text:        ansi.RGB{R: 0xc0, G: 0xca, B: 0xf5}, // foreground
	TextInverse: ansi.RGB{R: 0x1a, G: 0x1b, B: 0x26}, // background
	BorderColor: ansi.RGB{R: 0x29, G: 0x2e, B: 0x42}, // bg_highlight
	Focus:       ansi.RGB{R: 0x7a, G: 0xa2, B: 0xf7}, // blue
	Selection:   ansi.RGB{R: 0x28, G: 0x3b, B: 0x4d}, // selection-ish highlight
	Border:      basetypes.NormalBorder,
}

// Monokai is the classic Monokai theme, a dark theme famous for its
// high-contrast pink/green/orange palette. Normal borders match its
// bold, sharp-edged terminal look.
var monokai = Theme{
	Primary:     ansi.RGB{R: 0x66, G: 0xd9, B: 0xef}, // blue/cyan
	Secondary:   ansi.RGB{R: 0xae, G: 0x81, B: 0xff}, // purple
	Success:     ansi.RGB{R: 0xa6, G: 0xe2, B: 0x2e}, // green
	Warning:     ansi.RGB{R: 0xe6, G: 0xdb, B: 0x74}, // yellow
	Error:       ansi.RGB{R: 0xf9, G: 0x26, B: 0x72}, // pink/red
	Info:        ansi.RGB{R: 0x66, G: 0xd9, B: 0xef}, // blue/cyan
	Muted:       ansi.RGB{R: 0x75, G: 0x71, B: 0x5e}, // comment gray
	Text:        ansi.RGB{R: 0xf8, G: 0xf8, B: 0xf2}, // foreground
	TextInverse: ansi.RGB{R: 0x27, G: 0x28, B: 0x22}, // background
	BorderColor: ansi.RGB{R: 0x49, G: 0x48, B: 0x3e}, // line highlight
	Focus:       ansi.RGB{R: 0xfd, G: 0x97, B: 0x1f}, // orange
	Selection:   ansi.RGB{R: 0x49, G: 0x48, B: 0x3e}, // selection
	Border:      basetypes.NormalBorder,
}

// SolarizedDark is the Solarized theme (https://ethanschoonover.com/solarized/)
// in its dark variant: base03 background, base0 foreground, and the shared
// Solarized accent palette (yellow/orange/red/magenta/violet/blue/cyan/green).
var solarizedDark = Theme{
	Primary:     ansi.RGB{R: 0x26, G: 0x8b, B: 0xd2}, // blue
	Secondary:   ansi.RGB{R: 0x6c, G: 0x71, B: 0xc4}, // violet
	Success:     ansi.RGB{R: 0x85, G: 0x99, B: 0x00}, // green
	Warning:     ansi.RGB{R: 0xb5, G: 0x89, B: 0x00}, // yellow
	Error:       ansi.RGB{R: 0xdc, G: 0x32, B: 0x2f}, // red
	Info:        ansi.RGB{R: 0x2a, G: 0xa1, B: 0x98}, // cyan
	Muted:       ansi.RGB{R: 0x58, G: 0x6e, B: 0x75}, // base01
	Text:        ansi.RGB{R: 0x83, G: 0x94, B: 0x96}, // base0
	TextInverse: ansi.RGB{R: 0x00, G: 0x2b, B: 0x36}, // base03
	BorderColor: ansi.RGB{R: 0x07, G: 0x36, B: 0x42}, // base02
	Focus:       ansi.RGB{R: 0x26, G: 0x8b, B: 0xd2}, // blue
	Selection:   ansi.RGB{R: 0x07, G: 0x36, B: 0x42}, // base02
	Border:      basetypes.NormalBorder,
}

// SolarizedLight is the Solarized theme's light variant: base3 background,
// base00 foreground, and the same shared Solarized accent palette as
// SolarizedDark. Rounded borders match Light's light-background convention.
var solarizedLight = Theme{
	Primary:   ansi.RGB{R: 0x26, G: 0x8b, B: 0xd2}, // blue
	Secondary: ansi.RGB{R: 0x6c, G: 0x71, B: 0xc4}, // violet
	Success:   ansi.RGB{R: 0x85, G: 0x99, B: 0x00}, // green
	Warning:   ansi.RGB{R: 0xb5, G: 0x89, B: 0x00}, // yellow
	Error:     ansi.RGB{R: 0xdc, G: 0x32, B: 0x2f}, // red
	Info:      ansi.RGB{R: 0x2a, G: 0xa1, B: 0x98}, // cyan
	Muted:     ansi.RGB{R: 0x93, G: 0xa1, B: 0xa1}, // base1
	// base01, not the spec's own "primary content" base00: base00 on base3
	// measures ~4.1:1 (WCAG relative luminance), just under the 4.5:1 AA
	// threshold for normal text — a known, often-criticized trait of
	// Solarized's light variant. base01 keeps the same hue family while
	// clearing AA; several popular Solarized-light forks make this same
	// swap for readability.
	Text:        ansi.RGB{R: 0x58, G: 0x6e, B: 0x75}, // base01
	TextInverse: ansi.RGB{R: 0xfd, G: 0xf6, B: 0xe3}, // base3
	BorderColor: ansi.RGB{R: 0xee, G: 0xe8, B: 0xd5}, // base2
	Focus:       ansi.RGB{R: 0x26, G: 0x8b, B: 0xd2}, // blue
	Selection:   ansi.RGB{R: 0xee, G: 0xe8, B: 0xd5}, // base2
	Border:      basetypes.RoundedBorder,
}

// Catppuccin is the Catppuccin theme's Mocha variant
// (https://github.com/catppuccin/catppuccin), a soft, low-contrast pastel
// palette. Rounded borders match its gentle, community-favorite aesthetic.
var catppuccin = Theme{
	Primary:     ansi.RGB{R: 0xcb, G: 0xa6, B: 0xf7}, // mauve
	Secondary:   ansi.RGB{R: 0xf5, G: 0xc2, B: 0xe7}, // pink
	Success:     ansi.RGB{R: 0xa6, G: 0xe3, B: 0xa1}, // green
	Warning:     ansi.RGB{R: 0xf9, G: 0xe2, B: 0xaf}, // yellow
	Error:       ansi.RGB{R: 0xf3, G: 0x8b, B: 0xa8}, // red
	Info:        ansi.RGB{R: 0x89, G: 0xdc, B: 0xeb}, // sky
	Muted:       ansi.RGB{R: 0x7f, G: 0x84, B: 0x9c}, // overlay1
	Text:        ansi.RGB{R: 0xcd, G: 0xd6, B: 0xf4}, // text
	TextInverse: ansi.RGB{R: 0x1e, G: 0x1e, B: 0x2e}, // base
	BorderColor: ansi.RGB{R: 0x45, G: 0x47, B: 0x5a}, // surface1
	Focus:       ansi.RGB{R: 0xcb, G: 0xa6, B: 0xf7}, // mauve
	Selection:   ansi.RGB{R: 0x31, G: 0x32, B: 0x44}, // surface0
	Border:      basetypes.RoundedBorder,
}

// OneDark is Atom's One Dark theme
// (https://github.com/atom/atom/tree/master/packages/one-dark-ui), a
// balanced, editor-native dark palette. Normal borders match its crisp,
// no-frills look.
var oneDark = Theme{
	Primary:     ansi.RGB{R: 0x61, G: 0xaf, B: 0xef}, // blue
	Secondary:   ansi.RGB{R: 0xc6, G: 0x78, B: 0xdd}, // purple
	Success:     ansi.RGB{R: 0x98, G: 0xc3, B: 0x79}, // green
	Warning:     ansi.RGB{R: 0xe5, G: 0xc0, B: 0x7b}, // yellow
	Error:       ansi.RGB{R: 0xe0, G: 0x6c, B: 0x75}, // red
	Info:        ansi.RGB{R: 0x56, G: 0xb6, B: 0xc2}, // cyan
	Muted:       ansi.RGB{R: 0x5c, G: 0x63, B: 0x70}, // comment gray
	Text:        ansi.RGB{R: 0xab, G: 0xb2, B: 0xbf}, // foreground
	TextInverse: ansi.RGB{R: 0x28, G: 0x2c, B: 0x34}, // background
	BorderColor: ansi.RGB{R: 0x3e, G: 0x44, B: 0x51}, // gutter/selection
	Focus:       ansi.RGB{R: 0x61, G: 0xaf, B: 0xef}, // blue
	Selection:   ansi.RGB{R: 0x3e, G: 0x44, B: 0x51}, // selection
	Border:      basetypes.NormalBorder,
}

// NightOwl is the Night Owl theme
// (https://github.com/sdras/night-owl-vscode-theme), a deep-blue dark
// palette designed for low-light coding. Normal borders match its
// editor-native aesthetic.
var nightOwl = Theme{
	Primary:     ansi.RGB{R: 0x82, G: 0xaa, B: 0xff}, // blue
	Secondary:   ansi.RGB{R: 0xc7, G: 0x92, B: 0xea}, // purple
	Success:     ansi.RGB{R: 0xad, G: 0xdb, B: 0x67}, // green
	Warning:     ansi.RGB{R: 0xf7, G: 0x8c, B: 0x6c}, // orange
	Error:       ansi.RGB{R: 0xef, G: 0x53, B: 0x50}, // red
	Info:        ansi.RGB{R: 0x7f, G: 0xdb, B: 0xca}, // cyan
	Muted:       ansi.RGB{R: 0x63, G: 0x77, B: 0x77}, // comment
	Text:        ansi.RGB{R: 0xd6, G: 0xde, B: 0xeb}, // foreground
	TextInverse: ansi.RGB{R: 0x01, G: 0x16, B: 0x27}, // background
	BorderColor: ansi.RGB{R: 0x1d, G: 0x3b, B: 0x53}, // selection-ish highlight
	Focus:       ansi.RGB{R: 0x82, G: 0xaa, B: 0xff}, // blue
	Selection:   ansi.RGB{R: 0x1d, G: 0x3b, B: 0x53}, // selection-ish highlight
	Border:      basetypes.NormalBorder,
}

// RosePine is the Rosé Pine theme's main variant (https://rosepinetheme.com),
// a muted, low-saturation palette named for its rose/pine color pairing.
// Rosé Pine's accent set has no true green; Pine (a muted teal) stands in
// for Success, the theme's own closest analog. Rounded borders match its
// soft aesthetic.
var rosePine = Theme{
	Primary:     ansi.RGB{R: 0xc4, G: 0xa7, B: 0xe7}, // iris
	Secondary:   ansi.RGB{R: 0xeb, G: 0xbc, B: 0xba}, // rose
	Success:     ansi.RGB{R: 0x31, G: 0x74, B: 0x8f}, // pine
	Warning:     ansi.RGB{R: 0xf6, G: 0xc1, B: 0x77}, // gold
	Error:       ansi.RGB{R: 0xeb, G: 0x6f, B: 0x92}, // love
	Info:        ansi.RGB{R: 0x9c, G: 0xcf, B: 0xd8}, // foam
	Muted:       ansi.RGB{R: 0x6e, G: 0x6a, B: 0x86}, // muted
	Text:        ansi.RGB{R: 0xe0, G: 0xde, B: 0xf4}, // text
	TextInverse: ansi.RGB{R: 0x19, G: 0x17, B: 0x24}, // base
	BorderColor: ansi.RGB{R: 0x40, G: 0x3d, B: 0x52}, // highlight med
	Focus:       ansi.RGB{R: 0xc4, G: 0xa7, B: 0xe7}, // iris
	Selection:   ansi.RGB{R: 0x21, G: 0x20, B: 0x2e}, // highlight low
	Border:      basetypes.RoundedBorder,
}

// EverforestDark is the Everforest theme's dark, medium-contrast variant
// (https://github.com/sainnhe/everforest), an earthy, green-forward
// palette. Normal borders match its boxy, terminal-native look.
var everforestDark = Theme{
	Primary:     ansi.RGB{R: 0xa7, G: 0xc0, B: 0x80}, // green
	Secondary:   ansi.RGB{R: 0xd6, G: 0x99, B: 0xb6}, // purple
	Success:     ansi.RGB{R: 0xa7, G: 0xc0, B: 0x80}, // green
	Warning:     ansi.RGB{R: 0xdb, G: 0xbc, B: 0x7f}, // yellow
	Error:       ansi.RGB{R: 0xe6, G: 0x7e, B: 0x80}, // red
	Info:        ansi.RGB{R: 0x83, G: 0xc0, B: 0x92}, // aqua
	Muted:       ansi.RGB{R: 0x85, G: 0x92, B: 0x89}, // grey1
	Text:        ansi.RGB{R: 0xd3, G: 0xc6, B: 0xaa}, // fg
	TextInverse: ansi.RGB{R: 0x2d, G: 0x35, B: 0x3b}, // bg0
	BorderColor: ansi.RGB{R: 0x47, G: 0x52, B: 0x58}, // bg3
	Focus:       ansi.RGB{R: 0xe6, G: 0x98, B: 0x75}, // orange
	Selection:   ansi.RGB{R: 0x54, G: 0x3a, B: 0x48}, // bg_visual
	Border:      basetypes.NormalBorder,
}

// GitHubDark is GitHub's dark color scheme, drawn from GitHub Primer's
// published color primitives (https://primer.style/primitives). Normal
// borders match GitHub's clean, sharp-edged UI.
var githubDark = Theme{
	Primary:     ansi.RGB{R: 0x58, G: 0xa6, B: 0xff}, // accent.fg
	Secondary:   ansi.RGB{R: 0xa3, G: 0x71, B: 0xf7}, // done.fg
	Success:     ansi.RGB{R: 0x3f, G: 0xb9, B: 0x50}, // success.fg
	Warning:     ansi.RGB{R: 0xd2, G: 0x99, B: 0x22}, // attention.fg
	Error:       ansi.RGB{R: 0xf8, G: 0x51, B: 0x49}, // danger.fg
	Info:        ansi.RGB{R: 0x58, G: 0xa6, B: 0xff}, // accent.fg
	Muted:       ansi.RGB{R: 0x8b, G: 0x94, B: 0x9e}, // fg.muted
	Text:        ansi.RGB{R: 0xc9, G: 0xd1, B: 0xd9}, // fg.default
	TextInverse: ansi.RGB{R: 0x0d, G: 0x11, B: 0x17}, // canvas.default
	BorderColor: ansi.RGB{R: 0x30, G: 0x36, B: 0x3d}, // border.default
	Focus:       ansi.RGB{R: 0x58, G: 0xa6, B: 0xff}, // accent.fg
	Selection:   ansi.RGB{R: 0x16, G: 0x1b, B: 0x22}, // canvas.subtle
	Border:      basetypes.NormalBorder,
}

// HighContrast is a black-background, maximum-contrast theme for low-vision
// users and bright or washed-out displays. Every colour token, including
// Muted and the border, clears WCAG AAA (7:1) against the black background
// (TextInverse), which the tests assert; Text is pure white. The accents are
// distinct in lightness as well as hue, and Success, Warning and Error stay
// clearly different from one another. Focus and Selection are yellow so the
// focused row stands out from the cyan Primary. Borders are white.
var highContrast = Theme{
	Primary:     ansi.RGB{R: 0x00, G: 0xff, B: 0xff}, // cyan
	Secondary:   ansi.RGB{R: 0xff, G: 0x77, B: 0xff}, // pink
	Success:     ansi.RGB{R: 0x00, G: 0xff, B: 0x00}, // green
	Warning:     ansi.RGB{R: 0xff, G: 0xb0, B: 0x00}, // orange
	Error:       ansi.RGB{R: 0xff, G: 0x7f, B: 0x7f}, // light red
	Info:        ansi.RGB{R: 0x7f, G: 0xbf, B: 0xff}, // light blue
	Muted:       ansi.RGB{R: 0xb3, G: 0xb3, B: 0xb3}, // grey, still AAA
	Text:        ansi.RGB{R: 0xff, G: 0xff, B: 0xff}, // white
	TextInverse: ansi.RGB{R: 0x00, G: 0x00, B: 0x00}, // black background
	BorderColor: ansi.RGB{R: 0xff, G: 0xff, B: 0xff}, // white
	Focus:       ansi.RGB{R: 0xff, G: 0xff, B: 0x00}, // yellow
	Selection:   ansi.RGB{R: 0xff, G: 0xff, B: 0x00}, // yellow
	Border:      basetypes.NormalBorder,
}

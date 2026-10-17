package theme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/basetypes"
)

// presetTable maps a preset's file name to its constructor. The names are the
// lower-case, hyphenated forms Parse accepts for "base".
var presetTable = map[string]func() Theme{
	"dark": DarkTheme, "light": LightTheme, "dracula": DraculaTheme, "nord": NordTheme,
	"gruvbox": GruvboxTheme, "tokyo-night": TokyoNightTheme, "monokai": MonokaiTheme,
	"solarized-dark": SolarizedDarkTheme, "solarized-light": SolarizedLightTheme,
	"catppuccin": CatppuccinTheme, "catppuccin-latte": CatppuccinLatteTheme,
	"one-dark": OneDarkTheme, "one-light": OneLightTheme, "night-owl": NightOwlTheme,
	"rose-pine": RosePineTheme, "rose-pine-dawn": RosePineDawnTheme,
	"everforest-dark": EverforestDarkTheme, "github-dark": GitHubDarkTheme,
	"github-light": GitHubLightTheme, "high-contrast": HighContrastTheme,
}

// PresetNames returns the names Preset and Parse's "base" accept, sorted.
func PresetNames() []string {
	names := make([]string, 0, len(presetTable))
	for n := range presetTable {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Preset returns the built-in theme with the given name, such as "nord" or
// "catppuccin-latte" (see PresetNames), and whether there is one.
func Preset(name string) (Theme, bool) {
	ctor, ok := presetTable[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Theme{}, false
	}
	return ctor(), true
}

// fileTheme is the JSON form of a theme. Every field is optional.
type fileTheme struct {
	Base        string `json:"base"`
	Primary     string `json:"primary"`
	Secondary   string `json:"secondary"`
	Success     string `json:"success"`
	Warning     string `json:"warning"`
	Error       string `json:"error"`
	Info        string `json:"info"`
	Muted       string `json:"muted"`
	Text        string `json:"text"`
	TextInverse string `json:"text_inverse"`
	BorderColor string `json:"border_color"`
	Focus       string `json:"focus"`
	Selection   string `json:"selection"`
	Background  string `json:"background"`
	Surface     string `json:"surface"`
	Overlay     string `json:"overlay"`
	Border      string `json:"border"`
	// The rest changes what a colour alone cannot: see applyExtras.
	Glyphs     string                `json:"glyphs"`
	Spacing    *fileSpacing          `json:"spacing"`
	States     *fileStates           `json:"states"`
	Typography *fileTypography       `json:"typography"`
	Components map[string]fileTokens `json:"components"`
}

// Parse reads a theme from JSON, so the people who run a program can restyle it
// without a rebuild. The document starts from a built-in theme named by "base"
// (default "dark"; see PresetNames) and changes what it names:
//
//	{
//	  "base": "nord",
//	  "primary": "#ff8800",
//	  "text_inverse": "#101820",
//	  "border": "rounded"
//	}
//
// The colour fields are primary, secondary, success, warning, error, info,
// muted, text, text_inverse, border_color, focus, selection, background,
// surface and overlay, each "#rrggbb" or "#rgb". "border" is "normal",
// "rounded", "ascii", "double", "thick" or "hidden". An unknown field, an unknown base or a bad colour is an
// error that names it. When text, text_inverse or background changes, the
// surface, overlay and the focus, hover, disabled and selected states are
// derived again from the new colours, except for a surface or overlay the file
// sets. Parse does not judge contrast: call Theme.Check on the result.
//
// Beyond colours a file can set "glyphs" ("unicode", "ascii" or "nerd"),
// "spacing" ({"xs": 1, "s": 2, "m": 3, "l": 4}), the four interactive
// "states" and the six "typography" styles, each a style like {"fg": "#ff8800",
// "bg": "#101820", "bold": true} (also faint, italic, underline, reverse,
// strike), and "components": per-widget colours by widget name (see
// Components), like {"tabs": {"accent": "#ff8800"}} with the roles of Tokens in
// lower case. These apply after the colours, so they are not derived again.
func Parse(r io.Reader) (Theme, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Theme{}, err
	}
	var f fileTheme
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return Theme{}, fmt.Errorf("theme: %w", err)
	}
	if dec.More() {
		return Theme{}, fmt.Errorf("theme: more than one JSON value")
	}
	base := f.Base
	if base == "" {
		base = "dark"
	}
	t, ok := Preset(base)
	if !ok {
		return Theme{}, fmt.Errorf("theme: unknown base %q (one of %s)", base, strings.Join(PresetNames(), ", "))
	}
	changed := false
	set := func(name, value string, dst *ansi.Color) error {
		if value == "" {
			return nil
		}
		c, err := parseHex(value)
		if err != nil {
			return fmt.Errorf("theme: %s: %w", name, err)
		}
		*dst = c
		return nil
	}
	for _, e := range []struct {
		name, value string
		dst         *ansi.Color
		derives     bool
	}{
		{"primary", f.Primary, &t.Primary, true}, {"secondary", f.Secondary, &t.Secondary, true},
		{"success", f.Success, &t.Success, true}, {"warning", f.Warning, &t.Warning, true},
		{"error", f.Error, &t.Error, true}, {"info", f.Info, &t.Info, true},
		{"muted", f.Muted, &t.Muted, true}, {"text", f.Text, &t.Text, true},
		{"text_inverse", f.TextInverse, &t.TextInverse, true},
		{"border_color", f.BorderColor, &t.BorderColor, true},
		{"focus", f.Focus, &t.Focus, true}, {"selection", f.Selection, &t.Selection, true},
		{"background", f.Background, &t.Background, true},
		{"surface", f.Surface, &t.Surface, false}, {"overlay", f.Overlay, &t.Overlay, false},
	} {
		if e.value == "" {
			continue
		}
		if err := set(e.name, e.value, e.dst); err != nil {
			return Theme{}, err
		}
		if e.derives {
			changed = true
		}
	}
	switch strings.ToLower(f.Border) {
	case "":
	case "normal":
		t.Border = basetypes.NormalBorder
	case "rounded":
		t.Border = basetypes.RoundedBorder
	case "ascii":
		t.Border = basetypes.ASCIIBorder
	case "double":
		t.Border = basetypes.DoubleBorder
	case "thick":
		t.Border = basetypes.ThickBorder
	case "hidden":
		t.Border = basetypes.HiddenBorder
	default:
		return Theme{}, fmt.Errorf("theme: border: %q is not normal, rounded, ascii, double, thick or hidden", f.Border)
	}
	if changed {
		if f.Background == "" {
			t.Background = t.TextInverse
		}
		keepSurface, keepOverlay := t.Surface, t.Overlay
		fillSurfaces(&t)
		if f.Surface != "" {
			t.Surface = keepSurface
		}
		if f.Overlay != "" {
			t.Overlay = keepOverlay
		}
		t.States = States{}
		t.States = t.ResolvedStates()
	}
	if err := applyExtras(&t, f); err != nil {
		return Theme{}, err
	}
	return t, nil
}

// parseHex reads "#rgb" or "#rrggbb".
func parseHex(s string) (ansi.Color, error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return nil, fmt.Errorf("%q is not a colour like #rrggbb", s)
	}
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("%q is not a colour like #rrggbb", s)
	}
	return ansi.RGB{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, nil // #nosec G115 -- n was parsed from six hex digits
}

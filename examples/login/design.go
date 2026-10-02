package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// This file is the design-system side of the example: a list of settings,
// each a choice among named options, folded into one theme.Theme that the
// login panel, its form and its spinner are all drawn with.

// presets are the built-in themes, in the order the Theme setting cycles.
var presets = []struct {
	name string
	th   func() theme.Theme
}{
	{"Dark", theme.DarkTheme},
	{"Light", theme.LightTheme},
	{"Dracula", theme.DraculaTheme},
	{"Nord", theme.NordTheme},
	{"Gruvbox", theme.GruvboxTheme},
	{"Tokyo Night", theme.TokyoNightTheme},
	{"Monokai", theme.MonokaiTheme},
	{"Solarized Dark", theme.SolarizedDarkTheme},
	{"Solarized Light", theme.SolarizedLightTheme},
	{"Catppuccin", theme.CatppuccinTheme},
	{"One Dark", theme.OneDarkTheme},
	{"Night Owl", theme.NightOwlTheme},
	{"Rose Pine", theme.RosePineTheme},
	{"Everforest Dark", theme.EverforestDarkTheme},
	{"GitHub Dark", theme.GitHubDarkTheme},
	{"High Contrast", theme.HighContrastTheme},
}

// roles are the colour roles the Border color and Accent settings can pick.
var roles = []struct {
	name string
	of   func(theme.Theme) ansi.Color
}{
	{"Primary", func(t theme.Theme) ansi.Color { return t.Primary }},
	{"Secondary", func(t theme.Theme) ansi.Color { return t.Secondary }},
	{"Success", func(t theme.Theme) ansi.Color { return t.Success }},
	{"Warning", func(t theme.Theme) ansi.Color { return t.Warning }},
	{"Error", func(t theme.Theme) ansi.Color { return t.Error }},
	{"Info", func(t theme.Theme) ansi.Color { return t.Info }},
	{"Focus", func(t theme.Theme) ansi.Color { return t.Focus }},
	{"Muted", func(t theme.Theme) ansi.Color { return t.Muted }},
}

var borders = []struct {
	name string
	b    layout.Border
}{
	{"Single", layout.NormalBorder()},
	{"Rounded", layout.RoundedBorder()},
	{"Double", layout.DoubleBorder()},
	{"Thick", layout.ThickBorder()},
	{"ASCII", layout.ASCIIBorder()},
}

var glyphSets = []struct {
	name string
	g    func() theme.Glyphs
}{
	{"Unicode", theme.UnicodeGlyphSet},
	{"ASCII", theme.ASCIIGlyphSet},
	{"Nerd Font", theme.NerdGlyphSet},
}

var profiles = []struct {
	name string
	p    ansi.Profile
}{
	{"True color", ansi.TrueColor},
	{"256 colors", ansi.ANSI256},
	{"16 colors", ansi.ANSI16},
	{"No color", ansi.NoColor},
}

// setting is one row of the design panel.
type setting int

const (
	setTheme setting = iota
	setMode
	setBorder
	setBorderColor
	setAccent
	setGlyphs
	setDepth
	setPadding
	setTitle
	setFill
	numSettings
)

// Title options.
const (
	titleH1 = iota
	titleH2
	titleBig
	titleBorder
)

// Fill options.
const (
	fillNone = iota
	fillSurface
	fillBackground
)

var settingNames = [numSettings]string{
	"Theme", "Mode", "Border", "Border color", "Accent",
	"Glyphs", "Color depth", "Padding", "Title", "Fill",
}

// withTheme prefixes "Theme" (meaning: keep the preset's own value) to names.
func withTheme(names ...string) []string { return append([]string{"Theme"}, names...) }

// options lists the choices of s.
func options(s setting) []string {
	var out []string
	switch s {
	case setTheme:
		for _, p := range presets {
			out = append(out, p.name)
		}
	case setMode:
		out = []string{"As designed", "Light twin"}
	case setBorder:
		for _, b := range borders {
			out = append(out, b.name)
		}
		out = append(withTheme(out...), "None")
	case setBorderColor, setAccent:
		for _, r := range roles {
			out = append(out, r.name)
		}
		out = withTheme(out...)
	case setGlyphs:
		for _, g := range glyphSets {
			out = append(out, g.name)
		}
	case setDepth:
		for _, p := range profiles {
			out = append(out, p.name)
		}
	case setPadding:
		out = []string{"Small", "Medium", "Large"}
	case setTitle:
		out = []string{"Heading", "Subheading", "Big text", "On border"}
	case setFill:
		out = []string{"None", "Surface", "Background"}
	}
	return out
}

// design is the chosen option of every setting and the highlighted row.
type design struct {
	pick   [numSettings]int
	cursor setting
}

func (d design) value(s setting) string { return options(s)[d.pick[s]] }

// move highlights the previous (delta -1) or next (+1) setting, wrapping.
func (d design) move(delta int) design {
	d.cursor = setting((int(d.cursor) + delta + int(numSettings)) % int(numSettings))
	return d
}

// change chooses the previous or next option of the highlighted setting,
// wrapping.
func (d design) change(delta int) design {
	n := len(options(d.cursor))
	d.pick[d.cursor] = (d.pick[d.cursor] + delta + n) % n
	return d
}

// theme folds every setting into one theme. The colour depth is applied
// before the role-based picks, so a picked role is already downgraded.
func (d design) theme() theme.Theme {
	t := presets[d.pick[setTheme]].th()
	if d.pick[setMode] == 1 {
		t = t.LightTwin()
	}
	switch b := d.pick[setBorder]; {
	case b == len(borders)+1:
		t = t.Plain()
	case b > 0:
		t.Border = borders[b-1].b
	}
	t.Glyphs = glyphSets[d.pick[setGlyphs]].g()
	t = t.ForProfile(profiles[d.pick[setDepth]].p)
	if c := d.pick[setBorderColor]; c > 0 {
		t.BorderColor = roles[c-1].of(t)
	}
	if a := d.pick[setAccent]; a > 0 {
		// Component tokens override the accent of these widgets only; the
		// rest of the theme keeps its roles.
		c := roles[a-1].of(t)
		tok := theme.Tokens{Accent: c, Focus: c}
		t = t.WithTokens(theme.ComponentForm, tok).WithTokens(theme.ComponentSpinner, tok)
	}
	return t
}

// padding is the panels' horizontal padding, from the theme's spacing scale.
func (d design) padding(t theme.Theme) int {
	s := t.ResolvedSpacing()
	return [...]int{s.S, s.M, s.L}[d.pick[setPadding]]
}

// fill is the panel background the Fill setting asks for, nil for none.
func (d design) fill(t theme.Theme) ansi.Color {
	switch d.pick[setFill] {
	case fillSurface:
		if t.Surface != nil {
			return t.Surface
		}
		return t.Selection // presets without a Surface role
	case fillBackground:
		if t.Background != nil {
			return t.Background
		}
		return t.TextInverse
	}
	return nil
}

// designWidth is the design panel's content width.
const designWidth = 40

// panel renders the design panel's content: the settings, then samples of
// the resulting palette, typography and glyphs, and its contrast check.
func (d design) panel(t theme.Theme, active bool, hints string, w int) string {
	states := t.ResolvedStates()
	ty := t.ResolvedTypography()
	muted := ansi.NewStyle().Foreground(t.Muted)
	g := t.GlyphSet()

	var b strings.Builder
	b.WriteString(ty.H2.Render("Design system") + "\n\n")
	for s := setting(0); s < numSettings; s++ {
		label := fmt.Sprintf("%-13s", settingNames[s])
		val := d.value(s)
		var line string
		if s == d.cursor && active {
			line = states.Focus.Bold().Render("› "+label) +
				states.Selected.Render(" ◂ "+val+" ▸ ")
		} else {
			line = "  " + muted.Render(label) + " " + val
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + ty.Strong.Render("Palette") + "\n")
	sw := []struct {
		name string
		c    ansi.Color
	}{
		{"Primary", t.Primary}, {"Second.", t.Secondary}, {"Success", t.Success},
		{"Warning", t.Warning}, {"Error", t.Error}, {"Info", t.Info},
		{"Focus", t.Focus}, {"Muted", t.Muted}, {"Text", t.Text},
		{"Border", t.BorderColor}, {"Select", t.Selection}, {"Surface", t.Surface},
	}
	for i, s := range sw {
		chip := "  "
		if s.c != nil {
			chip = ansi.NewStyle().Background(s.c).Render("  ")
		}
		b.WriteString(chip + " " + fmt.Sprintf("%-9s", s.name))
		if i%3 == 2 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}

	b.WriteString("\n" + ty.Strong.Render("Type") + "  " +
		ty.H1.Render("H1") + " " + ty.H2.Render("H2") + " " + ty.Strong.Render("bold") + " " +
		ty.Emphasis.Render("italic") + " " + ty.Code.Render("code") + " " + ty.Link.Render("link") + "\n")
	b.WriteString(ty.Strong.Render("Glyph") + " " + strings.Join([]string{
		g.Check, g.Cross, g.Warning, g.Info, g.Dot, g.DotEmpty, g.Bullet, g.Arrow,
		strings.Repeat(g.BarFull, 3) + strings.Repeat(g.BarEmpty, 2), strings.Repeat(g.Mask, 3),
	}, " ") + "\n")

	if issues := t.Check(4.5); len(issues) == 0 {
		b.WriteString(ansi.NewStyle().Foreground(t.Success).Render(g.Check+" WCAG AA: every role ≥ 4.5:1") + "\n")
	} else {
		parts := make([]string, len(issues))
		for i, is := range issues {
			parts[i] = fmt.Sprintf("%s %.1f", is.Role, is.Ratio)
		}
		b.WriteString(ansi.NewStyle().Foreground(t.Warning).Render(
			ansi.Truncate(fmt.Sprintf("%s AA: %d low — %s", g.Warning, len(issues), strings.Join(parts, ", ")), w)) + "\n")
	}

	b.WriteString("\n" + hints)
	return b.String()
}

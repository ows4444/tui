package theme

import "github.com/ows4444/tui/ansi"

// States are the styles for a control's interactive states. A zero Style in
// any field means "derive it from the colour roles", which
// Theme.ResolvedStates does, so a Theme built before States existed keeps
// working and reads the same.
type States struct {
	// Focus styles the control that has keyboard focus.
	Focus ansi.Style
	// Hover styles the control under the pointer.
	Hover ansi.Style
	// Disabled styles a control that cannot be used.
	Disabled ansi.Style
	// Selected styles a chosen item.
	Selected ansi.Style
}

// ResolvedStates returns t.States with each empty style replaced by one built
// from t's colour roles: Focus in t.Focus, Hover in t.Primary, Disabled in
// t.Muted, Selected on a t.Selection background in t.Text. A style the theme
// set explicitly is kept as it is.
func (t Theme) ResolvedStates() States {
	s := t.States
	var zero ansi.Style
	if s.Focus == zero {
		s.Focus = ansi.NewStyle().Foreground(t.Focus)
	}
	if s.Hover == zero {
		s.Hover = ansi.NewStyle().Foreground(t.Primary)
	}
	if s.Disabled == zero {
		s.Disabled = ansi.NewStyle().Foreground(t.Muted)
	}
	if s.Selected == zero {
		s.Selected = ansi.NewStyle().Foreground(t.Text).Background(t.Selection)
	}
	return s
}

// Auto picks a theme from the terminal's background: Light when the
// background is light, Dark when it is dark or could not be detected. Give it
// to tui.WithTheme.
type Auto struct {
	Dark, Light Theme
}

// DefaultAuto returns Auto{Dark: DarkTheme(), Light: LightTheme()}.
func DefaultAuto() Auto { return Auto{Dark: dark, Light: light} }

// For returns the theme for a terminal whose background is bg.
func (a Auto) For(bg ansi.RGB) Theme {
	if isLight(bg) {
		return a.Light
	}
	return a.Dark
}

func init() {
	// Fill the presets' states from their roles, so every built-in theme
	// carries all four and reads the same either way.
	for _, t := range []*Theme{&dark, &light, &dracula, &nord, &gruvbox, &tokyoNight, &monokai,
		&solarizedDark, &solarizedLight, &catppuccin, &oneDark, &nightOwl, &rosePine,
		&everforestDark, &githubDark, &highContrast} {
		fillSurfaces(t)
		t.States = t.ResolvedStates()
	}
}

// fillSurfaces sets a preset's Background to its TextInverse (the colour the
// preset is designed on) and derives Surface and Overlay as steps of 6% and 12%
// from the background toward Text, so panels and floating layers read as raised.
func fillSurfaces(t *Theme) {
	t.Background = t.TextInverse
	bg, fg := rgbOf3(t.Background), rgbOf3(t.Text)
	t.Surface = asColor(mix(bg, fg, 0.06))
	t.Overlay = asColor(mix(bg, fg, 0.12))
}

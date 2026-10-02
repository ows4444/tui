package theme

import "github.com/ows4444/tui/ansi"

// mix returns a blended with b; t=0 is a, t=1 is b.
func mix(a, b [3]uint8, t float64) [3]uint8 {
	var o [3]uint8
	for i := range o {
		o[i] = uint8(float64(a[i])*(1-t) + float64(b[i])*t + 0.5)
	}
	return o
}

func rgbOf3(c ansi.Color) [3]uint8 {
	r, g, b, _ := toRGB(c)
	return [3]uint8{r, g, b}
}

func asColor(c [3]uint8) ansi.Color { return ansi.RGB{R: c[0], G: c[1], B: c[2]} }

// darkenTo darkens c toward black, keeping its hue, until it reaches min
// contrast against bg.
func darkenTo(c, bg [3]uint8, min float64) [3]uint8 {
	black := [3]uint8{}
	for step := 0; step <= 20; step++ {
		out := mix(c, black, float64(step)/20)
		if Contrast(asColor(out), asColor(bg)) >= min {
			return out
		}
	}
	return black
}

// isLightTheme reports whether t's background (TextInverse) is light.
func isLightTheme(t Theme) bool {
	r, g, b, ok := toRGB(t.background())
	return ok && luminance(r, g, b) > 0.4
}

// LightTwin returns the light-background counterpart of t. The twin keeps t's
// hues, Border and other settings, but swaps the roles of text and
// background: the background is t's Text tinted toward white, Text is t's
// background, and every accent is darkened until it reaches a 4.5:1 contrast
// against the new background, so the result passes Check(4.5). A theme that is
// already light is returned unchanged.
func (t Theme) LightTwin() Theme {
	if isLightTheme(t) || t.Text == nil || t.TextInverse == nil {
		return t
	}
	white := [3]uint8{255, 255, 255}
	black := [3]uint8{}
	bg := mix(rgbOf3(t.Text), white, 0.7)
	accent := func(c ansi.Color) ansi.Color {
		if c == nil {
			return nil
		}
		return asColor(darkenTo(rgbOf3(c), bg, 4.5))
	}
	out := t
	out.TextInverse = asColor(bg)
	out.Background = out.TextInverse
	out.Surface = asColor(mix(bg, black, 0.04))
	out.Overlay = asColor(mix(bg, black, 0.08))
	out.Text = asColor(darkenTo(rgbOf3(t.TextInverse), bg, 4.5))
	out.Primary, out.Secondary = accent(t.Primary), accent(t.Secondary)
	out.Success, out.Warning = accent(t.Success), accent(t.Warning)
	out.Error, out.Info = accent(t.Error), accent(t.Info)
	out.Muted, out.Focus = accent(t.Muted), accent(t.Focus)
	out.BorderColor = asColor(mix(bg, black, 0.2))
	out.Selection = asColor(mix(bg, black, 0.1))
	// The dark states were derived from the dark roles; derive them again.
	out.States = States{}
	out.States = out.ResolvedStates()
	out.Typography = Typography{}
	return out
}

// Pair returns the Auto for preset: preset on a dark background and its
// LightTwin on a light one. Give it to tui.WithTheme.
func Pair(preset Theme) Auto {
	return Auto{Dark: preset, Light: preset.LightTwin()}
}

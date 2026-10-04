package faces

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func fg(c ansi.Color) string {
	return strings.SplitN(ansi.NewStyle().Foreground(c).Render("x"), "x", 2)[0]
}

// With no Color the face is drawn in the theme's primary colour, exactly as
// before the field existed.
func TestNoColorDrawsInTheThemesPrimary(t *testing.T) {
	m := New()
	frame := m.Face().Frames(m.Size)[0]
	if got, want := m.View(), ansi.NewStyle().Foreground(m.Theme.Primary).Render(frame); got != want {
		t.Errorf("View without a Color is not the frame in the primary colour:\n%q\n%q", got, want)
	}
	if m.Tokens().Accent != m.Theme.Primary {
		t.Errorf("Tokens().Accent = %v, want the theme's primary", m.Tokens().Accent)
	}
}

// A Color draws the face in that colour, and an Accent from WithTokens still
// overrides it.
func TestColorDrawsTheFaceAndTokensStillWin(t *testing.T) {
	pink, green := ansi.RGB{R: 230, G: 120, B: 180}, ansi.RGB{R: 20, G: 200, B: 90}
	m := New()
	m.Color = pink
	if v := m.View(); !strings.HasPrefix(v, fg(pink)) || strings.Contains(v, fg(m.Theme.Primary)) {
		t.Errorf("a face with a Color is not drawn in it: %q", v[:40])
	}
	if m.Tokens().Accent != ansi.Color(pink) {
		t.Errorf("Tokens().Accent = %v, want the Color", m.Tokens().Accent)
	}
	n := m.WithTokens(theme.Tokens{Accent: green})
	if v := n.View(); !strings.HasPrefix(v, fg(green)) || strings.Contains(v, fg(pink)) {
		t.Errorf("WithTokens did not override the Color: %q", v[:40])
	}
	if n.Tokens().Accent != ansi.Color(green) {
		t.Errorf("Tokens().Accent = %v, want the token", n.Tokens().Accent)
	}
	// A token that names another role leaves the Color in charge.
	o := m.WithTokens(theme.Tokens{Muted: green})
	if v := o.View(); !strings.HasPrefix(v, fg(pink)) {
		t.Errorf("a token for another role replaced the Color: %q", v[:40])
	}
	// The label keeps its own faint style; only the sprite takes the Color.
	m.ShowLabel = true
	if rows := strings.Split(m.View(), "\n"); strings.Contains(rows[len(rows)-1], fg(pink)) {
		t.Error("the label is drawn in the Color")
	}
}

// ColorFor is stable, ignores case and surrounding space, spreads over the
// hues, and is chosen independently of the face.
func TestColorForNames(t *testing.T) {
	if ColorFor("Ada") != ColorFor(" ada ") || ColorFor("ada") != ColorFor("ada") {
		t.Error("ColorFor is not stable under case and space")
	}
	if got, want := ColorFor("ada"), (ansi.RGB{R: 0x62, G: 0xd8, B: 0x5a}); got != want {
		t.Errorf("ColorFor(ada) = %#v, want %#v", got, want)
	}
	colours := map[ansi.RGB]bool{}
	sameFace := map[int]map[ansi.RGB]bool{}
	for i := 0; i < 400; i++ {
		name := fmt.Sprintf("user-%d", i)
		c := ColorFor(name)
		colours[c] = true
		if sameFace[IndexFor(name)] == nil {
			sameFace[IndexFor(name)] = map[ansi.RGB]bool{}
		}
		sameFace[IndexFor(name)][c] = true
		// Every colour has the same lightness and saturation: its lightest
		// and darkest channels are fixed, whatever the hue.
		hi, lo := math.Max(float64(c.R), math.Max(float64(c.G), float64(c.B))), math.Min(float64(c.R), math.Min(float64(c.G), float64(c.B)))
		if hi != 216 || lo != 90 {
			t.Fatalf("ColorFor(%q) = %v: channels span %v..%v, want 90..216", name, c, lo, hi)
		}
	}
	if len(colours) < 200 {
		t.Errorf("400 names drew only %d colours", len(colours))
	}
	for face, cs := range sameFace {
		if len(cs) < 2 {
			t.Errorf("every name with face %d has the same colour", face)
		}
	}
}

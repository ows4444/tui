package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestToggleOn proves criterion #324: on=true renders a distinct 'on'
// visual state colored via t.
func TestToggleOn(t *testing.T) {
	tt := theme.DarkTheme()
	got := Toggle("Notifications", true, false, tt)
	want := ansi.NewStyle().Foreground(tt.Success).Render("(●)") + " Notifications"
	if got != want {
		t.Errorf("Toggle(%q, true, false, Dark) = %q, want %q", "Notifications", got, want)
	}
}

// TestToggleOff proves criterion #325: on=false renders a distinct 'off'
// visual state, visually different from on, colored via t.
func TestToggleOff(t *testing.T) {
	tt := theme.DarkTheme()
	got := Toggle("Notifications", false, false, tt)
	want := ansi.NewStyle().Foreground(tt.Muted).Render("( )") + " Notifications"
	if got != want {
		t.Errorf("Toggle(%q, false, false, Dark) = %q, want %q", "Notifications", got, want)
	}

	on := Toggle("Notifications", true, false, tt)
	if got == on {
		t.Errorf("Toggle(off) and Toggle(on) rendered identically: %q", got)
	}
	if ansi.StripANSI(got) == ansi.StripANSI(on) {
		t.Errorf("Toggle(off) and Toggle(on) visible text identical: %q", ansi.StripANSI(got))
	}
}

// TestToggleFocus proves criterion #326: focused=true renders distinct
// from focused=false, matching Checkbox's focus convention.
func TestToggleFocus(t *testing.T) {
	tt := theme.DarkTheme()

	offFocused := Toggle("Notifications", false, true, tt)
	offBlurred := Toggle("Notifications", false, false, tt)
	if offFocused == offBlurred {
		t.Errorf("Toggle(false, focused=true) == Toggle(false, focused=false): %q", offFocused)
	}
	wantOffFocused := ansi.NewStyle().Foreground(tt.Focus).Bold().Render("( )") + " Notifications"
	if offFocused != wantOffFocused {
		t.Errorf("Toggle(false, true, Dark) = %q, want %q", offFocused, wantOffFocused)
	}

	onFocused := Toggle("Notifications", true, true, tt)
	onBlurred := Toggle("Notifications", true, false, tt)
	if onFocused == onBlurred {
		t.Errorf("Toggle(true, focused=true) == Toggle(true, focused=false): %q", onFocused)
	}
	wantOnFocused := ansi.NewStyle().Foreground(tt.Success).Bold().Render("(●)") + " Notifications"
	if onFocused != wantOnFocused {
		t.Errorf("Toggle(true, true, Dark) = %q, want %q", onFocused, wantOnFocused)
	}
}

// TestToggleEmptyLabel proves criterion #327 for Toggle: an empty label
// renders just the indicator, no stray trailing space.
func TestToggleEmptyLabel(t *testing.T) {
	tt := theme.DarkTheme()
	for _, on := range []bool{true, false} {
		for _, focused := range []bool{true, false} {
			got := Toggle("", on, focused, tt)
			if strings.HasSuffix(got, " ") {
				t.Errorf("Toggle(%q, %v, %v, Dark) = %q, has a trailing space", "", on, focused, got)
			}
			plain := ansi.StripANSI(got)
			wantPlain := "( )"
			if on {
				wantPlain = "(●)"
			}
			if plain != wantPlain {
				t.Errorf("Toggle(\"\", %v, %v, Dark) visible text = %q, want %q", on, focused, plain, wantPlain)
			}
		}
	}
}

// TestToggleUsesGivenTheme confirms Toggle reads colors from t, not a
// hardcoded theme.
func TestToggleUsesGivenTheme(t *testing.T) {
	got := Toggle("Notifications", true, false, theme.LightTheme())
	want := ansi.NewStyle().Foreground(theme.LightTheme().Success).Render("(●)") + " Notifications"
	if got != want {
		t.Errorf("Toggle(%q, true, false, Light) = %q, want %q", "Notifications", got, want)
	}
}

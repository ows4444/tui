package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestCheckboxChecked proves criterion #321: checked=true renders a
// filled indicator ("[x]") followed by the label.
func TestCheckboxChecked(t *testing.T) {
	tt := theme.DarkTheme()
	got := Checkbox("Agree", true, false, tt)
	want := ansi.NewStyle().Foreground(tt.Success).Render("[x]") + " Agree"
	if got != want {
		t.Errorf("Checkbox(%q, true, false, Dark) = %q, want %q", "Agree", got, want)
	}
	if !strings.Contains(ansi.StripANSI(got), "[x]") {
		t.Errorf("Checkbox(checked=true) visible text = %q, want it to contain %q", ansi.StripANSI(got), "[x]")
	}
}

// TestCheckboxUnchecked proves criterion #322: checked=false renders an
// empty indicator ("[ ]") followed by the label, distinct from checked.
func TestCheckboxUnchecked(t *testing.T) {
	tt := theme.DarkTheme()
	got := Checkbox("Agree", false, false, tt)
	want := ansi.NewStyle().Foreground(tt.Muted).Render("[ ]") + " Agree"
	if got != want {
		t.Errorf("Checkbox(%q, false, false, Dark) = %q, want %q", "Agree", got, want)
	}

	checked := Checkbox("Agree", true, false, tt)
	if got == checked {
		t.Errorf("Checkbox(checked=false) and Checkbox(checked=true) rendered identically: %q", got)
	}
	if strings.Contains(ansi.StripANSI(got), "[x]") {
		t.Errorf("Checkbox(checked=false) visible text = %q, must not contain the checked indicator", ansi.StripANSI(got))
	}
}

// TestCheckboxFocus proves criterion #323: focused=true renders distinct
// from focused=false, using the given Theme's colors.
func TestCheckboxFocus(t *testing.T) {
	tt := theme.DarkTheme()

	uncheckedFocused := Checkbox("Agree", false, true, tt)
	uncheckedBlurred := Checkbox("Agree", false, false, tt)
	if uncheckedFocused == uncheckedBlurred {
		t.Errorf("Checkbox(false, focused=true) == Checkbox(false, focused=false): %q", uncheckedFocused)
	}
	wantFocused := ansi.NewStyle().Foreground(tt.Focus).Bold().Render("[ ]") + " Agree"
	if uncheckedFocused != wantFocused {
		t.Errorf("Checkbox(false, true, Dark) = %q, want %q", uncheckedFocused, wantFocused)
	}

	checkedFocused := Checkbox("Agree", true, true, tt)
	checkedBlurred := Checkbox("Agree", true, false, tt)
	if checkedFocused == checkedBlurred {
		t.Errorf("Checkbox(true, focused=true) == Checkbox(true, focused=false): %q", checkedFocused)
	}
}

// TestCheckboxEmptyLabel proves criterion #327 for Checkbox: an empty
// label renders just the indicator, no stray trailing space.
func TestCheckboxEmptyLabel(t *testing.T) {
	tt := theme.DarkTheme()
	for _, checked := range []bool{true, false} {
		for _, focused := range []bool{true, false} {
			got := Checkbox("", checked, focused, tt)
			if strings.HasSuffix(got, " ") {
				t.Errorf("Checkbox(%q, %v, %v, Dark) = %q, has a trailing space", "", checked, focused, got)
			}
			plain := ansi.StripANSI(got)
			wantPlain := "[ ]"
			if checked {
				wantPlain = "[x]"
			}
			if plain != wantPlain {
				t.Errorf("Checkbox(\"\", %v, %v, Dark) visible text = %q, want %q", checked, focused, plain, wantPlain)
			}
		}
	}
}

// TestCheckboxUsesGivenTheme confirms Checkbox reads colors from t, not a
// hardcoded theme (same convention as TestBadgeUsesGivenTheme).
func TestCheckboxUsesGivenTheme(t *testing.T) {
	got := Checkbox("Agree", true, false, theme.LightTheme())
	want := ansi.NewStyle().Foreground(theme.LightTheme().Success).Render("[x]") + " Agree"
	if got != want {
		t.Errorf("Checkbox(%q, true, false, Light) = %q, want %q", "Agree", got, want)
	}
}

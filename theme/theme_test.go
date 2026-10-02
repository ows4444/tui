package theme

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui/layout"
)

// TestBuiltinThemesHaveNoNilColors guards against a forgotten field in
// Dark/Light: ansi.Color is an interface, so an unset field is nil and
// would panic the first time something calls its Render method, rather
// than failing to compile.
func TestBuiltinThemesHaveNoNilColors(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
	}{
		{"Dark", DarkTheme()},
		{"Light", LightTheme()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := map[string]any{
				"Primary":     tt.theme.Primary,
				"Secondary":   tt.theme.Secondary,
				"Success":     tt.theme.Success,
				"Warning":     tt.theme.Warning,
				"Error":       tt.theme.Error,
				"Info":        tt.theme.Info,
				"Muted":       tt.theme.Muted,
				"Text":        tt.theme.Text,
				"TextInverse": tt.theme.TextInverse,
				"BorderColor": tt.theme.BorderColor,
				"Focus":       tt.theme.Focus,
				"Selection":   tt.theme.Selection,
			}
			for name, v := range fields {
				if v == nil {
					t.Errorf("%s.%s is nil", tt.name, name)
				}
			}
			if tt.theme.Border == (layout.Border{}) {
				t.Errorf("%s.Border is the zero value", tt.name)
			}
		})
	}
}

func TestDarkAndLightAreDistinct(t *testing.T) {
	if DarkTheme().Primary == LightTheme().Primary {
		t.Error("Dark and Light have the same Primary color; they should look visibly different")
	}
	if DarkTheme().Border == LightTheme().Border {
		t.Error("Dark and Light have the same Border style; InkUI's are single vs rounded")
	}
}

// TestPlainClearsBorderOnly proves acceptance criterion for the
// box-drawing-decoration half: Plain zeroes Border (so layout.Box.Border
// treats it as "no border") and changes nothing else, on any existing
// Theme value, not just Dark/Light.
func TestPlainClearsBorderOnly(t *testing.T) {
	tests := []struct {
		name  string
		theme Theme
	}{
		{"Dark", DarkTheme()},
		{"Light", LightTheme()},
		{"Dracula", DraculaTheme()},
		{"Catppuccin", CatppuccinTheme()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.theme.Plain()

			if got.Border != (layout.Border{}) {
				t.Errorf("%s.Plain().Border = %+v, want the zero value", tt.name, got.Border)
			}

			// Every other field must be untouched.
			want := tt.theme
			want.Border = layout.Border{}
			if got != want {
				t.Errorf("%s.Plain() changed a field other than Border:\n got  %+v\n want %+v", tt.name, got, want)
			}

			// The original theme value itself must be unmodified (Plain
			// takes a value receiver, so this should be structurally
			// guaranteed, but assert it directly rather than just trusting
			// the language rule).
			if tt.theme.Border == (layout.Border{}) {
				t.Errorf("%s's own Border was mutated by calling Plain()", tt.name)
			}
		})
	}
}

func TestSpacingDefaultsAndComparable(t *testing.T) {
	if !reflect.TypeOf(Theme{}).Comparable() {
		t.Fatal("Theme must be comparable")
	}
	if got := (Theme{}).ResolvedSpacing(); got != (SpacingScale{XS: 1, S: 1, M: 2, L: 3}) {
		t.Fatalf("defaults = %+v", got)
	}
	if got := (Theme{Spacing: SpacingScale{S: 4}}).ResolvedSpacing(); got.S != 4 || got.M != 2 {
		t.Fatalf("override = %+v", got)
	}
}

package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestTagSolidUsesBadgeConvention(t *testing.T) {
	tt := theme.DarkTheme()
	tests := []struct {
		name    string
		text    string
		variant Variant
		color   ansi.Color
	}{
		{"neutral", "n/a", VariantNeutral, tt.Muted},
		{"info", "info", VariantInfo, tt.Info},
		{"success", "OK", VariantSuccess, tt.Success},
		{"warning", "warn", VariantWarning, tt.Warning},
		{"error", "FAIL", VariantError, tt.Error},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Tag(test.text, TagSolid, test.variant, tt)
			want := ansi.NewStyle().Bold().Background(test.color).Foreground(tt.TextInverse).Render(" " + markPrefix(test.variant, tt) + test.text + " ")
			if got != want {
				t.Errorf("Tag(%q, TagSolid, %v, Dark) = %q, want %q", test.text, test.variant, got, want)
			}
		})
	}
}

func TestTagOutlineHasNoBackgroundFill(t *testing.T) {
	tt := theme.DarkTheme()
	tests := []struct {
		name    string
		variant Variant
		color   ansi.Color
	}{
		{"neutral", VariantNeutral, tt.Muted},
		{"info", VariantInfo, tt.Info},
		{"success", VariantSuccess, tt.Success},
		{"warning", VariantWarning, tt.Warning},
		{"error", VariantError, tt.Error},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Tag("tag", TagOutline, test.variant, tt)
			want := ansi.NewStyle().Bold().Foreground(test.color).Render("[" + markPrefix(test.variant, tt) + "tag]")
			if got != want {
				t.Errorf("Tag(%q, TagOutline, %v, Dark) = %q, want %q", "tag", test.variant, got, want)
			}
		})
	}
}

// TestTagSolidVsOutlineDistinct proves the solid and outline styles render
// differently for the same text/variant/theme (criterion #498).
func TestTagSolidVsOutlineDistinct(t *testing.T) {
	tt := theme.DarkTheme()
	solid := Tag("label", TagSolid, VariantWarning, tt)
	outline := Tag("label", TagOutline, VariantWarning, tt)
	if solid == outline {
		t.Errorf("Tag TagSolid and TagOutline rendered identically: %q", solid)
	}

	// Extract the background-only SGR code for this color (e.g. "103") and
	// confirm solid carries it while outline does not, so the two remain
	// visually distinct (filled pill vs colored border/text only).
	bgOnly := ansi.NewStyle().Background(VariantWarning.Color(tt)).Render("")
	bgCode := strings.TrimSuffix(strings.TrimPrefix(bgOnly, ansi.CSI), "m"+ansi.Reset)
	if bgCode == "" {
		t.Fatalf("could not extract background SGR code from %q", bgOnly)
	}
	if !strings.Contains(solid, bgCode) {
		t.Errorf("Tag TagSolid output %q does not contain expected background SGR code %q", solid, bgCode)
	}
	if strings.Contains(outline, bgCode) {
		t.Errorf("Tag TagOutline output %q unexpectedly contains background SGR code %q", outline, bgCode)
	}
}

// TestTagEmptyTextRendersMinimalChip proves empty text renders a minimal
// chip without panicking (criterion #500).
func TestTagEmptyTextRendersMinimalChip(t *testing.T) {
	tt := theme.DarkTheme()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Tag panicked on empty text: %v", r)
		}
	}()

	solid := Tag("", TagSolid, VariantInfo, tt)
	wantSolid := ansi.NewStyle().Bold().Background(tt.Info).Foreground(tt.TextInverse).Render("  ")
	if solid != wantSolid {
		t.Errorf("Tag(\"\", TagSolid, VariantInfo, Dark) = %q, want %q", solid, wantSolid)
	}

	outline := Tag("", TagOutline, VariantInfo, tt)
	wantOutline := ansi.NewStyle().Bold().Foreground(tt.Info).Render("[]")
	if outline != wantOutline {
		t.Errorf("Tag(\"\", TagOutline, VariantInfo, Dark) = %q, want %q", outline, wantOutline)
	}
}

// TestTagUsesGivenTheme proves Tag reads colors from the passed theme, not
// a hardcoded one.
func TestTagUsesGivenTheme(t *testing.T) {
	got := Tag("OK", TagSolid, VariantSuccess, theme.LightTheme())
	want := ansi.NewStyle().Bold().Background(theme.LightTheme().Success).Foreground(theme.LightTheme().TextInverse).Render(" " + markPrefix(VariantSuccess, theme.LightTheme()) + "OK ")
	if got != want {
		t.Errorf("Tag(%q, TagSolid, VariantSuccess, Light) = %q, want %q", "OK", got, want)
	}
}

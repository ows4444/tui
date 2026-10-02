package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestAlertVariantColor proves criterion #345: Alert colors its border
// and leading severity icon via the given Variant's Color(t), reusing
// widgets.Variant.
func TestAlertVariantColor(t *testing.T) {
	dt := theme.DarkTheme()
	for _, tc := range []struct {
		name    string
		variant Variant
	}{
		{"info", VariantInfo},
		{"success", VariantSuccess},
		{"warning", VariantWarning},
		{"error", VariantError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Alert("hello", tc.variant, dt, 0)
			color := tc.variant.Color(dt)

			lines := strings.Split(got, "\n")
			topPlain := ansi.StripANSI(lines[0])
			wantTop := ansi.NewStyle().Foreground(color).Render(topPlain)
			if lines[0] != wantTop {
				t.Errorf("top border line = %q, want %q (colored via variant.Color(t))", lines[0], wantTop)
			}

			icon := alertIcon(tc.variant)
			if !strings.Contains(got, icon) {
				t.Errorf("Alert output does not contain severity icon %q", icon)
			}
			if !strings.Contains(got, "hello") {
				t.Errorf("Alert output does not contain message %q", "hello")
			}
		})
	}
}

// TestAlertIconDiffersByVariant proves criterion #346: VariantError and
// VariantSuccess render different severity icons, so severity is
// distinguishable even without color.
func TestAlertIconDiffersByVariant(t *testing.T) {
	dt := theme.DarkTheme()
	errAlert := Alert("msg", VariantError, dt, 0)
	okAlert := Alert("msg", VariantSuccess, dt, 0)

	errIcon := alertIcon(VariantError)
	okIcon := alertIcon(VariantSuccess)

	if errIcon == okIcon {
		t.Fatalf("VariantError and VariantSuccess icons must differ, both are %q", errIcon)
	}
	if !strings.Contains(errAlert, errIcon) {
		t.Errorf("error Alert missing icon %q", errIcon)
	}
	if !strings.Contains(okAlert, okIcon) {
		t.Errorf("success Alert missing icon %q", okIcon)
	}
	if strings.Contains(errAlert, okIcon) {
		t.Errorf("error Alert unexpectedly contains success icon %q", okIcon)
	}
}

// TestAlertSizeToContent proves criterion #347 (width=0 case): Alert with
// width 0 sizes to its content like Box/Panel, i.e. does not pad every
// line to some fixed unrelated width.
func TestAlertSizeToContent(t *testing.T) {
	dt := theme.DarkTheme()
	got := Alert("hi", VariantInfo, dt, 0)
	lines := strings.Split(got, "\n")
	// The content line should be exactly as wide as icon+space+message
	// plus 1 column of padding on each side plus 1 border char each side,
	// not some arbitrary larger fixed width.
	want := ansi.Width(alertIcon(VariantInfo) + " hi")
	for _, l := range lines {
		w := ansi.Width(l)
		if w > want+4 {
			t.Errorf("line %q width %d exceeds expected size-to-content bound %d", l, w, want+4)
		}
	}
}

// TestAlertFixedWidth proves criterion #347 (width>0 case): every
// rendered line's ansi.Width stays within the requested width.
func TestAlertFixedWidth(t *testing.T) {
	dt := theme.DarkTheme()
	const width = 24
	got := Alert("this is a somewhat long alert message that should wrap across lines", VariantWarning, dt, width)
	for i, l := range strings.Split(got, "\n") {
		if w := ansi.Width(l); w > width {
			t.Errorf("line %d width = %d, want <= %d; line: %q", i, w, width, l)
		}
	}
}

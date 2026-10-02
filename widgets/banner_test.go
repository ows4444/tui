package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestBannerNoBorder proves criterion #348: Banner renders as a
// full-width solid/reverse-styled bar with no border characters, visually
// distinct from Alert's bordered box.
func TestBannerNoBorder(t *testing.T) {
	dt := theme.DarkTheme()
	got := Banner("hello", VariantError, dt, 40)

	borderChars := []string{
		dt.Border.TopLeft, dt.Border.TopRight,
		dt.Border.BottomLeft, dt.Border.BottomRight,
		dt.Border.Top, dt.Border.Left, dt.Border.Right,
	}
	plain := ansi.StripANSI(got)
	for _, bc := range borderChars {
		if bc != "" && strings.Contains(plain, bc) {
			t.Errorf("Banner output %q contains border character %q", plain, bc)
		}
	}
	if strings.Count(got, "\n") != 0 {
		t.Errorf("Banner output should be a single-line bar, got %d newlines", strings.Count(got, "\n"))
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("Banner output does not contain message %q", "hello")
	}
}

// TestBannerWidthExact proves criterion #349: Banner pads every rendered
// line so its ansi.Width equals the given width exactly.
func TestBannerWidthExact(t *testing.T) {
	dt := theme.DarkTheme()
	for _, width := range []int{10, 40, 80} {
		got := Banner("hi", VariantSuccess, dt, width)
		if w := ansi.Width(got); w != width {
			t.Errorf("width=%d: Banner ansi.Width = %d, want %d; got %q", width, w, width, got)
		}
	}
}

// TestBannerWidthExactTruncates proves criterion #349 also holds when the
// message is longer than the requested width: the rendered line is still
// padded/truncated to exactly that width.
func TestBannerWidthExactTruncates(t *testing.T) {
	dt := theme.DarkTheme()
	const width = 8
	got := Banner("this message is much longer than the width", VariantError, dt, width)
	if w := ansi.Width(got); w != width {
		t.Errorf("Banner ansi.Width = %d, want %d; got %q", w, width, got)
	}
}

// TestBannerVariantColor proves criterion #350: Banner colors the bar via
// the given Variant's Color(t), reusing widgets.Variant like Alert does.
func TestBannerVariantColor(t *testing.T) {
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
			got := Banner("msg", tc.variant, dt, 20)
			want := ansi.NewStyle().Background(tc.variant.Color(dt)).Foreground(dt.TextInverse).Render(padRow(markPrefix(tc.variant, dt)+"msg", 20))
			if got != want {
				t.Errorf("Banner(%v) = %q, want %q", tc.variant, got, want)
			}
		})
	}
}

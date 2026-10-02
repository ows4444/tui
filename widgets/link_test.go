package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestLinkRendersOSC8Hyperlink(t *testing.T) {
	tt := theme.DarkTheme()
	got := Link("click me", "https://example.com", false, tt)
	want := ansi.NewStyle().Underline().Foreground(tt.Primary).Render(ansi.Hyperlink("click me", "https://example.com"))
	if got != want {
		t.Errorf("Link(%q, %q, false, Dark) = %q, want %q", "click me", "https://example.com", got, want)
	}
	if !strings.Contains(got, "\x1b]8;;https://example.com\x1b\\") {
		t.Errorf("Link(...) = %q, want it to contain the OSC 8 open sequence for the href", got)
	}
}

func TestLinkShowHrefAppendsFaintURL(t *testing.T) {
	tt := theme.DarkTheme()
	got := Link("docs", "https://example.com", true, tt)
	wantSuffix := " " + ansi.NewStyle().Faint().Render("https://example.com")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Errorf("Link(%q, %q, true, Dark) = %q, want it to end with faint href %q", "docs", "https://example.com", got, wantSuffix)
	}
	if !strings.Contains(got, "\x1b]8;;https://example.com\x1b\\docs") {
		t.Errorf("Link(...) = %q, want it to still contain the OSC 8-wrapped text before the fallback URL", got)
	}
}

func TestLinkShowHrefFalseOmitsURL(t *testing.T) {
	tt := theme.DarkTheme()
	got := Link("docs", "https://example.com", false, tt)
	if strings.Contains(got, ansi.NewStyle().Faint().Render("https://example.com")) {
		t.Errorf("Link(%q, %q, false, Dark) = %q, want it not to contain the faint fallback URL", "docs", "https://example.com", got)
	}
}

func TestLinkEmptyHrefRendersPlainText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		showHref bool
	}{
		{"showHref false", "no link here", false},
		{"showHref true", "still no link", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Link(test.text, "", test.showHref, theme.DarkTheme())
			if got != test.text {
				t.Errorf("Link(%q, \"\", %v, Dark) = %q, want %q (plain text, no OSC 8 wrapping)", test.text, test.showHref, got, test.text)
			}
			if strings.Contains(got, "\x1b]8;;") {
				t.Errorf("Link(%q, \"\", %v, Dark) = %q, want no OSC 8 escape sequence for an empty href", test.text, test.showHref, got)
			}
		})
	}
}

func TestLinkEmptyHrefDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Link with empty href panicked: %v", r)
		}
	}()
	Link("", "", false, theme.DarkTheme())
	Link("text", "", true, theme.LightTheme())
}

func TestLinkUnsafeTargetRendersTextOnly(t *testing.T) {
	for _, href := range []string{"javascript:alert(1)", "https://x/\x1b]8;;evil\x07", "https://x/\x07", "https://x/\x1b\\", " https://x", "/relative", "ftp://x"} {
		for _, show := range []bool{false, true} {
			got := Link("text", href, show, theme.DarkTheme())
			if got != "text" {
				t.Errorf("Link(text, %q, %v) = %q, want plain text", href, show, got)
			}
		}
	}
	if got := Link("t", "HTTP://X.test", false, theme.DarkTheme()); !strings.Contains(got, "\x1b]8;;HTTP://X.test") {
		t.Errorf("mixed-case scheme should link, got %q", got)
	}
}

func TestLinkUsesGivenTheme(t *testing.T) {
	got := Link("go", "https://go.dev", false, theme.LightTheme())
	want := ansi.NewStyle().Underline().Foreground(theme.LightTheme().Primary).Render(ansi.Hyperlink("go", "https://go.dev"))
	if got != want {
		t.Errorf("Link(%q, %q, false, Light) = %q, want %q (should use Light's Primary color, not Dark's)", "go", "https://go.dev", got, want)
	}
}

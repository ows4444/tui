package markdown

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestTypographyH1DrivesHeading(t *testing.T) {
	th := theme.DarkTheme()
	th.Typography.H1 = ansi.NewStyle().Foreground(ansi.RGB{R: 1, G: 2, B: 3}).Italic()
	got := Render("# Title", 40, th)
	want := th.Typography.H1.Render("Title")
	if !strings.Contains(got, want) {
		t.Fatalf("H1 output %q does not use Typography.H1 (%q)", got, want)
	}
	if def := Render("# Title", 40, theme.DarkTheme()); def == got {
		t.Fatal("changing Typography.H1 did not change the output")
	}
}

func TestTypographyLinkAndCode(t *testing.T) {
	th := theme.DarkTheme()
	th.Typography.Link = ansi.NewStyle().Foreground(ansi.RGB{R: 9, G: 8, B: 7})
	th.Typography.Code = ansi.NewStyle().Foreground(ansi.RGB{R: 4, G: 5, B: 6})
	if !strings.Contains(Render("[x](http://a)", 40, th), th.Typography.Link.Render("x")) {
		t.Error("link ignores Typography.Link")
	}
	if !strings.Contains(Render("`c`", 40, th), th.Typography.Code.Render("c")) {
		t.Error("code ignores Typography.Code")
	}
}

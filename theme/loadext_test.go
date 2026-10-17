package theme

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestParseGlyphsSpacingBorder(t *testing.T) {
	th, err := Parse(strings.NewReader(`{"base":"nord","glyphs":"ascii","spacing":{"xs":2,"m":5},"border":"double"}`))
	if err != nil {
		t.Fatal(err)
	}
	if th.Glyphs != ASCIIGlyphSet() {
		t.Error("glyphs not ascii")
	}
	if th.Spacing.XS != 2 || th.Spacing.M != 5 || th.Spacing.S != 0 || th.Spacing.L != 0 {
		t.Errorf("spacing %+v; unset fields stay 0 (the default)", th.Spacing)
	}
	if th.Border.TopLeft != "╔" {
		t.Errorf("border %+v", th.Border)
	}
	for name, want := range map[string]string{"thick": "┏", "hidden": " ", "rounded": "╭"} {
		got, err := Parse(strings.NewReader(`{"border":"` + name + `"}`))
		if err != nil || got.Border.TopLeft != want {
			t.Errorf("%s: %v %+v", name, err, got.Border)
		}
	}
	for _, g := range []string{"unicode", "nerd"} {
		if _, err := Parse(strings.NewReader(`{"glyphs":"` + g + `"}`)); err != nil {
			t.Errorf("glyphs %s: %v", g, err)
		}
	}
}

func TestParseStatesAndTypography(t *testing.T) {
	th, err := Parse(strings.NewReader(`{
		"primary": "#ff8800",
		"states": {"focus": {"fg": "#00ff00", "bold": true}, "disabled": {"faint": true}},
		"typography": {"h1": {"fg": "#112233", "bg": "#ffffff", "underline": true}, "code": {"italic": true, "reverse": true, "strike": true}}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := ansi.NewStyle().Foreground(ansi.RGB{G: 255}).Bold(); th.States.Focus != want {
		t.Errorf("focus %v", th.States.Focus)
	}
	if want := ansi.NewStyle().Faint(); th.States.Disabled != want {
		t.Errorf("disabled %v", th.States.Disabled)
	}
	if th.States.Hover == (ansi.Style{}) {
		t.Error("hover was derived from the changed colours and should not be empty")
	}
	if want := ansi.NewStyle().Foreground(ansi.RGB{R: 0x11, G: 0x22, B: 0x33}).Background(ansi.RGB{R: 255, G: 255, B: 255}).Underline(); th.Typography.H1 != want {
		t.Errorf("h1 %v", th.Typography.H1)
	}
	if want := ansi.NewStyle().Italic().Reverse().Strikethrough(); th.Typography.Code != want {
		t.Errorf("code %v", th.Typography.Code)
	}
}

func TestParseComponentTokens(t *testing.T) {
	th, err := Parse(strings.NewReader(`{"components": {"tabs": {"accent": "#ff8800", "text_inverse": "#000"}, "toast": {"error": "#ff0000"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	tabs := th.TokensFor("tabs")
	if tabs.Accent != (ansi.RGB{R: 255, G: 0x88}) || tabs.TextInverse != (ansi.RGB{}) {
		t.Errorf("tabs %+v", tabs)
	}
	if th.TokensFor("toast").Error != (ansi.RGB{R: 255}) {
		t.Error("toast error")
	}
	if th.TokensFor("menu").Accent == (ansi.RGB{R: 255, G: 0x88}) {
		t.Error("a token leaked to another component")
	}
}

func TestParseExtrasErrorsNameTheField(t *testing.T) {
	for in, want := range map[string]string{
		`{"glyphs":"emoji"}`:                        "glyphs",
		`{"spacing":{"m":-1}}`:                      "spacing.m",
		`{"states":{"focus":{"fg":"red"}}}`:         "states.focus.fg",
		`{"typography":{"h2":{"bg":"#12"}}}`:        "typography.h2.bg",
		`{"components":{"nope":{"accent":"#fff"}}}`: `"nope" is not a component`,
		`{"components":{"tabs":{"accent":"blue"}}}`: "components.tabs.accent",
		`{"states":{"focus":{"bogus":true}}}`:       "bogus",
		`{"border":"wavy"}`:                         "double, thick or hidden",
		`{"components":{"tabs":{"colour":"#fff"}}}`: "colour",
	} {
		_, err := Parse(strings.NewReader(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%s) error = %v; want it to mention %q", in, err, want)
		}
	}
}

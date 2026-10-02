package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

const truecolorView = "\x1b[1;38;2;255;0;0mred\x1b[0m \x1b[48;2;0;0;255mblue\x1b[0m"

func TestColorProfileDetectedTrueColorLeavesOutputUnchanged(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1") // output is a regular file, not a TTY
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("TERM", "xterm-256color")
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out))
	if p.ColorProfile() != ansi.TrueColor {
		t.Fatalf("default ColorProfile = %v, want TrueColor", p.ColorProfile())
	}
	p.render()
	if got := string(read()); !strings.Contains(got, truecolorView) {
		t.Errorf("default render altered the view: %q", got)
	}
}

func TestColorProfile256TermDowngradesTruecolor(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("WT_SESSION", "")
	t.Setenv("CLICOLOR_FORCE", "1") // output is a regular file, not a TTY
	t.Setenv("NO_COLOR", "")
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out))
	if p.ColorProfile() != ansi.ANSI256 {
		t.Fatalf("ColorProfile = %v, want ANSI256", p.ColorProfile())
	}
	p.render()
	got := string(read())
	if strings.Contains(got, ";2;255;0;0") || !strings.Contains(got, "38;5;") {
		t.Errorf("truecolor not emitted as 256-colour: %q", got)
	}
}

func TestWithColorProfileRewritesFrames(t *testing.T) {
	cases := []struct {
		profile ansi.Profile
		want    string
	}{
		{ansi.ANSI256, "\x1b[1;38;5;196mred\x1b[0m \x1b[48;5;21mblue\x1b[0m"},
		{ansi.ANSI16, "\x1b[1;91mred\x1b[0m \x1b[44mblue\x1b[0m"},
		{ansi.NoColor, "\x1b[1mred\x1b[0m blue\x1b[0m"},
	}
	for _, c := range cases {
		out, read := captureOutput(t)
		p := NewProgram(staticModel{view: truecolorView}, WithOutput(out), WithColorProfile(c.profile), WithCellRenderer(false))
		if p.ColorProfile() != c.profile {
			t.Errorf("ColorProfile() = %v, want %v", p.ColorProfile(), c.profile)
		}
		p.render()
		if got := string(read()); !strings.Contains(got, c.want) {
			t.Errorf("profile %v: output %q does not contain %q", c.profile, got, c.want)
		}
	}
}

func TestWithColorProfileAppliesToPrintlnAndKeepsDiffing(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a"}, WithOutput(out), WithColorProfile(ansi.NoColor))
	p.render()
	p.printLines("\x1b[31mnote\x1b[0m", out)
	if got := string(read()); strings.Contains(got, "\x1b[31m") || !strings.Contains(got, "note\x1b[0m") {
		t.Errorf("Println text not downgraded: %q", got)
	}
	// Downgraded frames still diff: an identical View writes no content.
	p.model = staticModel{view: "\x1b[31mzz\x1b[0m"}
	p.render()
	before := len(read())
	p.render()
	if added := string(read()[before:]); strings.Contains(added, "zz") {
		t.Errorf("unchanged downgraded frame was rewritten: %q", added)
	}
}

func TestAccessibleModeStillStripsEverythingWithColorProfile(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: truecolorView}, WithOutput(out), WithAccessible(true), WithColorProfile(ansi.ANSI16))
	p.render()
	if got := string(read()); got != "red blue\r\n" {
		t.Errorf("output = %q, want plain text", got)
	}
}

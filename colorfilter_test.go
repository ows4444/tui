package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

func filterProgram(view string, out *bytes.Buffer, opts ...ProgramOption) *Program {
	opts = append([]ProgramOption{WithOutput(out), WithColorProfile(ansi.TrueColor), WithAltScreen(false)}, opts...)
	p := NewProgram(staticModel{view: view}, opts...)
	p.width, p.height = 30, 5
	return p
}

func sgr(c ansi.RGB) string { return fmt.Sprintf("38;2;%d;%d;%d", c.R, c.G, c.B) }

// WithColorFilter rewrites the colours of a frame, whatever depth the view
// wrote them in, before the profile is applied.
func TestColorFilterRewritesFrameColours(t *testing.T) {
	for name, tc := range map[string]struct {
		view string
		from ansi.RGB // the colour the view means, as the filter sees it
	}{
		"basic":     {"\x1b[31mhi\x1b[0m", ansi.Red.RGB()}, // the xterm palette's red, not 255,0,0
		"256":       {"\x1b[38;5;196mhi\x1b[0m", ansi.RGB{R: 255}},
		"truecolor": {"\x1b[38;2;255;0;0mhi\x1b[0m", ansi.RGB{R: 255}},
		"colon":     {"\x1b[38:2::255:0:0mhi\x1b[0m", ansi.RGB{R: 255}},
		"bg":        {"\x1b[41mhi\x1b[0m", ansi.Red.RGB()},
	} {
		var out bytes.Buffer
		p := filterProgram(tc.view, &out, WithColorFilter(ansi.Grayscale))
		p.renderIfChanged()
		g := ansi.Grayscale(tc.from)
		want := fmt.Sprintf("%s;2;%d;%d;%d", map[bool]string{false: "38", true: "48"}[name == "bg"], g.R, g.G, g.B)
		got := out.String()
		if !strings.Contains(got, want) || strings.Contains(got, "[31m") || strings.Contains(got, "[41m") ||
			strings.Contains(got, "38;5;196") || strings.Contains(got, "38;2;255;0;0") || strings.Contains(got, "38:2") {
			t.Errorf("%s: frame = %q, want %s and no original colour", name, got, want)
		}
	}
}

func TestColorFilterNilAndAbsentChangeNothing(t *testing.T) {
	view := "\x1b[38;2;255;0;0mhi\x1b[0m"
	var a, b bytes.Buffer
	pa := filterProgram(view, &a)
	pa.renderIfChanged()
	pb := filterProgram(view, &b, WithColorFilter(nil))
	pb.renderIfChanged()
	if a.String() != b.String() {
		t.Errorf("a nil filter changed the frame:\n%q\n%q", a.String(), b.String())
	}
	if !strings.Contains(a.String(), "38;2;255;0;0") {
		t.Errorf("without a filter the colour was lost: %q", a.String())
	}
}

// The profile is applied after the filter: a 16-colour terminal gets the
// nearest basic colour of the filtered colour, not of the original.
func TestColorFilterRunsBeforeTheProfile(t *testing.T) {
	var out bytes.Buffer
	p := filterProgram("\x1b[38;2;255;0;0mhi\x1b[0m", &out,
		WithColorFilter(func(ansi.RGB) ansi.RGB { return ansi.RGB{B: 255} }), WithColorProfile(ansi.ANSI16))
	p.renderIfChanged()
	if got := out.String(); !strings.Contains(got, "94m") && !strings.Contains(got, "34m") {
		t.Errorf("frame = %q, want blue (34 or 94)", got)
	}
}

func TestColorFilterAppliesToCommittedText(t *testing.T) {
	var out bytes.Buffer
	p := filterProgram("x", &out, WithColorFilter(ansi.Grayscale))
	p.printLines("\x1b[31mcommitted\x1b[0m", &out)
	if got := out.String(); !strings.Contains(got, sgr(ansi.Grayscale(ansi.Red.RGB()))) || strings.Contains(got, "[31m") {
		t.Errorf("committed text = %q", got)
	}
}

// A root that draws cells is drawn from its View string while a filter is set,
// since the filter works on the string.
func TestColorFilterTurnsOffDirectDraw(t *testing.T) {
	m := &drawModel{draw: func(*cellbuf.Buffer, cellbuf.Rect) {}}
	var out bytes.Buffer
	p := cdProgram(m, &out)
	if _, ok := p.directDraw(); !ok {
		t.Fatal("setup: direct draw should be on without a filter")
	}
	p.colorFilter = ansi.Grayscale
	if _, ok := p.directDraw(); ok {
		t.Error("direct draw stayed on under a colour filter")
	}
}

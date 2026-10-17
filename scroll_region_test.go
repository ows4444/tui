package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

const srW, srH = 60, 24

// srView is a log of srH lines starting at entry from.
func srView(from int) string {
	var lines []string
	for i := 0; i < srH; i++ {
		l := fmt.Sprintf("%05d %s", from+i, strings.Repeat(string(rune('a'+(from+i)%26)), 40+(from+i)%7))
		lines = append(lines, l+strings.Repeat(" ", srW-len(l)))
	}
	return strings.Join(lines, "\n")
}

// srProgram is a cell-rendering Program drawing to out; alt picks the alternate
// screen, env the environment.
func srProgram(out *bytes.Buffer, alt bool, env ...string) *Program {
	opts := []ProgramOption{WithOutput(out), WithCellRenderer(true), WithAltScreen(alt), WithColorProfile(ansi.TrueColor)}
	if env != nil {
		opts = append(opts, WithEnvironment(env))
	}
	p := NewProgram(staticModel{view: srView(0)}, opts...)
	p.width, p.height = srW, srH
	return p
}

func srFrames(p *Program, out *bytes.Buffer, from ...int) (last string) {
	for _, f := range from {
		out.Reset()
		p.model = staticModel{view: srView(f)}
		p.render()
		last = out.String()
	}
	return last
}

func TestAScrollingViewInTheAlternateScreenIsScrolledByTheTerminal(t *testing.T) {
	var out bytes.Buffer
	p := srProgram(&out, true)
	last := srFrames(p, &out, 0, 1)
	if !strings.Contains(last, "\x1b[1;24r\x1b[1S\x1b[r") {
		t.Fatalf("no scroll in %q", last[:min(80, len(last))])
	}
	if len(last) > 400 {
		t.Errorf("%d bytes for a one-line scroll", len(last))
	}
}

func TestTheScreenIsRightAfterManyScrolls(t *testing.T) {
	var out, ref bytes.Buffer
	scr := vtscreen.NewScreen(srW, srH)
	p := srProgram(&out, true)
	q := srProgram(&ref, true, "TUI_NO_SCROLL_REGIONS=1")
	for step := 0; step < 30; step++ {
		out.Reset()
		p.model = staticModel{view: srView(step * 2)}
		p.render()
		scr.Write(out.Bytes())
	}
	refScr := vtscreen.NewScreen(srW, srH)
	for step := 0; step < 30; step++ {
		ref.Reset()
		q.model = staticModel{view: srView(step * 2)}
		q.render()
		refScr.Write(ref.Bytes())
	}
	for y := 0; y < srH; y++ {
		for x := 0; x < srW; x++ {
			if scr.Cell(x, y) != refScr.Cell(x, y) {
				t.Fatalf("cell (%d,%d) differs:\n%s\nwant\n%s", x, y, strings.Join(scr.Lines(), "\n"), strings.Join(refScr.Lines(), "\n"))
			}
		}
	}
}

func TestScrollRegionsAreOffInlineAndWhenTheEnvironmentSaysSo(t *testing.T) {
	for name, tc := range map[string]struct {
		alt bool
		env []string
	}{
		"inline":      {false, nil},
		"environment": {true, []string{"TUI_NO_SCROLL_REGIONS=1"}},
	} {
		var out bytes.Buffer
		p := srProgram(&out, tc.alt, tc.env...)
		last := srFrames(p, &out, 0, 1)
		if strings.Contains(last, "r\x1b[") && strings.Contains(last, "\x1b[1S") || strings.Contains(last, "\x1b[r") {
			t.Errorf("%s: scrolled: %q", name, last[:min(80, len(last))])
		}
		if last == "" {
			t.Errorf("%s: nothing was drawn", name)
		}
	}
}

func TestTheEnvironmentIsReadOnce(t *testing.T) {
	var out bytes.Buffer
	p := srProgram(&out, true, "TUI_NO_SCROLL_REGIONS=1")
	if p.scrollRegionsOn() || p.scrollEnv != 2 {
		t.Errorf("scrollRegionsOn = true with the variable set")
	}
	p2 := srProgram(&out, true)
	if !p2.scrollRegionsOn() || p2.scrollEnv != 1 {
		t.Error("scrollRegionsOn = false without the variable")
	}
	p3 := srProgram(&out, true)
	p3.height = 0
	if p3.scrollRegionsOn() {
		t.Error("scroll regions on with an unknown height")
	}
}

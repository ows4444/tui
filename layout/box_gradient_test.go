package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func drawBox(b Box, content string) string {
	return Draw(BoxNode(b, Block(content)), Unconstrained())
}

func TestBorderGradientChangesNoDimensionsOrText(t *testing.T) {
	plain := NewBox().Border(RoundedBorder()).PaddingAll(1).Title("grad", AlignCenter)
	grad := plain.BorderGradient(ansi.RGB{R: 255}, ansi.RGB{B: 255})
	a, b := drawBox(plain, "hello\nworld"), drawBox(grad, "hello\nworld")
	if ansi.StripANSI(a) != ansi.StripANSI(b) {
		t.Errorf("a gradient changed the text:\n%s\nvs\n%s", ansi.StripANSI(a), ansi.StripANSI(b))
	}
	for i, line := range strings.Split(b, "\n") {
		if ansi.Width(line) != ansi.Width(strings.Split(a, "\n")[i]) {
			t.Errorf("line %d width changed", i)
		}
		if strings.Count(line, "\x1b[0m") < strings.Count(line, "\x1b[")/2 {
			t.Errorf("line %d is not self-contained: %q", i, line)
		}
	}
}

func TestBorderGradientRunsFromStartThroughEndAndBack(t *testing.T) {
	from, to := ansi.RGB{R: 200}, ansi.RGB{B: 200}
	r := borderRing{from: from, to: to, topW: 10, botW: 10, side: 3}
	if got := r.colourAt(0); got != from {
		t.Errorf("start = %v, want %v", got, from)
	}
	half := r.colourAt(r.total() / 2)
	if half != to {
		t.Errorf("half way = %v, want %v", half, to)
	}
	// The cell before the start (the last on the loop) is close to from: no seam.
	last := r.colourAt(r.total() - 1)
	if d := int(last.R) - int(from.R); d < -25 || last.B > 25 {
		t.Errorf("last cell %v is not near the start colour %v", last, from)
	}
	if (borderRing{}).on() {
		t.Error("an empty ring is on")
	}
}

func TestBorderGradientColoursCornersAndSides(t *testing.T) {
	out := drawBox(NewBox().Border(NormalBorder()).BorderGradient(ansi.RGB{R: 255}, ansi.RGB{B: 255}), "ab")
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[0], "\x1b[38;2;255;0;0m") {
		t.Errorf("the top-left corner is not the start colour: %q", lines[0])
	}
	if len(lines) != 3 {
		t.Fatalf("%d lines", len(lines))
	}
	// Left and right side cells of the same row have different colours (they are at
	// different places on the loop).
	l, r := lines[1], lines[1][strings.LastIndex(lines[1], "\x1b[38;2;"):]
	if strings.HasPrefix(l, r[:12]) && strings.Count(out, "\x1b[38;2;") < 4 {
		t.Errorf("side cells are not coloured separately: %q", lines[1])
	}
	// No border, no gradient; a gradient wins over BorderColor.
	if strings.Contains(drawBox(NewBox().BorderGradient(ansi.RGB{R: 1}, ansi.RGB{B: 1}), "x"), "\x1b[") {
		t.Error("a box without a border was coloured")
	}
	both := drawBox(NewBox().Border(NormalBorder()).BorderColor(ansi.RGB{G: 255}).BorderGradient(ansi.RGB{R: 255}, ansi.RGB{B: 255}), "x")
	if strings.Contains(both, "\x1b[38;2;0;255;0m") {
		t.Error("BorderColor was used under a gradient")
	}
}

func TestBorderGradientOnABoxWithSomeSidesOff(t *testing.T) {
	b := NewBox().Border(NormalBorder()).BorderSides(true, false, true, false).BorderGradient(ansi.RGB{R: 255}, ansi.RGB{B: 255})
	out := drawBox(b, "abc")
	if got := ansi.StripANSI(out); got != "───\nabc\n───" {
		t.Errorf("top and bottom only = %q", got)
	}
}

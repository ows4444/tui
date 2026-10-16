package cellbuf

import (
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestSetTextWritesEachSpanInItsStyle(t *testing.T) {
	bold := ansi.NewStyle().Bold()
	text := ansi.NewText("ab", ansi.Style{}).Append("cd", bold).Append("世", ansi.NewStyle().Foreground(ansi.Red))
	b := New(10, 2)
	if n := b.SetText(1, 0, text); n != 6 {
		t.Errorf("wrote %d columns, want 6", n)
	}
	for x, want := range map[int]string{1: "a", 2: "b", 3: "c", 4: "d", 5: "世"} {
		if got := b.At(x, 0).Cluster; got != want {
			t.Errorf("cell %d = %q, want %q", x, got, want)
		}
	}
	if b.Style(b.At(1, 0).Style) != (Style{}) {
		t.Error("an unstyled span got a style")
	}
	if b.Style(b.At(3, 0).Style).Attrs&AttrBold == 0 {
		t.Error("the bold span lost its attribute")
	}
	if b.Style(b.At(5, 0).Style).FG == 0 {
		t.Error("the red span lost its colour")
	}
	// The result is the same as drawing the rendered string through SetStyled.
	c := New(10, 2)
	c.SetStyled(1, 0, text.String())
	for x := 0; x < 10; x++ {
		if b.At(x, 0).Cluster != c.At(x, 0).Cluster || b.Style(b.At(x, 0).Style) != c.Style(c.At(x, 0).Style) {
			t.Errorf("cell %d differs from SetStyled", x)
		}
	}
}

func TestSetTextStopsAtANewlineAndClips(t *testing.T) {
	b := New(4, 2)
	n := b.SetText(0, 0, ansi.NewText("ab\ncd", ansi.Style{}))
	if n != 2 || b.At(0, 1).Cluster == "c" {
		t.Errorf("a newline did not end the line: wrote %d", n)
	}
	b.SetText(2, 1, ansi.NewText("wxyz", ansi.Style{}))
	if b.At(3, 1).Cluster != "x" {
		t.Errorf("clipped text = %q", b.Lines()[1])
	}
	b.SetText(0, 5, ansi.NewText("off", ansi.Style{})) // outside: ignored, no panic
}

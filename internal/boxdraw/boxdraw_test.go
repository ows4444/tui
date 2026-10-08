package boxdraw

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// Draw must give exactly what layout.Box.Render gave the components that used
// to call it, so moving them here changes no output.
func TestDrawMatchesBoxRender(t *testing.T) {
	red := ansi.NewStyle().Foreground(ansi.Red)
	contents := []string{
		"", "a", "hello world", "one\ntwo\nthree", "a long line that is clipped\nx", "trail\n", "\nlead", "a\n\nb",
		red.Render("styled"), red.Render("styled") + "\nplain", "x " + red.Render("mid") + " y\nsecond", red.Bold().Render("two\nlines"),
		"日本語\nab",
	}
	borders := []layout.Border{{}, layout.RoundedBorder(), layout.ASCIIBorder()}
	for _, border := range borders {
		for _, colour := range []ansi.Color{nil, ansi.Red} {
			for _, pad := range []int{0, 1, 2} {
				for _, width := range []int{-1, 0, 1, 3, 8, 20} {
					for _, content := range contents {
						old := layout.NewBox().Border(border).BorderColor(colour).PaddingAll(pad)
						if width > 0 {
							old = old.Width(width)
						}
						//lint:ignore SA1019 the deprecated method is the reference this test compares against
						want := old.Render(content)
						got := Draw(layout.NewBox().BorderColor(colour), border, pad, width, content)
						if got != want {
							t.Errorf("border %q colour %v pad %d width %d content %q:\n got %q\nwant %q",
								border.TopLeft, colour != nil, pad, width, content, got, want)
						}
					}
				}
			}
		}
	}
}

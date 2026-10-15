package boxdraw

import (
	"testing"

	"github.com/ows4444/tui/layout"
)

// Draw frames content the way layout.Box.Render did before it was made
// unexported: the same cells for the components that moved here. The cases are
// the output of the old method for the same arguments.
func TestDrawFramesContent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		border layout.Border
		pad    int
		width  int
		in     string
		want   string
	}{
		{"auto width", layout.RoundedBorder(), 0, 0, "hi", "╭──╮\n│hi│\n╰──╯"},
		{"padding on every side", layout.RoundedBorder(), 1, 0, "a\nbb", "╭────╮\n│    │\n│ a  │\n│ bb │\n│    │\n╰────╯"},
		{"no border pads to width", layout.Border{}, 0, 5, "ab", "ab   "},
		{"empty content keeps one row", layout.ASCIIBorder(), 0, 0, "", "++\n||\n++"},
		{"fixed width clips and pads", layout.RoundedBorder(), 0, 3, "toolong\nx", "╭───╮\n│too│\n│x  │\n╰───╯"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Draw(layout.NewBox(), tc.border, tc.pad, tc.width, tc.in); got != tc.want {
				t.Errorf("Draw = %q, want %q", got, tc.want)
			}
		})
	}
}

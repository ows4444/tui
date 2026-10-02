package layout

import "testing"

// TestExistingExportedAPIUnchanged pins the signatures of the string-based
// layout API that remains after the v1.0 removal of the joins, Grid, FlexRow
// and GridFlex. Each assignment fails to compile if a signature changes or a
// name is removed, so a breaking edit to package layout cannot pass the test
// suite. (New exported identifiers, like Node, are allowed.)
func TestExistingExportedAPIUnchanged(t *testing.T) {
	var (
		_ func() Box                            = NewBox
		_ func(Box, int, int, int, int) Box     = Box.Padding
		_ func(Box, int) Box                    = Box.PaddingAll
		_ func(Box, Border) Box                 = Box.Border
		_ func(Box, int) Box                    = Box.Width
		_ func(Box, string) string              = Box.Render
		_ func(int, Justify, ...string) string  = JoinHorizontalJustify
		_ func(string, string, int, int) string = Overlay
	)
	_ = []Border{NormalBorder(), RoundedBorder(), DoubleBorder(), ThickBorder(), ASCIIBorder()}
	_ = []Align{AlignStart, AlignCenter, AlignEnd}
}

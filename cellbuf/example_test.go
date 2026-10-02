package cellbuf_test

import (
	"fmt"

	"github.com/ows4444/tui/cellbuf"
)

// Draw into a Buffer, then convert it to a styled string for a View.
func ExampleBuffer() {
	b := cellbuf.New(6, 2)
	bold := b.StyleID(cellbuf.Style{Attrs: cellbuf.AttrBold})
	b.SetString(0, 0, "hi", bold)
	b.Sub(cellbuf.Rect{X: 3, Y: 1, W: 3, H: 1}).Fill(cellbuf.Rect{W: 3, H: 1}, "=", 0)
	fmt.Printf("%q\n", b.Lines())
	// Output: ["\x1b[1mhi\x1b[0m    " "   ==="]
}

// A wide cluster is never split at the last column.
func ExampleBuffer_SetString() {
	b := cellbuf.New(4, 1)
	b.SetString(0, 0, "a日日", 0)
	fmt.Printf("%q\n", b.String())
	// Output: "a日 "
}

// Parse and String convert between styled strings and cells.
func ExampleParse() {
	b, err := cellbuf.Parse("\x1b[31mred\x1b[0m")
	fmt.Println(b.Width(), b.Style(b.At(0, 0).Style).FG == cellbuf.Basic(1), err)
	fmt.Printf("%q\n", b.String())
	// Output:
	// 3 true <nil>
	// "\x1b[31mred\x1b[0m"
}

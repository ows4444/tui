package imageview_test

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	"github.com/ows4444/tui/imageview"
)

// An imageview draws a PNG with the terminal's graphics protocol; its alt text
// is what shows when images are unavailable.
func Example() {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		fmt.Println(err)
		return
	}
	m := imageview.New(buf.Bytes(), 4, 2, "a 2x2 square")
	fmt.Println(m.View() != "")
	// Output:
	// true
}

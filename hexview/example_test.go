package hexview_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hexview"
)

// A hex view shows an offset, the bytes in hex and an ASCII gutter.
func Example() {
	m := hexview.New([]byte("hello, hex view!\nabc"))
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// 0000  68 65 6c 6c 6f 2c 20 68  65 78 20 76 69 65 77 21  |hello, hex view!|
	// 0010  0a 61 62 63                                       |.abc|
}

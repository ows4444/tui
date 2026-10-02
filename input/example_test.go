package input_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/input"
)

// A Reader turns the raw bytes of a terminal in raw mode into events. It is a
// pure decoder over an io.Reader, so it needs no TTY.
func ExampleReader() {
	rd := input.NewReader(strings.NewReader("a\x1b[A\x1b[1;5C"))
	for {
		ev, err := rd.ReadEvent()
		if err != nil {
			break
		}
		fmt.Println(ev.(input.Key))
	}
	// Output:
	// a
	// up
	// ctrl+right
}

func ExampleReader_SetEscTimeout() {
	rd := input.NewReader(strings.NewReader("\x1b"))
	rd.SetEscTimeout(0) // a bare ESC with nothing buffered is Escape, with no wait
	ev, _ := rd.ReadEvent()
	fmt.Println(ev.(input.Key))
	// Output: esc
}

package term_test

import (
	"fmt"

	"github.com/ows4444/tui/term"
)

// memConsole is an in-memory console, standing in for the Windows API.
type memConsole struct{ mode uint32 }

func (c *memConsole) Mode(int) (uint32, error) { return c.mode, nil }
func (c *memConsole) SetMode(_ int, m uint32) error {
	c.mode = m
	return nil
}

func ExampleIsTerminal() {
	// A descriptor that is not open is never a terminal, so callers can ask
	// before switching to raw mode.
	fmt.Println(term.IsTerminal(-1))
	// Output: false
}

func ExampleEnableOutputVTOn() {
	c := &memConsole{mode: 0x3}
	restore, err := term.EnableOutputVTOn(c, 1)
	fmt.Printf("%#x %v\n", c.mode, err)
	_ = restore()
	fmt.Printf("%#x\n", c.mode)
	// Output:
	// 0x7 <nil>
	// 0x3
}

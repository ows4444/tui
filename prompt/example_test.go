package prompt_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/prompt"
)

// Ask one question and use the answer. Here the keys come from a string and
// the output is discarded; a real program passes no options.
func Example() {
	keys := tui.WithInput(strings.NewReader("Ada\r"))
	out := tui.WithOutput(io.Discard)

	name, err := prompt.Input("Your name:", prompt.Config{Program: []tui.ProgramOption{keys, out}})
	fmt.Println(name, err)
	// Output: Ada <nil>
}

// Select returns the index of the choice picked; typing narrows the list.
func ExampleSelect() {
	keys := tui.WithInput(strings.NewReader("ap\r"))
	out := tui.WithOutput(io.Discard)

	i, _ := prompt.Select("Region:", []string{"eu", "us", "ap"}, prompt.Config{Program: []tui.ProgramOption{keys, out}})
	fmt.Println(i)
	// Output: 2
}

package prompt_test

import (
	"context"
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

// Track runs fn for each item with a bar that advances after each.
func ExampleTrack() {
	files := []string{"a.txt", "b.txt"}
	pr, pw := io.Pipe() // a terminal that stays open and quiet
	defer pw.Close()
	err := prompt.Track("Copying", files, func(_ context.Context, name string) error {
		return nil // copy name
	}, prompt.Config{Program: []tui.ProgramOption{tui.WithOutput(io.Discard), tui.WithInput(pr)}})
	fmt.Println(err)
	// Output: <nil>
}

// Choose is Select for values of any type, with groups.
func ExampleChoose() {
	type region struct{ id string }
	pr, pw := io.Pipe()
	defer pw.Close()
	go func() { _, _ = io.WriteString(pw, "\x1b[B\r") }() // Down, Enter
	r, err := prompt.Choose("Region", []prompt.Choice[region]{
		{Label: "Frankfurt", Value: region{"eu-central"}, Group: "Europe"},
		{Label: "Oregon", Value: region{"us-west"}, Group: "Americas"},
	}, prompt.Config{Program: []tui.ProgramOption{tui.WithOutput(io.Discard), tui.WithInput(pr), tui.WithReducedMotion(true)}})
	fmt.Println(r.id, err)
	// Output: us-west <nil>
}

// Int, Float, Duration and Path return a typed answer; text that does not parse
// is refused and the question stays open. Parse does the same for a type of
// your own.
func ExampleInt() {
	// "abc" is refused, Ctrl+U clears it, and 42 is taken.
	keys := tui.WithInput(strings.NewReader("abc\r\x1542\r"))
	out := tui.WithOutput(io.Discard)

	n, err := prompt.Int("Count:", prompt.Config{Program: []tui.ProgramOption{keys, out}})
	fmt.Println(n, err)
	// Output: 42 <nil>
}

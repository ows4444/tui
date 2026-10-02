package tuitest_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

type typer struct{ text string }

func (typer) Init() tui.Cmd { return nil }

func (e typer) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		e.text += k.Text
	}
	return e, nil
}

func (e typer) View() string { return "> " + e.text }

// A Session runs a model in a real Program against an in-memory terminal, so
// a test reads what the user would see.
func Example() {
	s := tuitest.New(typer{}, 20, 3)
	defer s.Close()
	s.Keys("h", "i")
	fmt.Println(strings.TrimRight(s.Screen()[0], " "))
	fmt.Printf("%q\n", s.Cell(2, 0).Grapheme)
	// Output:
	// > hi
	// "h"
}

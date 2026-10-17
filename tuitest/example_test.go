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

// SVG draws the screen as an image, for documentation or a pull request.
func ExampleSession_SVG() {
	s := tuitest.New(typer{}, 12, 2)
	defer s.Close()
	s.Keys("h", "i")
	svg := s.SVG(tuitest.SVGOptions{Title: "typer"})
	fmt.Println(strings.HasPrefix(svg, "<svg "), strings.Contains(svg, ">&gt;hi</text>") || strings.Contains(svg, ">&gt;</text>"))
	// Output: true true
}

// Find and ClickText locate text on the screen, so a test does not depend on
// where a layout draws it.
func ExampleSession_Find() {
	s := tuitest.New(typer{}, 20, 3)
	defer s.Close()
	s.Keys("h", "i")
	x, y, ok := s.Find("hi")
	fmt.Println(x, y, ok)
	// Output: 2 0 true
}

// SVGOf draws styled text, such as a running program's tui.Screenshot.ANSI, as
// an image.
func ExampleSVGOf() {
	svg := tuitest.SVGOf("\x1b[1mhello\x1b[0m", 10, 2, tuitest.SVGOptions{TrimRows: true})
	fmt.Println(strings.HasPrefix(svg, "<svg "), strings.Contains(svg, ">hello</text>"))
	// Output: true true
}

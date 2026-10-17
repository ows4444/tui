package textarea_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/textarea"
)

// A textarea edits several lines; Enter inserts a newline.
func Example() {
	m := textarea.New()
	m.Focus()
	for _, k := range []tui.Key{
		{Type: tui.KeyRunes, Text: "hi"},
		{Type: tui.KeyEnter},
		{Type: tui.KeyRunes, Text: "there"},
	} {
		m, _ = m.Update(k)
	}
	fmt.Printf("%q\n", m.Value())
	// Output:
	// "hi\nthere"
}

// ShowLineNumbers draws a number column; Line and Column say where the cursor
// is, and InsertString adds text at it as one undo step.
func ExampleModel_ShowLineNumbers() {
	m := textarea.New()
	m.ShowLineNumbers = true
	m.SetValue("first\nsecond")
	m.SetCursorPosition(0, 5)
	m.InsertString("!")
	fmt.Println(m.Line(), m.Column(), m.LineCount())
	fmt.Println(ansi.StripANSI(m.View()))
	// Output:
	// 0 6 2
	// 1 first!
	// 2 second
}

// Find selects a match; ReplaceAll replaces every one in a single undo step.
func ExampleModel_ReplaceAll() {
	m := textarea.New()
	m.SetValue("one cat, two Cats")
	fmt.Println(m.Find("cat"), m.SelectedText())
	fmt.Println(m.ReplaceAll("cat", "dog"), m.Value())
	// Output:
	// 2 cat
	// 2 one dog, two dogs
}

// ShowWhitespace draws each space as a mark, which shows trailing spaces.
func ExampleModel_showWhitespace() {
	m := textarea.New()
	m.SetValue("a b  ")
	m.ShowWhitespace = true
	fmt.Println(ansi.StripANSI(m.View()))
	// Output: a·b··
}

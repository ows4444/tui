package ansi_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
)

// Width counts terminal columns, not bytes or runes: escape sequences take
// none and a wide character takes two.
func ExampleWidth() {
	fmt.Println(ansi.Width("abc"), ansi.Width(ansi.NewStyle().Bold().Render("abc")), ansi.Width("日本"))
	// Output: 3 3 4
}

func ExampleWrap() {
	fmt.Println(ansi.Wrap("the quick brown fox", 9))
	// Output:
	// the quick
	// brown fox
}

func ExampleTruncate() {
	fmt.Println(ansi.Truncate("hello world", 5))
	// Output: hello
}

// Text holds styled text as spans, so it can be measured and wrapped without
// parsing escape sequences, and each piece keeps its style across a break.
func ExampleText_Wrap() {
	bold := ansi.NewStyle().Bold()
	t := ansi.NewText("plain ", ansi.Style{}).Append("bold words here", bold)
	for _, line := range t.Wrap(10) {
		fmt.Printf("%q %d\n", line.Plain(), line.Width())
	}
	// Output:
	// "plain bold" 10
	// "words here" 10
}

package clipboard_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/clipboard"
)

// Enter copies the text by writing an OSC 52 sequence; here Write is replaced
// to show what would reach the terminal.
func Example() {
	m := clipboard.New("secret-token", "Copy")
	var sent string
	m.Write = func(s string) (int, error) { sent = s; return len(s), nil }
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println(m.Copied(), strings.HasPrefix(sent, "\x1b]52;"))
	// Output:
	// true true
}

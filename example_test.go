package tui_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ows4444/tui"
)

type counter struct{ n int }

func (c counter) Init() tui.Cmd { return nil }

func (c counter) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes {
		switch k.Text {
		case "+":
			c.n++
		case "q":
			return c, tui.Quit()
		}
	}
	return c, nil
}

func (c counter) View() string { return fmt.Sprintf("count: %d", c.n) }

// A Program runs a Model. Here the keys come from a string and the output is
// discarded, so no terminal is needed; Run returns the final model.
func ExampleNewProgram() {
	p := tui.NewProgram(counter{},
		tui.WithInput(strings.NewReader("++q")),
		tui.WithOutput(io.Discard),
	)
	final, err := p.Run()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(final.View())
	// Output: count: 2
}

// Tick returns a Cmd that waits and then produces a Msg. A Program runs it on
// its own goroutine with its context; RunCmd runs it the same way for a test or
// an example.
func ExampleTick() {
	cmd := tui.Tick(time.Millisecond, func(time.Time) tui.Msg { return "tick" })
	fmt.Println(tui.RunCmd(context.Background(), cmd))
	// Output: tick
}

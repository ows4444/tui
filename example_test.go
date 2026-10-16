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

// A model states the terminal modes it wants beside its View: ViewState is
// read before each frame, and a nil field leaves that mode alone.
func ExampleViewStater() {
	var m fullscreenModel
	st := m.ViewState()
	fmt.Println(*st.AltScreen, *st.Title, st.Mouse == nil)
	// Output: true editing notes.txt true
}

type fullscreenModel struct{}

func (fullscreenModel) Init() tui.Cmd                         { return nil }
func (m fullscreenModel) Update(tui.Msg) (tui.Model, tui.Cmd) { return m, nil }
func (fullscreenModel) View() string                          { return "" }
func (fullscreenModel) ViewState() tui.ViewState {
	return tui.ViewState{AltScreen: new(true), Title: new("editing notes.txt")}
}

// A search box starts one task per keystroke under the same key: Tasks
// cancels the run before it, so only the newest result is current.
func ExampleTasks() {
	tasks := tui.NewTasks()
	cmd := tasks.Start("search", func(ctx context.Context) tui.Msg {
		return "results for go"
	})
	msg := tui.RunCmd(context.Background(), cmd)
	done := msg.(tui.TaskDoneMsg)
	fmt.Println(done.Key, done.Msg, done.Canceled, tasks.Running("search"))
	// Output: search results for go false false
}

// A parent tags the messages of a child's Cmd once with Map, and tells them
// apart in its own Update.
func ExampleMap() {
	type fromSidebar struct{ tui.Msg }
	child := func() tui.Msg { return "selected" }
	cmd := tui.Map(child, func(m tui.Msg) tui.Msg { return fromSidebar{m} })
	fmt.Println(cmd().(fromSidebar).Msg)
	// Output: selected
}

# Tutorial: your first program

This takes you from an empty directory to a small task list that runs in a
terminal: you type a task, press enter, move through the list and mark tasks
done. It takes about fifteen minutes. You need Go 1.26 or later and a
terminal.

The finished program is [examples/tutorial](../examples/tutorial), with the
tests this page ends on. If a step does not behave as described, compare with
it.

## 1. A program that does nothing

Make a directory and a module, and add tui:

```console
$ mkdir tasks && cd tasks
$ go mod init example.com/tasks
$ go get github.com/ows4444/tui
```

Put this in `main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

type model struct{}

func (model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	return m, nil
}

func (model) View() string { return "Hello, tui. Press ctrl+c to quit." }

func main() {
	if _, err := tui.NewProgram(model{}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Run it with `go run .`. You see the text; ctrl+c leaves.

A tui program is a value with three methods, the Elm architecture:

- `Init` returns a `Cmd` to run first, or nil.
- `Update` receives one message (`Msg`), such as a key press or a resize, and
  returns the next model and a `Cmd` for work to do next.
- `View` returns what the screen shows, as a string. The Program draws it and
  only writes the cells that changed.

`tui.Quit()` is a `Cmd` that ends the program. A model never touches the
terminal itself.

## 2. State and keys

State lives in the model, and `Update` is the only place it changes. Replace
the model with a counter:

```go
type model struct {
	count int
}

func (model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok {
		switch k.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyUp:
			m.count++
		case tui.KeyDown:
			m.count--
		}
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("Count: %d\n\nup/down change it, ctrl+c quits", m.count)
}
```

`Update` has a value receiver and returns the changed copy; nothing is shared.
Run it and press the arrow keys. Every key press is one `Update`, then one
`View`.

## 3. A text field

You rarely write a text field yourself. `textinput` is a component: a `Model`
of its own that you hold as a field, hand messages to, and draw inside your
`View`. Replace `main.go` with the whole program:

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/textinput"
)

type task struct {
	title string
	done  bool
}

type model struct {
	input  textinput.Model
	tasks  []task
	cursor int
	width  int
}

func newModel() model {
	in := textinput.New()
	in.Prompt = "> "
	in.Placeholder = "What needs doing?"
	in.Width = 30
	in.Focus() // the blink Cmd comes back from Init
	return model{input: in}
}

func (m model) Init() tui.Cmd { return m.input.Focus() }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width = msg.Width
		return m, nil
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyEnter:
			if title := strings.TrimSpace(m.input.Value()); title != "" {
				m.tasks = append(m.tasks, task{title: title})
				m.input.Reset()
			}
			return m, nil
		case tui.KeyUp:
			m.cursor = max(m.cursor-1, 0)
			return m, nil
		case tui.KeyDown:
			m.cursor = max(min(m.cursor+1, len(m.tasks)-1), 0)
			return m, nil
		case tui.KeyTab:
			if m.cursor < len(m.tasks) {
				m.tasks[m.cursor].done = !m.tasks[m.cursor].done
			}
			return m, nil
		}
	}
	var cmd tui.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

var (
	titleStyle = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	doneStyle  = ansi.NewStyle().Faint().Strikethrough()
	helpStyle  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Tasks"))
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render(m.fit("Nothing yet. Type a task and press enter.")))
		b.WriteString("\n")
	}
	for i, t := range m.tasks {
		cursor, box, title := "  ", "[ ]", t.title
		if i == m.cursor {
			cursor = "> "
		}
		if t.done {
			box, title = "[x]", doneStyle.Render(t.title)
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, box, title)
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.fit("enter add   up/down move   tab done   ctrl+c quit")))
	return b.String()
}

// fit wraps s to the terminal's width once the size is known.
func (m model) fit(s string) string {
	if m.width > 0 {
		return ansi.Wrap(s, m.width)
	}
	return s
}

func main() {
	if _, err := tui.NewProgram(newModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Run it. Type "write the tutorial", press enter, and the task appears. Type
another, press up and down to move the `>` marker, and tab to mark one done.

What is new here:

- **A component in your model.** `input textinput.Model` is a field. `Update`
  gives it every message it did not handle itself: `m.input, cmd =
  m.input.Update(msg)`. The keys you handled return early, so the field never
  sees them.
- **Focus.** A text field only takes typing while it is focused. `newModel`
  focuses it, and `Init` returns the `Cmd` that starts the cursor blinking.
- **The size.** `tui.ResizeMsg` arrives at start and whenever the terminal
  changes. `View` uses the width to wrap the help line.
- **Styles.** `ansi.NewStyle()` builds a style once; `Render` applies it.
  Colours are cut down to what the terminal supports.

## 4. Test it without a terminal

`tuitest` runs a model in a real Program against a virtual screen. Put this in
`main_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/tuitest"
)

func TestAddAndFinishATask(t *testing.T) {
	s := tuitest.New(newModel(), 60, 12)
	defer s.Close()
	s.Keys("write", "enter")
	if got := strings.Join(s.Screen(), "\n"); !strings.Contains(got, "[ ] write") {
		t.Fatalf("task not listed:\n%s", got)
	}
	s.Keys("tab")
	if got := strings.Join(s.Screen(), "\n"); !strings.Contains(got, "[x] ") {
		t.Fatalf("task not marked done:\n%s", got)
	}
	s.Keys("ctrl+c")
}
```

`go test` runs it. `Keys` takes key names (`"enter"`, `"tab"`, `"ctrl+c"`) or
literal text typed one character at a time, and returns when the model has
received them and the frame has settled.

## Where next

| To do this | Read |
|---|---|
| Lay out panels side by side | [Layout](layout.md) |
| Pick a widget (lists, tables, dialogs, forms) | [Widgets](widgets.md) |
| Handle mouse, paste and chords | [Input](input.md) |
| Run work in the background and get its result | [Program](program.md) |
| Copy a snippet | [Cookbook](cookbook.md) |
| Test more | [Testing](testing.md) |
| Make it readable by a screen reader | [Accessibility](accessibility.md) |

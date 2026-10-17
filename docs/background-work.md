# Background work

`Update` and `View` run on the event loop's one goroutine and must return
quickly. Anything slow (a request, a directory walk, a search) runs in a Cmd, on
its own goroutine, and comes back as a message.

## One job: tui.Go

```go
func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case startMsg:
		return m, tui.Go(func(ctx context.Context) tui.Msg {
			rows, err := load(ctx)
			return loadedMsg{rows, err}
		})
	case loadedMsg:
		m.rows, m.err = msg.rows, msg.err
	}
	return m, nil
}
```

`ctx` is cancelled when `Run` returns or the context given to `tui.WithContext`
is, so a long job can stop early.

## Jobs you replace: tui.Tasks

Search-as-you-type starts a new job for each key and wants the older one to
stop and its answer ignored. A `tui.Tasks` keys jobs by name:

```go
type model struct{ tasks *tui.Tasks /* ... */ }

// in Update, after the query changed:
cmd := m.tasks.Start("search", func(ctx context.Context) tui.Msg {
	return resultsMsg{query: q, rows: search(ctx, q)}
})

// and when it ends:
case tui.TaskDoneMsg:
	if msg.Key == "search" && !msg.Canceled {
		if r, ok := msg.Msg.(resultsMsg); ok && r.query == m.query {
			m.rows = r.rows
		}
	}
```

Starting a key that is still running cancels the earlier run first, at the call
to `Start`. A cancelled run still delivers a `TaskDoneMsg` with `Canceled` set;
ignore it, and check that the answer is for the query now on screen, since a run
that does not honour its context may finish after being replaced.
`Tasks.Cancel(key)`, `CancelAll` and `Running(key)` complete it. Keep the
`*Tasks` in your model: it is a pointer, shared by copies.

`commandpalette.Source` and `autocomplete.Source` are this pattern built in: give
the widget a function from the typed text to results and pass the
`tui.TaskDoneMsg` it causes back through its `Update`.

## Things that repeat: Every and Tick

`tui.Tick(d, fn)` delivers one message after `d`; `tui.Every(d, fn)` delivers one
every `d` until the cancel func it returns is called. Both are cancelled by the
Program ending.

## Progress and spinners without a Model

In a script, `prompt.Spin(label, fn)` shows a spinner while `fn` runs, and
`prompt.Progress(label, total, fn)` and `prompt.Track(label, items, fn)` show a
bar with the elapsed time, the rate and the time left. Ctrl+C cancels the
function's context.

## Rules

- Never touch the Model from a Cmd. Return a message and change it in `Update`.
- Never block in `Update`. `Program.Send` must not be called from `Update` or
  `View`; return a Cmd instead.
- A Cmd that can run long takes the context and returns when it is done.
- `tui.WithMaxConcurrentCmds` bounds how many Cmds run at once.

## Running a program in a pane

`termpane` runs a command on a pseudo-terminal and draws its screen, so a shell
or an editor can live inside the application (darwin and linux; experimental).
It follows the usual pattern: start it in `Init`, pass every `Msg` to `Update`,
and draw `LayoutNode()`. Tell it the size of its area whenever that changes:

```go
type model struct{ term termpane.Model }

func initial() model {
	t := termpane.New("sh", "-l")
	t.ID = "shell"
	t.Focus()
	return model{term: t}
}

func (m model) Init() tui.Cmd { return m.term.Init() }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.term.SetSize(msg.Width, msg.Height-1) // the room the layout gives it
	case termpane.ExitedMsg:
		return m, tui.Quit() // the shell ended
	}
	var cmd tui.Cmd
	m.term, cmd = m.term.Update(msg) // keys, pastes, output and mouse
	return m, cmd
}
```

Keys go to the program while the pane has focus, except Shift+PgUp, Shift+PgDn
and Shift+End, which scroll back through the lines that left the screen (the
wheel does too, with `Mouse` and `Bounds` set, unless the program asked for
mouse reports). Call `Close` when the application is done with the pane; it ends
the program. The program's output is untrusted: it only changes the pane's
screen, and nothing it writes reaches the host terminal.


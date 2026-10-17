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

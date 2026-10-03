# Running a Program

`tui.NewProgram` wraps a Model in a Program, and `Program.Run` takes over the
terminal, runs the event loop until the model quits, and restores the terminal
before it returns. Without options a Program reads `os.Stdin`, writes
`os.Stdout`, draws on the alternate screen, and has bracketed paste on.

```go
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
```

(From `ExampleNewProgram` in `example_test.go`. With a `strings.Reader` as
input, Run needs no terminal, and reaching EOF quits the Program.)

A Program runs once. A second `Run` returns `ErrProgramReused` without touching
the terminal; build a new Program instead.

## The Model

A Model has three methods:

- `Init() Cmd` runs once, before the first frame.
- `Update(Msg) (Model, Cmd)` handles one message and returns the next model.
- `View() string` returns the screen as text. ANSI styling is fine; cursor
  movement is not, because the renderer positions the cursor.

At startup the Program calls, in order: `SetTheme` (only with `WithTheme` and a
`Themeable` model), `Init`, then `Update` with a `ResizeMsg` holding the
starting size, then draws the first frame. After that, each message goes to
`Update` and the Program redraws if the View changed. See
[rendering.md](rendering.md) for what a redraw does.

The loop doesn't handle Ctrl+C. Raw mode turns off the terminal's signal keys,
so Ctrl+C reaches `Update` as a `Key` with `Type == tui.KeyCtrlC`, and the
model has to return `tui.Quit()` itself. Key decoding is covered in
[input.md](input.md).

A model can implement optional interfaces that the Program looks for:
`CursorPlacer` or `CursorProvider` for the hardware cursor, `Linearizer` for
accessible output ([accessibility.md](accessibility.md)), `Themeable` for
`WithTheme`, and `CellDrawer` for direct cell drawing. A model that wraps
another model can implement `Unwrapper`, and the Program then looks for the
cursor interfaces, `Linearizer` and `Themeable` on the wrapped model too.
Themeable also needs the wrapper to implement `Rewrapper`. `CellDrawer` is
used only on the root model itself.

## Messages the Program sends

| Msg | When |
| --- | --- |
| `ResizeMsg` | Once at startup, on every resize, and again if a capability probe changes how the terminal measures text |
| `Key`, `PasteEvent` | Key presses; a bracketed paste arrives as one `PasteEvent` unless `WithBracketedPaste(false)` |
| `MouseEvent` | Only with `WithMouse` or after `EnableMouse` |
| `FocusEvent` | Only with `WithFocusReporting(true)`; a repeat of the last state is dropped |
| `KeyRepeatMsg`, `KeyReleaseMsg` | Only with `WithKeyboard(... \| KeyboardReportEvents)` |
| `ChordMsg` | Only with `WithChords` |
| `InputErrorMsg` | Once, if reading input fails. Input stops but the Program keeps running |
| `SuspendMsg` | After a `Suspend` Cmd's function has returned |
| `CapabilitiesMsg` | Once, with `WithCapabilityProbe` ([capabilities.md](capabilities.md)) |
| `BackgroundColorEvent`, `PaletteColorEvent`, `BackgroundUnknownMsg` | With `WithBackgroundDetection` or `WithTheme` |
| `ClipboardMsg` | The answer to `ReadClipboard`, if the terminal answers |

A `WithInput` reader that isn't a file quits the Program at `io.EOF`, after the
events read before it. Any other read error is an `InputErrorMsg`.

From outside the loop, `Program.Send` delivers a message as if it came from
the terminal. It is safe from any goroutine. Before Run starts it queues the
message, and after Run returns it drops it. The queue holds 64 messages, and
`Send` blocks while it is full, so don't call `Send` from `Update` or `View`:
use `TrySend`, which never blocks and reports whether the message was queued,
or return a Cmd. `Program.Quit` sends a quit after the messages already queued.

## Commands

A Cmd is a `func() Msg` that the Program runs on its own goroutine. Its result
goes to `Update`. A nil Cmd does nothing.

| Cmd | What it does |
| --- | --- |
| `Quit()` | Ends Run |
| `Batch(cmds...)` | Runs the Cmds concurrently; their messages arrive in no particular order |
| `Sequence(cmds...)` | Runs the Cmds one at a time and delivers their messages in order; stops after a `QuitMsg` |
| `Tick(d, fn)` | Waits `d`, then sends `fn(t)`. The wait ends with no message if Run returns first |
| `Every(d, fn)` | Returns a Cmd that sends `fn(t)` every `d`, plus a cancel func; no tick arrives after cancel returns |
| `FromCtx(fn)`, `Go(fn)` | Run `fn` with `Program.Context`, which is cancelled when Run returns |
| `Println(text)`, `Eprintln(text)` | Write text above the live region, into scrollback (see below) |
| `Suspend(fn)` | Hands the terminal back while `fn` runs (see below) |
| `EnableMouse(mode)`, `EnterAltScreen()`, `ExitAltScreen()`, `ClearScreen()` | Change a terminal mode while running |
| `SetWindowTitle(s)`, `SetCursorShape(s)` | Both are undone on every exit path |
| `ReadClipboard()` | Asks the terminal for the clipboard over OSC 52; many terminals refuse, so no reply may come |
| `Announce(text)`, `AnnounceWith(text, level)` | See [accessibility.md](accessibility.md) |

`WithMaxConcurrentCmds(n)` caps how many Cmds run at once; queued Cmds start
in the order the loop received them. Sequence steps and Every timers aren't
counted against the cap.

A Cmd made by `Tick`, `FromCtx` or a `motion` wait does nothing useful when you
call it directly; it returns an internal message the Program runs. In a test,
run it with `tui.RunCmd`:

```go
cmd := tui.Tick(time.Millisecond, func(time.Time) tui.Msg { return "tick" })
fmt.Println(tui.RunCmd(context.Background(), cmd))
// Output: tick
```

(From `ExampleTick` in `example_test.go`.) `RunCmd` doesn't expand `Batch` or
`Sequence`.

## Options

Every option is a `ProgramOption` passed to `NewProgram`; their godoc has the
details.

| Option | Default |
| --- | --- |
| `WithAltScreen(bool)` | on |
| `WithInput(r)`, `WithInputCloser(c)` | `os.Stdin` |
| `WithOutput(w)`, `WithErrOutput(w)` | `os.Stdout`, `os.Stderr` |
| `WithMouse(mode)` | `MouseOff` |
| `WithBracketedPaste(bool)` | on |
| `WithFocusReporting(bool)` | off |
| `WithKittyKeyboard(bool)`, `WithKeyboard(flags)` | off |
| `WithChords(defs...)`, `WithChordTimeout(d)` | no chords; 500ms timeout |
| `WithEscTimeout(d)` | `input.DefaultEscTimeout` (30ms) |
| `WithMaxFPS(fps)` | 60 fps cap on non-input messages; input and resizes draw at once |
| `WithMaxConcurrentCmds(n)` | no limit |
| `WithContext(ctx)` | `context.Background()` |
| `WithClock(c)` | wall clock, for `Every` |
| `WithExitOnSignal(bool)` | on |
| `WithRecover(bool)` | off |
| `WithSuspendOnCtrlZ(bool)` | off |
| `WithTerminal(t)` | the OS terminal behind input and output |
| `WithResizePoll(d)` | 250ms, Windows only |
| `WithTheme(auto)`, `WithBackgroundDetection(d)` | off; see [theming.md](theming.md) |
| `WithColorProfile(p)`, `WithCapabilityProbe(d)` | detected from the environment; probe off. See [capabilities.md](capabilities.md) |
| `WithAccessible(bool)`, `WithAccessibleAuto()`, `WithAnnounceRegion(rows)`, `WithLinearizeFullLine()`, `WithReducedMotion(bool)` | See [accessibility.md](accessibility.md) |
| `WithCellRenderer(bool)`, `WithBidi(bool)`, `WithFrameLog(w)`, `WithInspector(keys)`, `WithRecorder(w)`, `WithRecorderSidecar(w)` | See [rendering.md](rendering.md) |

Options apply in order, so when two set the same value the later one wins
(`WithChords` adds to the chords instead). Accessible mode is the exception: once
it is on, `NewProgram` turns the alternate screen off and reduced motion on
after every option has run, so no option order can undo that.

## Inline or alternate screen

On the alternate screen (the default) the app gets a full screen of its own, and
the user's shell history comes back when it quits. Under `TERM=dumb`, a
Program given neither `WithAltScreen` nor `WithAccessible` runs in accessible
mode instead, which never uses the alternate screen
([accessibility.md](accessibility.md)). A View taller than the
terminal is cut to the terminal's height.

With `WithAltScreen(false)` the Program draws inline, below the shell prompt.
The live region is as tall as the View. When the View grows taller than the
terminal, the rows that no longer fit are written into scrollback once and are
not redrawn after that. On exit the last frame stays on screen, and the Program
moves to a new line so the prompt starts below it.

`Println` and `Eprintln` write lines above the live region, into scrollback,
and redraw the live region below them. On the alternate screen there is no
scrollback, so the Program holds those lines (up to 10,000; the oldest are
dropped beyond that) and writes them once it leaves the alternate screen, with
`ExitAltScreen` or on exit. `examples/inlinebuild` is an inline program that
commits finished lines with `Println`.

## Suspend and resume

`Suspend(fn)` undoes the terminal setup (modes off, raw mode off), stops
reading input so `fn` owns stdin, runs `fn`, then puts everything back and
repaints the whole frame. Use it to run an editor or a shell. `fn` starts and
waits for the external program itself. Its error reaches `Update` as
`SuspendMsg.Err`, and the Program keeps running either way.

`WithSuspendOnCtrlZ(true)` makes Ctrl+Z stop the process as a shell job does.
The Program restores the terminal, stops itself with SIGSTOP, and on `fg`
re-enters raw mode and every mode, re-reads the size (sending a `ResizeMsg` if
it changed), and repaints. `Update` sees neither the Ctrl+Z nor a `SuspendMsg`.
Without the option, or on Windows, Ctrl+Z is an ordinary `Key`.

## Signals, panics and terminal restore

Run restores the terminal on every return path: alternate screen, cursor,
mouse, bracketed paste, keyboard and focus modes, the window title and cursor
shape, and raw mode.

When a write to the output fails, Run returns an error that wraps the write
error. A Program does not go on rendering to a writer that is gone, such as a
closed remote connection.

SIGTERM and SIGHUP terminate a Go process without running deferred functions,
so Run catches them. It restores the terminal and cancels `Program.Context`,
then by default exits with status 1. With `WithExitOnSignal(false)`, Run
instead returns `ErrInterrupted`, so the app can run its own shutdown.

A panic in `Init`, `Update` or `View` restores the terminal and then continues
as a panic. A panic in a Cmd's goroutine restores the terminal (waiting up to
500ms for the loop to do it, then writing a fixed restore sequence) and
re-panics with the same value. With `WithRecover(true)`, Run returns a
`*PanicError` holding the value and stack for a panic in `Init`, `Update`,
`View` or any Cmd: a plain Cmd, a `Tick` or `FromCtx` Cmd, a `Sequence` step
or an `Every` callback (`recover_test.go` covers each).

## Context and cancellation

`Program.Context` is valid as soon as `NewProgram` returns, so you can pass it
to the model before Run. Run cancels it on every return path, as does
cancelling the parent given to `WithContext`. Cancelling that parent also ends
Run: it restores the terminal and returns the parent's error
(`context.Canceled` or `context.DeadlineExceeded`), so a host can stop a
session without the model's help. Cmds made by `Tick`, `FromCtx`
and `Go` receive this context. A Cmd that does I/O can use the context to stop
early instead of outliving Run.

## Other terminals

`WithTerminal` replaces the terminal the Program controls (size, raw mode,
output VT processing) with your own `Terminal`. Use it with `WithInput` and
`WithOutput` for a remote session such as SSH. A `Terminal` that also
implements `ResizeNotifier` delivers resizes through the same port. The
headless test harness is described in [testing.md](testing.md).
